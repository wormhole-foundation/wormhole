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
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	stellarxdr "github.com/stellar/go/xdr"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

var (
	stellarConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_stellar_connection_errors_total",
			Help: "Total number of Stellar connection errors",
		}, []string{"network", "reason"})

	stellarMessagesObserved = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_stellar_messages_observed_total",
			Help: "Total number of Stellar messages observed (pre-confirmation)",
		}, []string{"network"})

	stellarMessagesConfirmed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_stellar_messages_confirmed_total",
			Help: "Total number of Stellar messages confirmed (post-publish)",
		}, []string{"network"})

	currentStellarLedger = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_stellar_current_ledger",
			Help: "Current Stellar ledger sequence number",
		}, []string{"network"})
)

type WatcherConfig struct {
	NetworkID    string
	NetworkName  string // human-readable name for logging and metrics labels
	ChainID      vaa.ChainID
	Rpc          string // Soroban RPC HTTP endpoint
	Contract     string // Core contract id
	PollInterval time.Duration
	ReadTimeout  time.Duration
	StartLedger  uint64
	MaxPerPoll   int
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

	networkName := wc.NetworkName
	if networkName == "" {
		networkName = wc.NetworkID
	}

	w := NewWatcher(
		wc.Rpc,
		wc.Contract,
		networkName,
		wc.ChainID,
		wc.StartLedger,
		wc.PollInterval,
		wc.ReadTimeout,
		wc.MaxPerPoll,
		msgC,
		obsvReqC,
		env,
	)
	return w.Run, w, nil
}

func (wc *WatcherConfig) GetChainID() vaa.ChainID {
	return wc.ChainID
}

func (wc *WatcherConfig) GetNetworkID() watchers.NetworkID {
	return watchers.NetworkID(wc.NetworkID)
}

type watcher struct {
	rpc           string
	contract      string
	networkName   string
	chainID       vaa.ChainID
	nextLedger    uint64
	pollInterval  time.Duration
	httpTimeout   time.Duration
	maxPerPoll    int
	msgC          chan<- *common.MessagePublication
	obsvReqC      <-chan *gossipv1.ObservationRequest
	env           common.Environment
	httpClient    *http.Client
	readinessSync readiness.Component
	logger        *zap.Logger
}

func NewWatcher(
	rpc string,
	contract string,
	networkName string,
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
		rpc:           rpc,
		contract:      contract,
		networkName:   networkName,
		chainID:       chainID,
		nextLedger:    startLedger,
		pollInterval:  pollInterval,
		httpTimeout:   readTimeout,
		maxPerPoll:    maxPerPoll,
		msgC:          msgC,
		obsvReqC:      obsvReqC,
		env:           env,
		httpClient:    &http.Client{Timeout: readTimeout},
		readinessSync: common.MustConvertChainIdToReadinessSyncing(chainID),
	}
}

func (w *watcher) Run(ctx context.Context) error {
	logger := supervisor.Logger(ctx).With(
		zap.String("component", "stellar_watcher"),
		zap.String("rpc", w.rpc),
		zap.String("contract", w.contract),
		zap.String("chain", w.chainID.String()),
	)
	w.logger = logger

	if w.nextLedger == 0 {
		seq, err := w.getInitialLedger(ctx, logger)
		if err != nil {
			return err
		}
		w.nextLedger = seq
		logger.Info("initialized start ledger", zap.Uint64("ledger", w.nextLedger))
	}

	p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: w.contract,
	})

	errC := make(chan error)
	common.RunWithScissors(ctx, errC, "stellar_reobservation", func(ctx context.Context) error {
		return w.runReobservationHandler(ctx)
	})

	t := time.NewTicker(w.pollInterval)
	defer t.Stop()
	logger.Info("stellar watcher started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("stellar watcher stopping")
			return nil
		case err := <-errC:
			return err
		case <-t.C:
			if _, err := w.pollOnce(ctx, logger); err != nil {
				stellarConnectionErrors.WithLabelValues(w.networkName, "poll").Inc()
				p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
				logger.Warn("pollOnce error", zap.Error(err))
			}
		}
	}
}

// getInitialLedger fetches the starting ledger, retrying with capped exponential
// backoff. A transient RPC outage at startup (e.g. the Soroban RPC not yet
// reachable when the guardian boots) must not kill the watcher, mirroring the
// resilience of the poll loop. Returns an error only when the context is cancelled.
func (w *watcher) getInitialLedger(ctx context.Context, logger *zap.Logger) (uint64, error) {
	backoff := w.pollInterval
	if backoff <= 0 {
		backoff = 700 * time.Millisecond
	}
	const maxBackoff = 60 * time.Second

	for {
		seq, err := w.getLatestLedger(ctx)
		if err == nil {
			return seq, nil
		}

		stellarConnectionErrors.WithLabelValues(w.networkName, "initial_ledger").Inc()
		p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
		logger.Warn("failed to get latest ledger, retrying", zap.Error(err), zap.Duration("backoff", backoff))

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(backoff):
		}

		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// runReobservationHandler handles incoming reobservation requests. Returns an error only on fatal failure.
func (w *watcher) runReobservationHandler(ctx context.Context) error {
	logger := w.logger
	for {
		select {
		case <-ctx.Done():
			return nil
		case req := <-w.obsvReqC:
			if vaa.ChainID(req.ChainId) != w.chainID {
				logger.Debug("ignoring reobservation request for different chain",
					zap.Uint32("requestChainId", req.ChainId),
					zap.String("watcherChainId", w.chainID.String()),
				)
				continue
			}

			txHash := hex.EncodeToString(req.TxHash)
			logger.Info("received reobservation request", zap.String("txHash", txHash))

			count, err := w.handleReobservationRequest(ctx, txHash, w.rpc, w.httpClient)
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

// Reobserve implements the Reobserver interface, allowing reobservation via a custom RPC endpoint.
func (w *watcher) Reobserve(ctx context.Context, chainID vaa.ChainID, txID []byte, customEndpoint string) (uint32, error) {
	if chainID != w.chainID {
		return 0, fmt.Errorf("unexpected chain id: %v", chainID)
	}
	txHash := hex.EncodeToString(txID)
	w.logger.Info("received request to reobserve using custom endpoint",
		zap.Stringer("chainID", chainID),
		zap.String("txHash", txHash),
		zap.String("endpoint", customEndpoint),
	)
	httpClient := &http.Client{Timeout: w.httpTimeout}
	return w.handleReobservationRequest(ctx, txHash, customEndpoint, httpClient)
}

// handleReobservationRequest fetches events for a specific transaction using getTransaction
// to get the accurate ledger timestamp, then queries events for that ledger.
func (w *watcher) handleReobservationRequest(ctx context.Context, txHash, rpcURL string, httpClient *http.Client) (uint32, error) {
	logger := w.logger

	// Use getTransaction to get the ledger sequence and timestamp for this transaction.
	txRes, err := rpcCall(ctx, "getTransaction", map[string]any{"hash": txHash}, rpcURL, httpClient)
	if err != nil {
		return 0, fmt.Errorf("getTransaction failed: %w", err)
	}

	status := gjson.GetBytes(*txRes, "status").Str
	switch status {
	case "NOT_FOUND":
		return 0, fmt.Errorf("transaction %s not found (may be outside retention window)", txHash)
	case "FAILED":
		return 0, fmt.Errorf("transaction %s failed on-chain", txHash)
	case "SUCCESS":
		// continue below
	default:
		return 0, fmt.Errorf("unexpected transaction status %q for %s", status, txHash)
	}

	ledger := gjson.GetBytes(*txRes, "ledger").Uint()
	createdAt := gjson.GetBytes(*txRes, "createdAt").Int()
	timestamp := time.Unix(createdAt, 0).UTC()

	txIDBytes, err := hex.DecodeString(txHash)
	if err != nil {
		return 0, fmt.Errorf("failed to decode txHash: %w", err)
	}

	// Query events starting from the transaction's ledger.
	params := map[string]any{
		"startLedger": ledger,
		"filters": []map[string]any{
			{"type": "contract", "contractIds": []string{w.contract}},
		},
		"pagination": map[string]any{"limit": w.maxPerPoll},
	}
	res, err := rpcCall(ctx, "getEvents", params, rpcURL, httpClient)
	if err != nil {
		return 0, fmt.Errorf("getEvents failed: %w", err)
	}

	events := gjson.GetBytes(*res, "events")
	if !events.Exists() {
		return 0, nil
	}

	var messagesFound uint32
	for _, e := range events.Array() {
		if e.Get("txHash").Str != txHash {
			continue
		}

		mp := w.parseEventJSON(e, logger)
		if mp == nil {
			continue
		}

		mp.TxID = txIDBytes
		mp.Timestamp = timestamp
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

// rpcCall executes a JSON-RPC request against the given URL using the provided HTTP client.
func rpcCall(ctx context.Context, method string, params any, rpcURL string, httpClient *http.Client) (*json.RawMessage, error) {
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	body, _ := json.Marshal(&req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
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

func (w *watcher) call(ctx context.Context, method string, params any) (*json.RawMessage, error) {
	return rpcCall(ctx, method, params, w.rpc, w.httpClient)
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
	advanced := false
	cursor := ""

	for {
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
		if cursor == "" {
			params["startLedger"] = w.nextLedger
		} else {
			params["pagination"].(map[string]any)["cursor"] = cursor
		}

		res, err := w.call(ctx, "getEvents", params)
		if err != nil {
			return false, err
		}

		// getEvents returns latestLedger in the response; use it to avoid an extra RPC call.
		latestLedger := gjson.GetBytes(*res, "latestLedger").Uint()
		if latestLedger > 0 {
			currentStellarLedger.WithLabelValues(w.networkName).Set(float64(latestLedger))
			p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
				Height:          int64(latestLedger), // #nosec G115 -- ledger numbers are well within int64 range for the foreseeable future
				ContractAddress: w.contract,
			})
		}
		readiness.SetReady(w.readinessSync)

		events := gjson.GetBytes(*res, "events").Array()

		for _, e := range events {
			cursor = e.Get("id").Str

			ledger := e.Get("ledger").Uint()
			txHash := e.Get("txHash").Str

			mp := w.parseEventJSON(e, logger)
			if mp == nil {
				continue
			}

			txIDBytes, err := hex.DecodeString(txHash)
			if err != nil {
				logger.Warn("failed to decode txHash", zap.String("txHash", txHash), zap.Error(err))
				continue
			}
			mp.TxID = txIDBytes
			mp.EmitterChain = w.chainID

			// Use ledgerClosedAt from the event for a deterministic timestamp.
			// All guardians observing the same event will use the same timestamp,
			// ensuring identical VAAs are produced.
			closedAt := e.Get("ledgerClosedAt").Str
			ts, err := time.Parse(time.RFC3339, closedAt)
			if err != nil {
				logger.Warn("failed to parse ledgerClosedAt, skipping event",
					zap.String("ledgerClosedAt", closedAt),
					zap.Error(err),
				)
				continue
			}
			mp.Timestamp = ts

			stellarMessagesObserved.WithLabelValues(w.networkName).Inc()

			logger.Info("stellar message published",
				zap.Uint64("ledger", ledger),
				zap.String("tx", txHash),
				zap.Uint64("seq", mp.Sequence),
				zap.Uint8("consistency", mp.ConsistencyLevel),
			)

			select {
			case w.msgC <- mp:
				stellarMessagesConfirmed.WithLabelValues(w.networkName).Inc()
			case <-ctx.Done():
				return advanced, ctx.Err()
			}

			if ledger >= w.nextLedger {
				w.nextLedger = ledger + 1
				advanced = true
			}
		}

		if len(events) < w.maxPerPoll {
			// Received fewer events than the limit — no more pages.
			if !advanced && latestLedger > w.nextLedger {
				w.nextLedger = latestLedger
				advanced = true
			}
			break
		}
		// Received exactly maxPerPoll events — there may be more pages; continue with cursor.
	}

	return advanced, nil
}

// parseEventJSON checks the event topics for a message_published event and parses the event data.
// Returns nil if the event is not a message_published event or cannot be parsed.
func (w *watcher) parseEventJSON(e gjson.Result, logger *zap.Logger) *common.MessagePublication {
	topics := e.Get("topic").Array()
	if len(topics) < 2 {
		return nil
	}

	eventNameBytes, err := base64.StdEncoding.DecodeString(topics[1].Str)
	if err != nil {
		return nil
	}
	if !bytes.Contains(eventNameBytes, []byte("message_published")) {
		return nil
	}

	valueBytes, err := base64.StdEncoding.DecodeString(e.Get("value").Str)
	if err != nil {
		logger.Debug("failed to decode event value", zap.Error(err))
		return nil
	}

	return parseMessageFromXDR(valueBytes, logger)
}

// parseMessageFromXDR parses the XDR-encoded Soroban event value into a MessagePublication.
func parseMessageFromXDR(data []byte, logger *zap.Logger) *common.MessagePublication {
	var scVal stellarxdr.ScVal

	_, err := stellarxdr.Unmarshal(bytes.NewReader(data), &scVal)
	if err != nil {
		logger.Debug("failed to unmarshal XDR", zap.Error(err))
		return nil
	}

	eventMap, ok := scVal.GetMap()
	if !ok {
		logger.Debug("event value is not a map")
		return nil
	}

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

		switch string(keySymbol) {
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

	if len(emitterAddress) == 0 {
		logger.Warn("message_published event has empty emitter address, skipping")
		return nil
	}

	var emitter vaa.Address
	if len(emitterAddress) >= 32 {
		copy(emitter[:], emitterAddress[:32])
	} else {
		copy(emitter[:], emitterAddress)
	}

	return &common.MessagePublication{
		Nonce:            nonce,
		Sequence:         sequence,
		ConsistencyLevel: uint8(consistencyLevel),
		EmitterAddress:   emitter,
		Payload:          payload,
		// TxID, Timestamp, and EmitterChain are set by the caller.
	}
}
