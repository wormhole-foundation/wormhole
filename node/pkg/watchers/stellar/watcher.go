package stellar

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
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

const (
	defaultPollInterval = 700 * time.Millisecond
	defaultReadTimeout  = 10 * time.Second
	eventTopicCount     = 2
	messageFieldCount   = 5
	stellarAddressLen   = 32
	txHashLen           = 32

	// Base64 XDR of the event topics Symbol("wormhole") and Symbol("message_published").
	topicWormhole         = "AAAADwAAAAh3b3JtaG9sZQ=="
	topicMessagePublished = "AAAADwAAABFtZXNzYWdlX3B1Ymxpc2hlZAAAAA=="
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

	stellarMessageParseFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_stellar_message_parse_failures_total",
			Help: "Total number of Stellar message_published events from the core contract that failed to parse",
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
	Rpc          string //nolint:staticcheck // Match the existing watcher config field name convention.
	Contract     string // Core contract id
	PollInterval time.Duration
	ReadTimeout  time.Duration
	StartLedger  uint64
	MaxPerPoll   int
}

//nolint:unparam // The error return is required by the watcher config interface.
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
		wc.PollInterval = defaultPollInterval
	}
	if wc.ReadTimeout == 0 {
		wc.ReadTimeout = defaultReadTimeout
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
		seq, err := w.getLatestLedger(ctx)
		if err != nil {
			stellarConnectionErrors.WithLabelValues(w.networkName, "initial_ledger").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("getLatestLedger failed: %w", err)
		}
		w.nextLedger = seq
		logger.Info("initialized start ledger", zap.Uint64("ledger", w.nextLedger))
	}

	p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: w.contract,
	})

	// Scissors sends without blocking, so the buffer keeps the error until Run reads it.
	errC := make(chan error, 1)
	common.RunWithScissors(ctx, errC, "stellar_reobservation", func(ctx context.Context) error {
		return w.runReobservationHandler(ctx)
	})

	t := time.NewTicker(w.pollInterval)
	defer t.Stop()
	supervisor.Signal(ctx, supervisor.SignalHealthy)
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

// runReobservationHandler serves reobservation requests until ctx is canceled. It panics on a
// request for another chain.
func (w *watcher) runReobservationHandler(ctx context.Context) error {
	logger := w.logger
	for {
		select {
		case <-ctx.Done():
			return nil
		case req := <-w.obsvReqC:
			// node/pkg/node/reobserve.go sends only this chain's requests here, so a mismatch
			// is a bug in the node's channel setup.
			if req.ChainId > math.MaxUint16 || vaa.ChainID(req.ChainId) != w.chainID {
				panic("invalid chain ID")
			}

			txHash := hex.EncodeToString(req.TxHash)
			logger.Info("received reobservation request", zap.String("txHash", txHash))

			count, err := w.handleReobservationRequest(ctx, txHash, w.rpc, w.httpClient)
			if err != nil {
				p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
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

// handleReobservationRequest finds the ledger of a specific transaction using getTransaction,
// then queries events for that ledger.
func (w *watcher) handleReobservationRequest(ctx context.Context, txHash, rpcURL string, httpClient *http.Client) (uint32, error) {
	logger := w.logger

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

	// A tx's events are all in its own ledger, so page from that ledger until a page ends past it.
	pagination := map[string]any{"limit": w.maxPerPoll}
	params := map[string]any{
		"startLedger": ledger,
		"filters": []map[string]any{
			{"type": "contract", "contractIds": []string{w.contract}},
		},
		"pagination": pagination,
	}
	var events []gjson.Result
	for {
		res, err := rpcCall(ctx, "getEvents", params, rpcURL, httpClient)
		if err != nil {
			return 0, fmt.Errorf("getEvents failed: %w", err)
		}
		page := gjson.GetBytes(*res, "events").Array()
		events = append(events, page...)
		if len(page) < w.maxPerPoll || page[len(page)-1].Get("ledger").Uint() > ledger {
			break
		}
		delete(params, "startLedger") // getEvents rejects startLedger together with a cursor
		pagination["cursor"] = page[len(page)-1].Get("id").Str
	}

	var messagesFound uint32
	for _, e := range events {
		if e.Get("txHash").Str != txHash {
			continue
		}

		mp := w.messageFromEvent(e, true)
		if mp == nil {
			continue
		}

		logger.Info("reobserved stellar message", mp.ZapFields(zap.Uint64("ledger", ledger))...)

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

	b, err := common.SafeRead(resp.Body)
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
	if rr.Result == nil {
		return nil, fmt.Errorf("jsonrpc: missing result")
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
	var lastLedger uint64

	for {
		pagination := map[string]any{
			"limit": w.maxPerPoll,
		}
		params := map[string]any{
			"filters": []map[string]any{
				{
					"type":        "contract",
					"contractIds": []string{w.contract},
				},
			},
			"pagination": pagination,
		}
		if cursor == "" {
			params["startLedger"] = w.nextLedger
		} else {
			pagination["cursor"] = cursor
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
			lastLedger = ledger

			mp := w.messageFromEvent(e, false)
			if mp == nil {
				continue
			}

			stellarMessagesObserved.WithLabelValues(w.networkName).Inc()

			logger.Info("stellar message published", mp.ZapFields(zap.Uint64("ledger", ledger))...)

			select {
			case w.msgC <- mp:
				stellarMessagesConfirmed.WithLabelValues(w.networkName).Inc()
			case <-ctx.Done():
				return advanced, ctx.Err()
			}
		}

		if len(events) < w.maxPerPoll {
			// Received fewer events than the limit — no more pages.
			if lastLedger > 0 {
				w.nextLedger = lastLedger + 1
				advanced = true
			} else if latestLedger > w.nextLedger {
				w.nextLedger = latestLedger
				advanced = true
			}
			break
		}
		// Received exactly maxPerPoll events — there may be more pages; continue with cursor.
		// The next page may continue lastLedger, so if it fails the next poll restarts at lastLedger.
		w.nextLedger = lastLedger
	}

	return advanced, nil
}

// parsedMessage holds the message fields parsed from a message_published event value.
// The caller combines these with the transaction context (TxID, timestamp, chain ID)
// to build a complete MessagePublication.
type parsedMessage struct {
	nonce            uint32
	sequence         uint64
	consistencyLevel uint8
	emitterAddress   vaa.Address
	payload          []byte
}

// messageFromEvent returns the MessagePublication for a message_published event from the core
// contract, or nil for any other event. It logs, counts, and skips events that fail to parse.
func (w *watcher) messageFromEvent(e gjson.Result, isReobservation bool) *common.MessagePublication {
	// Match loosely, so an event reshaped by a contract upgrade is a parse failure, not a skip.
	topics := e.Get("topic").Array()
	if e.Get("contractId").Str != w.contract || len(topics) < eventTopicCount || topics[1].Str != topicMessagePublished {
		return nil
	}

	mp, err := w.parseMessagePublished(e, isReobservation)
	if err != nil {
		stellarMessageParseFailures.WithLabelValues(w.networkName).Inc()
		p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
		w.logger.Warn("skipping invalid message_published event", zap.String("id", e.Get("id").Str), zap.Error(err))
		return nil
	}
	return mp
}

// parseMessagePublished builds the MessagePublication for a message_published event.
func (w *watcher) parseMessagePublished(e gjson.Result, isReobservation bool) (*common.MessagePublication, error) {
	if topics := e.Get("topic").Array(); e.Get("type").Str != "contract" || len(topics) != eventTopicCount || topics[0].Str != topicWormhole {
		return nil, fmt.Errorf("unexpected event type %q or topics %s", e.Get("type").Str, e.Get("topic").Raw)
	}

	txID, err := hex.DecodeString(e.Get("txHash").Str)
	if err != nil || len(txID) != txHashLen {
		return nil, fmt.Errorf("invalid txHash %q", e.Get("txHash").Str)
	}

	// ledgerClosedAt is the same for every guardian, so they all produce the same VAA.
	ts, err := time.Parse(time.RFC3339, e.Get("ledgerClosedAt").Str)
	if err != nil {
		return nil, fmt.Errorf("invalid ledgerClosedAt: %w", err)
	}
	// The VAA holds uint32 seconds, and the epoch itself means an unset close time.
	if ts.Unix() <= 0 || ts.Unix() > math.MaxUint32 {
		return nil, fmt.Errorf("ledgerClosedAt %q out of range", e.Get("ledgerClosedAt").Str)
	}

	value, err := base64.StdEncoding.DecodeString(e.Get("value").Str)
	if err != nil {
		return nil, fmt.Errorf("invalid event value: %w", err)
	}

	msg, err := w.parseMessageFromXDR(value)
	if err != nil {
		return nil, err
	}

	return &common.MessagePublication{
		TxID:             txID,
		Timestamp:        ts,
		Nonce:            msg.nonce,
		Sequence:         msg.sequence,
		ConsistencyLevel: msg.consistencyLevel,
		EmitterChain:     w.chainID,
		EmitterAddress:   msg.emitterAddress,
		Payload:          msg.payload,
		IsReobservation:  isReobservation,
		Unreliable:       false,
	}, nil
}

// parseMessageFromXDR parses the XDR-encoded Soroban event value into the published message fields.
func (w *watcher) parseMessageFromXDR(data []byte) (*parsedMessage, error) {
	var scVal stellarxdr.ScVal

	if err := stellarxdr.SafeUnmarshal(data, &scVal); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XDR: %w", err)
	}

	// An SCV_MAP without a map body decodes to a nil *ScMap.
	eventMap, ok := scVal.GetMap()
	if !ok || eventMap == nil {
		return nil, fmt.Errorf("event value is not a map")
	}

	fields := make(map[stellarxdr.ScSymbol]stellarxdr.ScVal, len(*eventMap))
	for _, entry := range *eventMap {
		key, isSym := entry.Key.GetSym()
		if _, dup := fields[key]; !isSym || dup {
			return nil, fmt.Errorf("event map has a non-symbol or duplicate key %q", key)
		}
		fields[key] = entry.Val
	}

	// A missing field is the zero ScVal, so its getter fails like a wrong type does.
	nonce, okNonce := fields["nonce"].GetU32()
	seq, okSeq := fields["sequence"].GetU64()
	emitter, okEmitter := fields["emitter_address"].GetBytes()
	payload, okPayload := fields["payload"].GetBytes()
	cl, okCL := fields["consistency_level"].GetU32()
	if len(fields) != messageFieldCount || !okNonce || !okSeq || !okEmitter || !okPayload || !okCL {
		return nil, fmt.Errorf("event map must hold exactly nonce (u32), sequence (u64), emitter_address (bytes), payload (bytes) and consistency_level (u32)")
	}
	if len(emitter) != stellarAddressLen {
		return nil, fmt.Errorf("emitter address is %d bytes, want %d", len(emitter), stellarAddressLen)
	}
	if cl > math.MaxUint8 {
		return nil, fmt.Errorf("consistency level %d overflows uint8", cl)
	}

	return &parsedMessage{
		nonce:            uint32(nonce),
		sequence:         uint64(seq),
		consistencyLevel: uint8(cl),
		emitterAddress:   vaa.Address(emitter),
		payload:          payload,
	}, nil
}
