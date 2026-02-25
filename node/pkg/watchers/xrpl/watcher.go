package xrpl

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/Peersyst/xrpl-go/xrpl/queries/server"
	subscribe "github.com/Peersyst/xrpl-go/xrpl/queries/subscription"
	streamtypes "github.com/Peersyst/xrpl-go/xrpl/queries/subscription/types"
	"github.com/Peersyst/xrpl-go/xrpl/queries/transactions"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"
	"github.com/Peersyst/xrpl-go/xrpl/websocket"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// NOTE: This watcher conforms to the guidelines in ../README.md

// Prometheus metrics
var (
	xrplConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_xrpl_connection_errors_total",
			Help: "Total number of XRPL connection errors",
		}, []string{"reason"})
	xrplMessagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_xrpl_observations_confirmed_total",
			Help: "Total number of verified XRPL observations found",
		})
	currentXrplLedger = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_xrpl_current_ledger",
			Help: "Current XRPL validated ledger index",
		})
	xrplTxDropped = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_xrpl_transactions_dropped_total",
			Help: "Total number of XRPL transactions dropped due to full channel",
		})
)

type Watcher struct {
	rpc           string
	contract      string
	nttAccounts   []string
	unsafeDevMode bool

	msgChan       chan<- *common.MessagePublication
	obsvReqC      <-chan *gossipv1.ObservationRequest
	readinessSync readiness.Component

	// WebSocket client - created once at startup, shared across all operations
	client *websocket.Client

	// parser handles NTT transaction parsing
	parser *Parser

	// txChan receives transactions from the WebSocket handler.
	// Created once in Run() to avoid multiple handler registrations.
	txChan chan *streamtypes.TransactionStream
}

func NewWatcher(
	rpc string,
	contract string,
	nttAccounts []string,
	unsafeDevMode bool,
	msgChan chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		rpc:           rpc,
		contract:      contract,
		nttAccounts:   nttAccounts,
		unsafeDevMode: unsafeDevMode,
		msgChan:       msgChan,
		obsvReqC:      obsvReqC,
		readinessSync: common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDXRPL),
	}
}

func (w *Watcher) Run(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDXRPL, &gossipv1.Heartbeat_Network{
		ContractAddress: w.contract,
	})

	logger.Info("Starting watcher",
		zap.String("watcher_name", "xrpl"),
		zap.String("rpc", w.rpc),
		zap.String("contract", w.contract),
		zap.Strings("nttAccounts", w.nttAccounts),
		zap.Bool("unsafeDevMode", w.unsafeDevMode),
	)

	// Connect to XRPL node once at startup
	cfg := websocket.NewClientConfig().WithHost(w.rpc)
	// If you need to change the timeout from a builtin default of 5s, then do:
	// cfg := websocket.NewClientConfig().
	//   WithHost(w.rpc).
	//   WithTimeout(10 * time.Second)

	w.client = websocket.NewClient(cfg)
	if err := w.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to XRPL node: %w", err)
	}
	defer func() {
		_ = w.client.Disconnect()
	}()

	logger.Info("Connected to XRPL node", zap.String("rpc", w.rpc))

	// Initialize the parser with the watcher's fetchMPTAssetScale method
	w.parser = NewParser(w.contract, w.nttAccounts, w.fetchMPTAssetScale)

	// Create the transaction channel once - handlers will write to this channel
	w.txChan = make(chan *streamtypes.TransactionStream, 100)

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	errC := make(chan error)
	defer close(errC)

	// GOROUTINE 1: Main event subscription loop
	// Subscribes to the Wormhole account and processes incoming transactions
	common.RunWithScissors(ctx, errC, "xrpl_data_pump", func(ctx context.Context) error {
		for {
			select {
			case err := <-errC:
				logger.Error("xrpl_data_pump died", zap.Error(err))
				return fmt.Errorf("xrpl_data_pump died: %w", err)
			case <-ctx.Done():
				return ctx.Err()
			default:
				err := w.subscribeAndProcess(ctx, logger)
				if err != nil {
					logger.Error("Subscription error, reconnecting", zap.Error(err))
					xrplConnectionErrors.WithLabelValues("subscription_error").Inc()
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDXRPL, 1)
					time.Sleep(time.Second * 5)
				}
			}
		}
	})

	// GOROUTINE 2: Ledger index tracking (for metrics/heartbeat)
	common.RunWithScissors(ctx, errC, "xrpl_ledger_height", func(ctx context.Context) error {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()

		for {
			select {
			case err := <-errC:
				logger.Error("xrpl_ledger_height died", zap.Error(err))
				return fmt.Errorf("xrpl_ledger_height died: %w", err)
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				ledgerIndex, err := w.getValidatedLedgerIndex(logger)
				if err != nil {
					logger.Error("Failed to get validated ledger index", zap.Error(err))
					xrplConnectionErrors.WithLabelValues("ledger_height_error").Inc()
					continue
				}

				currentXrplLedger.Set(float64(ledgerIndex))
				p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDXRPL, &gossipv1.Heartbeat_Network{
					Height:          ledgerIndex,
					ContractAddress: w.contract,
				})

				readiness.SetReady(w.readinessSync)
			}
		}
	})

	// GOROUTINE 3: Handle reobservation requests
	common.RunWithScissors(ctx, errC, "xrpl_fetch_obvs_req", func(ctx context.Context) error {
		for {
			select {
			case err := <-errC:
				logger.Error("xrpl_fetch_obvs_req died", zap.Error(err))
				return fmt.Errorf("xrpl_fetch_obvs_req died: %w", err)
			case <-ctx.Done():
				return ctx.Err()
			case req := <-w.obsvReqC:
				if req.ChainId > math.MaxUint16 {
					logger.Error("chain id for observation request is not a valid uint16",
						zap.Uint32("chainID", req.ChainId),
						zap.String("txID", hex.EncodeToString(req.TxHash)),
					)
					continue
				}
				if vaa.ChainID(req.ChainId) != vaa.ChainIDXRPL {
					panic("invalid chain ID")
				}

				txHash := req.TxHash
				logger.Info("Received reobservation request",
					zap.String("txHash", hex.EncodeToString(txHash)))

				msg, err := w.fetchAndParseTransaction(txHash)
				if err != nil {
					logger.Error("Failed to fetch transaction for reobservation",
						zap.String("txHash", hex.EncodeToString(txHash)),
						zap.Error(err))
					xrplConnectionErrors.WithLabelValues("reobservation_error").Inc()
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDXRPL, 1)
					continue
				}

				if msg != nil {
					msg.IsReobservation = true
					w.msgChan <- msg
					watchers.ReobservationsByChain.WithLabelValues("xrpl", "std").Inc()
				}
			}
		}
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// subscribeAndProcess subscribes to account transactions and processes them.
// It uses the WebSocket client that was connected in Run().
func (w *Watcher) subscribeAndProcess(ctx context.Context, logger *zap.Logger) error {
	// Create a child context for this subscription attempt.
	// When this function returns (error or context cancelled), the child context
	// is cancelled, causing the OnTransactions handler to stop writing to txChan.
	// This prevents stale handlers from writing after we've moved on to a new subscription.
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Subscribe to the core account and any NTT accounts
	accounts := []types.Address{types.Address(w.contract)}
	for _, nttAccount := range w.nttAccounts {
		accounts = append(accounts, types.Address(nttAccount))
	}
	subscribeReq := &subscribe.Request{
		Accounts: accounts,
	}

	_, err := w.client.Request(subscribeReq)
	if err != nil {
		return fmt.Errorf("failed to subscribe to accounts %v: %w", accounts, err)
	}

	logger.Info("Subscribed to accounts",
		zap.String("contract", w.contract),
		zap.Strings("nttAccounts", w.nttAccounts),
	)

	// Set up transaction handler for this subscription attempt.
	// The handler writes to the shared txChan but respects subCtx cancellation.
	w.client.OnTransactions(func(tx *streamtypes.TransactionStream) {
		select {
		case <-subCtx.Done():
			return
		case w.txChan <- tx:
		default:
			logger.Warn("Transaction channel full, dropping transaction",
				zap.String("hash", string(tx.Hash)))
			xrplTxDropped.Inc()
		}
	})

	// Process incoming transactions
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case tx := <-w.txChan:
			if err := w.processTransaction(logger, tx); err != nil {
				logger.Error("Failed to process transaction",
					zap.String("hash", string(tx.Hash)),
					zap.Error(err))
			}
		}
	}
}

// processTransaction handles an incoming transaction from the subscription.
func (w *Watcher) processTransaction(logger *zap.Logger, tx *streamtypes.TransactionStream) error {
	// Only process validated transactions
	if !tx.Validated {
		logger.Debug("Skipping unvalidated transaction", zap.String("hash", string(tx.Hash)))
		return nil
	}

	// Parse the transaction and extract Wormhole message
	msg, err := w.parser.ParseTransactionStream(tx)
	if err != nil {
		return fmt.Errorf("failed to parse transaction: %w", err)
	}

	// msg is nil if transaction doesn't contain a Wormhole message
	if msg == nil {
		return nil
	}

	// Send to processor
	w.msgChan <- msg
	xrplMessagesConfirmed.Inc()

	logger.Info("Message observed",
		zap.String("txHash", hex.EncodeToString(msg.TxID)),
		zap.Uint64("sequence", msg.Sequence),
		zap.Uint32("nonce", msg.Nonce),
	)

	return nil
}

// fetchAndParseTransaction fetches a specific transaction by hash for reobservation.
func (w *Watcher) fetchAndParseTransaction(txHash []byte) (*common.MessagePublication, error) {
	// Fetch transaction by hash
	txReq := &transactions.TxRequest{
		Transaction: hex.EncodeToString(txHash),
	}

	resp, err := w.client.Request(txReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	// Decode the response into TxResponse
	var txResp transactions.TxResponse
	if err := resp.GetResult(&txResp); err != nil {
		return nil, fmt.Errorf("failed to decode transaction response: %w", err)
	}

	// Only process validated transactions
	if !txResp.Validated {
		return nil, fmt.Errorf("transaction not yet validated")
	}

	// Parse the transaction
	return w.parser.ParseTxResponse(&txResp)
}

// getValidatedLedgerIndex returns the current validated ledger index.
func (w *Watcher) getValidatedLedgerIndex(logger *zap.Logger) (int64, error) {
	// Request server info
	infoReq := &server.InfoRequest{}
	resp, err := w.client.Request(infoReq)
	if err != nil {
		return 0, fmt.Errorf("failed to get server info: %w", err)
	}

	// Decode the response into InfoResponse
	var infoResp server.InfoResponse
	if err := resp.GetResult(&infoResp); err != nil {
		return 0, fmt.Errorf("failed to decode server info response: %w", err)
	}

	// Check server state for readiness
	state := infoResp.Info.ServerState
	// States are here: https://xrpl.org/docs/references/http-websocket-apis/api-conventions/rippled-server-states
	if state != "full" && state != "proposing" && state != "validating" {
		logger.Warn("XRPL node not fully synced", zap.String("state", state))
	}

	seq := infoResp.Info.ValidatedLedger.Seq
	if seq > math.MaxInt64 {
		return 0, fmt.Errorf("ledger sequence %d exceeds max int64", seq)
	}
	return int64(seq), nil
}

// mptLedgerEntryRequest is a custom request type for fetching MPT issuance details.
// The xrpl-go library doesn't have built-in support for MPT ledger entries.
type mptLedgerEntryRequest struct {
	MPTIssuance string `json:"mpt_issuance"`
	LedgerIndex string `json:"ledger_index"`
}

func (r *mptLedgerEntryRequest) Method() string  { return "ledger_entry" }
func (r *mptLedgerEntryRequest) Validate() error { return nil }
func (r *mptLedgerEntryRequest) APIVersion() int { return 1 }

// fetchMPTAssetScale fetches the AssetScale for an MPT from the ledger.
func (w *Watcher) fetchMPTAssetScale(mptID string) (uint8, error) {
	req := &mptLedgerEntryRequest{
		MPTIssuance: mptID,
		LedgerIndex: "validated",
	}

	resp, err := w.client.Request(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch MPT ledger entry: %w", err)
	}

	var result map[string]interface{}
	if err := resp.GetResult(&result); err != nil {
		return 0, fmt.Errorf("failed to decode MPT ledger entry response: %w", err)
	}

	// Extract node from response
	nodeRaw, ok := result["node"]
	if !ok {
		return 0, fmt.Errorf("MPT ledger entry response missing 'node' field")
	}
	node, ok := nodeRaw.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("MPT ledger entry 'node' is not a map")
	}

	// Extract AssetScale
	assetScaleRaw, ok := node["AssetScale"]
	if !ok {
		// AssetScale defaults to 0 if not present
		return 0, nil
	}

	// AssetScale can be a float64 from JSON
	switch v := assetScaleRaw.(type) {
	case float64:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("AssetScale out of range: %f", v)
		}
		return uint8(v), nil
	case int:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("AssetScale out of range: %d", v)
		}
		return uint8(v), nil
	default:
		return 0, fmt.Errorf("unexpected AssetScale type: %T", assetScaleRaw)
	}
}
