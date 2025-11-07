package stellar

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	stellarxdr "github.com/stellar/go/xdr"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type WatcherConfig struct {
	NetworkID    string
	ChainID      vaa.ChainID
	Rpc          string // Soroban RPC HTTP endpoint
	Contract     string // Core contract id
	PollInterval time.Duration
	ReadTimeout  time.Duration
	StartLedger  uint64
	MaxPerPoll   int
	RequestLimit int64
}

func (wc *WatcherConfig) Create(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	ccqReqC <-chan *query.PerChainQueryInternal,
	ccqRespC chan<- *query.PerChainQueryResponseInternal,
	guardianSetC chan<- *common.GuardianSet,
	env common.Environment,
) (supervisor.Runnable, interfaces.Reobserver, error) {
	_ = ccqReqC
	_ = ccqRespC
	_ = guardianSetC

	if wc.PollInterval == 0 {
		wc.PollInterval = 700 * time.Millisecond
	}
	if wc.ReadTimeout == 0 {
		wc.ReadTimeout = 10 * time.Second
	}
	if wc.MaxPerPoll <= 0 {
		wc.MaxPerPoll = 128
	}

	w := NewWatcher(
		wc.Rpc,
		wc.Contract,
		wc.ChainID,
		wc.StartLedger,
		wc.PollInterval,
		wc.ReadTimeout,
		wc.MaxPerPoll,
		msgC,
		obsvReqC,
		env,
	)
	return w.Run, nil, nil
}

func (wc *WatcherConfig) GetChainID() vaa.ChainID {
	return wc.ChainID
}

func (wc *WatcherConfig) GetNetworkID() watchers.NetworkID {
	return watchers.NetworkID(wc.NetworkID)
}

type watcher struct {
	rpc          string
	contract     string
	chainID      vaa.ChainID
	nextLedger   uint64
	pollInterval time.Duration
	httpTimeout  time.Duration
	maxPerPoll   int
	msgC         chan<- *common.MessagePublication
	obsvReqC     <-chan *gossipv1.ObservationRequest
	env          common.Environment
	httpClient   *http.Client
}

func NewWatcher(
	rpc string,
	contract string,
	chainID vaa.ChainID,
	startLedger uint64,
	pollInterval time.Duration,
	readTimeout time.Duration,
	maxPerPoll int,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	env common.Environment,
) *watcher {
	return &watcher{
		rpc:          rpc,
		contract:     contract,
		chainID:      chainID,
		nextLedger:   startLedger,
		pollInterval: pollInterval,
		httpTimeout:  readTimeout,
		maxPerPoll:   maxPerPoll,
		msgC:         msgC,
		obsvReqC:     obsvReqC,
		env:          env,
		httpClient:   &http.Client{Timeout: readTimeout},
	}
}

func (w *watcher) Run(ctx context.Context) error {
	logger := supervisor.Logger(ctx).With(
		zap.String("component", "stellar_watcher"),
		zap.String("rpc", w.rpc),
		zap.String("contract", w.contract),
		zap.String("chain", w.chainID.String()),
	)

	if w.nextLedger == 0 {
		seq, err := w.getLatestLedger(ctx)
		if err != nil {
			logger.Error("failed to get latest ledger", zap.Error(err))
			return err
		}
		w.nextLedger = seq
		logger.Info("initialized start ledger", zap.Uint64("ledger", w.nextLedger))
	}

	// Start reobservation request handler goroutine
	go w.runReobservationHandler(ctx, logger)

	t := time.NewTicker(w.pollInterval)
	defer t.Stop()
	logger.Info("stellar watcher started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("stellar watcher stopping")
			return nil

		case <-t.C:
			if _, err := w.pollOnce(ctx, logger); err != nil {
				logger.Warn("pollOnce error", zap.Error(err))
				continue
			}
		}
	}
}

// runReobservationHandler handles incoming reobservation requests
func (w *watcher) runReobservationHandler(ctx context.Context, logger *zap.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-w.obsvReqC:
			if vaa.ChainID(req.ChainId) != w.chainID {
				logger.Debug("ignoring reobservation request for different chain",
					zap.Uint32("requestChainId", req.ChainId),
					zap.String("watcherChainId", w.chainID.String()),
				)
				continue
			}

			txHash := hex.EncodeToString(req.TxHash)
			logger.Info("received reobservation request",
				zap.String("txHash", txHash),
			)

			count, err := w.handleReobservationRequest(ctx, logger, txHash)
			if err != nil {
				logger.Error("failed to handle reobservation request",
					zap.String("txHash", txHash),
					zap.Error(err),
				)
				continue
			}

			logger.Info("completed reobservation request",
				zap.String("txHash", txHash),
				zap.Uint32("messagesFound", count),
			)
		}
	}
}

// handleReobservationRequest processes a reobservation request by fetching events for a specific transaction
func (w *watcher) handleReobservationRequest(ctx context.Context, logger *zap.Logger, txHash string) (uint32, error) {
	// Query events for this specific transaction
	params := map[string]any{
		"filters": []map[string]any{
			{
				"type":        "contract",
				"contractIds": []string{w.contract},
			},
		},
		"pagination": map[string]any{
			"limit": w.maxPerPoll,
		},
	}

	res, err := w.call(ctx, "getEvents", params)
	if err != nil {
		return 0, fmt.Errorf("failed to query events: %w", err)
	}

	events := gjson.GetBytes(*res, "events")
	if !events.Exists() {
		return 0, nil
	}

	var messagesFound uint32
	now := time.Now().UTC()

	for _, e := range events.Array() {
		eventTxHash := e.Get("txHash").Str

		// Skip events that don't match the requested transaction
		if eventTxHash != txHash {
			continue
		}

		ledger := e.Get("ledger").Uint()

		// Check topics for message_published event
		topics := e.Get("topic").Array()
		if len(topics) < 2 {
			continue
		}

		// Decode the second topic to check event name
		eventNameB64 := topics[1].Str
		eventNameBytes, err := base64.StdEncoding.DecodeString(eventNameB64)
		if err != nil {
			continue
		}

		if !bytes.Contains(eventNameBytes, []byte("message_published")) {
			continue
		}

		// Decode the value field which contains the event data
		valueB64 := e.Get("value").Str
		valueBytes, err := base64.StdEncoding.DecodeString(valueB64)
		if err != nil {
			logger.Debug("failed to decode event value", zap.Error(err))
			continue
		}

		// Parse the XDR-encoded event data
		mp := parseMessageFromXDR(valueBytes, logger)
		if mp == nil {
			logger.Debug("failed to parse message from XDR")
			continue
		}

		mp.Timestamp = now
		mp.EmitterChain = w.chainID
		mp.IsReobservation = true

		logger.Info("reobserved stellar message",
			zap.Uint64("ledger", ledger),
			zap.String("tx", txHash),
			zap.Uint64("seq", mp.Sequence),
			zap.Uint8("consistency", mp.ConsistencyLevel),
		)

		select {
		case w.msgC <- mp:
			messagesFound++
		case <-ctx.Done():
			return messagesFound, ctx.Err()
		}
	}

	return messagesFound, nil
}

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      int         `json:"id,omitempty"`
}

type rpcResponse struct {
	JSONRPC string           `json:"jsonrpc,omitempty"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	ID int `json:"id,omitempty"`
}

func (w *watcher) call(ctx context.Context, method string, params any) (*json.RawMessage, error) {
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	body, _ := json.Marshal(&req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, w.rpc, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rr rpcResponse
	if err := json.Unmarshal(b, &rr); err != nil {
		return nil, fmt.Errorf("jsonrpc decode: %w", err)
	}
	if rr.Error != nil {
		return nil, fmt.Errorf("jsonrpc error %d: %s", rr.Error.Code, rr.Error.Message)
	}
	return rr.Result, nil
}

func (w *watcher) getLatestLedger(ctx context.Context) (uint64, error) {
	res, err := w.call(ctx, "getLatestLedger", nil)
	if err != nil {
		return 0, err
	}
	seq := gjson.GetBytes(*res, "sequence").Uint()
	return seq, nil
}

func (w *watcher) pollOnce(ctx context.Context, logger *zap.Logger) (bool, error) {
	params := map[string]any{
		"startLedger": w.nextLedger,
		"filters": []map[string]any{
			{
				"type":        "contract",
				"contractIds": []string{w.contract},
			},
		},
		"pagination": map[string]any{
			"limit": w.maxPerPoll,
		},
	}

	res, err := w.call(ctx, "getEvents", params)
	if err != nil {
		return false, err
	}

	events := gjson.GetBytes(*res, "events")
	if !events.Exists() || len(events.Array()) == 0 {
		latest, err := w.getLatestLedger(ctx)
		if err == nil && latest > w.nextLedger {
			w.nextLedger = latest
			return true, nil
		}
		return false, nil
	}

	advanced := false
	now := time.Now().UTC()

	for _, e := range events.Array() {
		ledger := e.Get("ledger").Uint()
		txHash := e.Get("txHash").Str

		// Check topics for message_published event
		topics := e.Get("topic").Array()
		if len(topics) < 2 {
			continue
		}

		// Decode the second topic to check event name
		eventNameB64 := topics[1].Str
		eventNameBytes, err := base64.StdEncoding.DecodeString(eventNameB64)
		if err != nil {
			continue
		}
		// Simple check: look for "message_published" in the decoded bytes
		if !bytes.Contains(eventNameBytes, []byte("message_published")) {
			continue
		}

		// Decode the value field which contains the event data
		valueB64 := e.Get("value").Str
		valueBytes, err := base64.StdEncoding.DecodeString(valueB64)
		if err != nil {
			logger.Debug("failed to decode event value", zap.Error(err))
			continue
		}

		// Parse the XDR-encoded event data
		mp := parseMessageFromXDR(valueBytes, logger)
		if mp == nil {
			logger.Debug("failed to parse message from XDR")
			continue
		}

		mp.Timestamp = now
		mp.EmitterChain = w.chainID

		logger.Info("stellar message published",
			zap.Uint64("ledger", ledger),
			zap.String("tx", txHash),
			zap.Uint64("seq", mp.Sequence),
			zap.Uint8("consistency", mp.ConsistencyLevel),
		)

		select {
		case w.msgC <- mp:
		case <-ctx.Done():
			return advanced, ctx.Err()
		}

		if ledger >= w.nextLedger {
			w.nextLedger = ledger + 1
			advanced = true
		}
	}
	return advanced, nil
}

// parseMessageFromXDR parses the XDR-encoded Soroban event into a MessagePublication
func parseMessageFromXDR(data []byte, logger *zap.Logger) *common.MessagePublication {
	var scVal stellarxdr.ScVal

	// Decode XDR
	_, err := stellarxdr.Unmarshal(bytes.NewReader(data), &scVal)
	if err != nil {
		logger.Debug("failed to unmarshal XDR", zap.Error(err))
		return nil
	}

	// The event value should be a Map
	eventMap, ok := scVal.GetMap()
	if !ok {
		logger.Debug("event value is not a map")
		return nil
	}

	// Extract fields from the map
	var nonce uint32
	var sequence uint64
	var emitterAddress []byte
	var payload []byte
	var consistencyLevel uint32

	for _, entry := range *eventMap {
		keySymbol, ok := entry.Key.GetSym()
		if !ok {
			continue
		}

		keyStr := string(keySymbol)

		switch keyStr {
		case "nonce":
			if val, ok := entry.Val.GetU32(); ok {
				nonce = uint32(val)
			}
		case "sequence":
			if val, ok := entry.Val.GetU64(); ok {
				sequence = uint64(val)
			}
		case "emitter_address":
			if val, ok := entry.Val.GetBytes(); ok {
				emitterAddress = val
			}
		case "payload":
			if val, ok := entry.Val.GetBytes(); ok {
				payload = val
			}
		case "consistency_level":
			if val, ok := entry.Val.GetU32(); ok {
				consistencyLevel = uint32(val)
			}
		}
	}

	// Convert emitter address to vaa.Address (32 bytes)
	var emitter vaa.Address
	if len(emitterAddress) > 0 {
		if len(emitterAddress) >= 32 {
			copy(emitter[:], emitterAddress[:32])
		} else {
			copy(emitter[:], emitterAddress)
		}
	}

	return &common.MessagePublication{
		Nonce:            nonce,
		Sequence:         sequence,
		ConsistencyLevel: uint8(consistencyLevel),
		EmitterAddress:   emitter,
		Payload:          payload,
		// Timestamp and EmitterChain are set by the caller
	}
}
