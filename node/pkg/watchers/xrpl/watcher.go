package xrpl

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"time"

	addresscodec "github.com/Peersyst/xrpl-go/address-codec"
	"github.com/Peersyst/xrpl-go/xrpl/queries/server"
	subscribe "github.com/Peersyst/xrpl-go/xrpl/queries/subscription"
	streamtypes "github.com/Peersyst/xrpl-go/xrpl/queries/subscription/types"
	"github.com/Peersyst/xrpl-go/xrpl/queries/transactions"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
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
)

type Watcher struct {
	rpc           string
	contract      string
	unsafeDevMode bool

	msgChan       chan<- *common.MessagePublication
	obsvReqC      <-chan *gossipv1.ObservationRequest
	readinessSync readiness.Component
}

func NewWatcher(
	rpc string,
	contract string,
	unsafeDevMode bool,
	msgChan chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		rpc:           rpc,
		contract:      contract,
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
		zap.Bool("unsafeDevMode", w.unsafeDevMode),
	)

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
				ledgerIndex, err := w.getValidatedLedgerIndex(ctx, logger)
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
				if vaa.ChainID(req.ChainId) != vaa.ChainIDXRPL {
					panic("invalid chain ID")
				}

				txHash := req.TxHash
				logger.Info("Received reobservation request",
					zap.String("txHash", hex.EncodeToString(txHash)))

				msg, err := w.fetchAndParseTransaction(ctx, logger, txHash)
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

// subscribeAndProcess connects to XRPL WebSocket and subscribes to account transactions.
// It processes incoming transactions and publishes Wormhole messages.
func (w *Watcher) subscribeAndProcess(ctx context.Context, logger *zap.Logger) error {
	// Create XRPL WebSocket client
	cfg := websocket.NewClientConfig().WithHost(w.rpc)
	client := websocket.NewClient(cfg)

	// Connect to XRPL node
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to XRPL node: %w", err)
	}
	defer client.Disconnect()

	logger.Info("Connected to XRPL node", zap.String("rpc", w.rpc))

	// Subscribe to the contract account
	subscribeReq := &subscribe.Request{
		Accounts: []types.Address{types.Address(w.contract)},
	}

	_, err := client.Request(subscribeReq)
	if err != nil {
		return fmt.Errorf("failed to subscribe to account %s: %w", w.contract, err)
	}

	logger.Info("Subscribed to account", zap.String("account", w.contract))

	// Set up transaction handler
	txChan := make(chan *streamtypes.TransactionStream, 100)
	client.OnTransactions(func(tx *streamtypes.TransactionStream) {
		select {
		case txChan <- tx:
		default:
			logger.Warn("Transaction channel full, dropping transaction")
		}
	})

	// Process incoming transactions
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case tx := <-txChan:
			if err := w.processTransaction(ctx, logger, tx); err != nil {
				logger.Error("Failed to process transaction",
					zap.String("hash", string(tx.Hash)),
					zap.Error(err))
			}
		}
	}
}

// processTransaction handles an incoming transaction from the subscription.
func (w *Watcher) processTransaction(ctx context.Context, logger *zap.Logger, tx *streamtypes.TransactionStream) error {
	// Only process validated transactions
	if !tx.Validated {
		logger.Debug("Skipping unvalidated transaction", zap.String("hash", string(tx.Hash)))
		return nil
	}

	// Parse the transaction and extract Wormhole message
	msg, err := w.parseTransactionStream(tx)
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
func (w *Watcher) fetchAndParseTransaction(ctx context.Context, logger *zap.Logger, txHash []byte) (*common.MessagePublication, error) {
	// Create WebSocket client for the request
	cfg := websocket.NewClientConfig().WithHost(w.rpc)
	client := websocket.NewClient(cfg)

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to XRPL node: %w", err)
	}
	defer client.Disconnect()

	// Fetch transaction by hash
	txReq := &transactions.TxRequest{
		Transaction: hex.EncodeToString(txHash),
	}

	resp, err := client.Request(txReq)
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
	return w.parseTxResponse(&txResp)
}

// getValidatedLedgerIndex returns the current validated ledger index.
func (w *Watcher) getValidatedLedgerIndex(ctx context.Context, logger *zap.Logger) (int64, error) {
	// Create WebSocket client for the request
	cfg := websocket.NewClientConfig().WithHost(w.rpc)
	client := websocket.NewClient(cfg)

	if err := client.Connect(); err != nil {
		return 0, fmt.Errorf("failed to connect to XRPL node: %w", err)
	}
	defer client.Disconnect()

	// Request server info
	infoReq := &server.InfoRequest{}
	resp, err := client.Request(infoReq)
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

	return int64(infoResp.Info.ValidatedLedger.Seq), nil
}

// parseTransactionStream converts an XRPL TransactionStream into a MessagePublication.
func (w *Watcher) parseTransactionStream(tx *streamtypes.TransactionStream) (*common.MessagePublication, error) {
	// Validate transaction result is tesSUCCESS
	if err := w.validateTransactionResult(tx.Transaction); err != nil {
		return nil, err
	}

	// Validate destination is our custody account (contract)
	if err := w.validateDestination(tx.Transaction); err != nil {
		return nil, err
	}

	// Extract payload from Memos
	payload, nonce, err := w.extractWormholePayload(tx.Transaction)
	if err != nil {
		return nil, err
	}

	// No Wormhole payload found - this is not an error, just not a Wormhole transaction
	if payload == nil {
		return nil, nil
	}

	// Validate recipient chain is a valid Wormhole chain ID
	if err := w.validateRecipientChain(payload); err != nil {
		return nil, err
	}

	// Validate NTT amount does not exceed delivered amount
	if err := w.validateAmount(tx.Meta.DeliveredAmount, payload); err != nil {
		return nil, err
	}

	// Extract transaction hash (32 bytes)
	txHash, err := hex.DecodeString(string(tx.Hash))
	if err != nil {
		return nil, fmt.Errorf("failed to decode tx hash: %w", err)
	}

	// Parse ledger close time
	timestamp, err := time.Parse(time.RFC3339, tx.CloseTimeISO)
	if err != nil {
		return nil, fmt.Errorf("failed to parse close time: %w", err)
	}

	// Calculate sequence: (ledgerIndex << 32) | txIndex
	// TransactionIndex is available in the Meta field for validated transactions
	ledgerIndex := uint64(tx.LedgerIndex)
	txIndex := uint64(tx.Meta.TransactionIndex)
	sequence := (ledgerIndex << 32) | txIndex

	// Convert contract address to 32-byte emitter address (left-padded with zeros)
	emitterAddress, err := w.addressToEmitter(w.contract)
	if err != nil {
		return nil, fmt.Errorf("failed to convert emitter address: %w", err)
	}

	return &common.MessagePublication{
		TxID:             txHash,
		Timestamp:        timestamp,
		Nonce:            nonce,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDXRPL,
		EmitterAddress:   emitterAddress,
		Payload:          payload,
		ConsistencyLevel: 0, // XRPL validated ledgers are final
		IsReobservation:  false,
	}, nil
}

// parseTxResponse converts a TxResponse (from reobservation) into a MessagePublication.
func (w *Watcher) parseTxResponse(tx *transactions.TxResponse) (*common.MessagePublication, error) {
	// Extract Wormhole payload from Memos in the transaction
	// TxResponse has TxJSON which is a FlatTransaction
	payload, nonce, err := w.extractWormholePayload(tx.TxJSON)
	if err != nil {
		return nil, err
	}

	if payload == nil {
		return nil, fmt.Errorf("no Wormhole payload found in transaction")
	}

	// Extract transaction hash
	txHash, err := hex.DecodeString(string(tx.Hash))
	if err != nil {
		return nil, fmt.Errorf("failed to decode tx hash: %w", err)
	}

	// Calculate sequence: (ledgerIndex << 32) | txIndex
	// TransactionIndex is available in the Meta field for validated transactions
	ledgerIndex := uint64(tx.LedgerIndex)
	txIndex := uint64(tx.Meta.TransactionIndex)
	sequence := (ledgerIndex << 32) | txIndex

	// Convert contract address to emitter
	emitterAddress, err := w.addressToEmitter(w.contract)
	if err != nil {
		return nil, fmt.Errorf("failed to convert emitter address: %w", err)
	}

	return &common.MessagePublication{
		TxID:             txHash,
		Timestamp:        time.Now(), // TxResponse may not have close time readily available
		Nonce:            nonce,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDXRPL,
		EmitterAddress:   emitterAddress,
		Payload:          payload,
		ConsistencyLevel: 0,
		IsReobservation:  false,
	}, nil
}

// nttMemoType is the hex-encoded MemoType for NTT transfers: "application/x-ntt-transfer"
const nttMemoType = "6170706C69636174696F6E2F782D6E74742D7472616E73666572"

// nttPayloadPrefix is the 4-byte prefix for NTT payloads: 0x994E5454
var nttPayloadPrefix = []byte{0x99, 0x4E, 0x54, 0x54}

// NTT payload offsets and sizes
const (
	nttPayloadMinLength     = 79 // Minimum valid NTT payload length
	nttRecipientChainOffset = 77 // Offset of recipient chain ID (2 bytes, big-endian)
	nttAmountOffset         = 5  // Offset of amount (8 bytes, big-endian)

	// TODO: These offsets are defined for future validation implementation
	// nttDecimalsOffset    = 4  // Offset of decimals (1 byte)
	// nttSourceTokenOffset = 13 // Offset of source token address (32 bytes)
)

// tesSUCCESS is the XRPL transaction result code for successful transactions
const tesSUCCESS = "tesSUCCESS"

// extractWormholePayload extracts the Wormhole message payload from transaction Memos.
// Returns the payload bytes, nonce, and any error.
// Returns (nil, 0, nil) if no Wormhole payload is found (not an error, just not a Wormhole tx).
//
// Currently only NTT transfers are supported on XRPL. The NTT payload format does not include
// a nonce field, so we return 0. If generic Wormhole messages are added in the future, the
// nonce would need to be extracted from a different memo format or field.
func (w *Watcher) extractWormholePayload(tx transaction.FlatTransaction) ([]byte, uint32, error) {
	// FlatTransaction is map[string]interface{}
	// Memos is an array of objects with structure: [{"Memo": {"MemoType": "...", "MemoData": "..."}}]
	memosRaw, ok := tx["Memos"]
	if !ok {
		return nil, 0, nil
	}

	memos, ok := memosRaw.([]interface{})
	if !ok {
		return nil, 0, nil
	}

	for _, memoWrapperRaw := range memos {
		memoWrapper, ok := memoWrapperRaw.(map[string]interface{})
		if !ok {
			continue
		}

		memoRaw, ok := memoWrapper["Memo"]
		if !ok {
			continue
		}

		memo, ok := memoRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Check MemoType matches NTT transfer type
		memoType, ok := memo["MemoType"].(string)
		if !ok || memoType != nttMemoType {
			continue
		}

		// Extract and decode MemoData (hex-encoded payload)
		memoData, ok := memo["MemoData"].(string)
		if !ok {
			continue
		}

		payload, err := hex.DecodeString(memoData)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to decode MemoData: %w", err)
		}

		// Verify NTT payload prefix
		if len(payload) < 4 || payload[0] != nttPayloadPrefix[0] || payload[1] != nttPayloadPrefix[1] ||
			payload[2] != nttPayloadPrefix[2] || payload[3] != nttPayloadPrefix[3] {
			continue
		}

		// NOTE: Nonce is not included in NTT payload, use 0
		return payload, 0, nil
	}

	return nil, 0, nil
}

// addressToEmitter converts an XRPL address to a 32-byte VAA emitter address.
// XRPL addresses are base58-encoded (r-address format) and decode to 20-byte account IDs.
// The account ID is left-padded with 12 zero bytes to create the 32-byte emitter address.
func (w *Watcher) addressToEmitter(address string) (vaa.Address, error) {
	// DecodeClassicAddressToAccountID returns the type prefix and 20-byte account ID
	_, accountID, err := addresscodec.DecodeClassicAddressToAccountID(address)
	if err != nil {
		return vaa.Address{}, fmt.Errorf("failed to decode XRPL address %s: %w", address, err)
	}

	// Account ID should be 20 bytes
	if len(accountID) != addresscodec.AccountAddressLength {
		return vaa.Address{}, fmt.Errorf("unexpected account ID length: got %d, want %d", len(accountID), addresscodec.AccountAddressLength)
	}

	// Left-pad with zeros to create 32-byte emitter address
	// vaa.Address is [32]byte, accountID is 20 bytes
	// Place accountID in the last 20 bytes (indices 12-31)
	var emitter vaa.Address
	copy(emitter[32-addresscodec.AccountAddressLength:], accountID)

	return emitter, nil
}

// validateTransactionResult checks that the transaction result is tesSUCCESS.
// Returns nil if valid, or an error describing why the transaction should be skipped.
// Returns nil (no error) if result cannot be determined - allows processing to continue.
func (w *Watcher) validateTransactionResult(tx transaction.FlatTransaction) error {
	metaRaw, ok := tx["meta"]
	if !ok {
		// No meta field - this might be a subscription stream where meta is at top level
		// Allow processing to continue
		return nil
	}

	meta, ok := metaRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	resultRaw, ok := meta["TransactionResult"]
	if !ok {
		return nil
	}

	result, ok := resultRaw.(string)
	if !ok {
		return nil
	}

	if result != tesSUCCESS {
		return fmt.Errorf("transaction result is %s, not %s", result, tesSUCCESS)
	}

	return nil
}

// validateDestination checks that the transaction destination is the custody account (contract).
// Returns nil if valid, or an error if the destination doesn't match.
// Returns nil (no error) if destination cannot be determined - allows processing to continue.
func (w *Watcher) validateDestination(tx transaction.FlatTransaction) error {
	destRaw, ok := tx["Destination"]
	if !ok {
		// No destination field - might not be a Payment transaction
		// Allow processing to continue, extractWormholePayload will filter non-NTT
		return nil
	}

	dest, ok := destRaw.(string)
	if !ok {
		return nil
	}

	if dest != w.contract {
		return fmt.Errorf("transaction destination %s does not match custody account %s", dest, w.contract)
	}

	return nil
}

// validateRecipientChain checks that the recipient chain in the NTT payload is a valid Wormhole chain ID.
// Returns nil if valid, or an error if the chain ID is invalid.
func (w *Watcher) validateRecipientChain(payload []byte) error {
	if len(payload) < nttPayloadMinLength {
		return fmt.Errorf("NTT payload too short: got %d bytes, need at least %d", len(payload), nttPayloadMinLength)
	}

	// Extract recipient chain ID (last 2 bytes, big-endian)
	recipientChain := binary.BigEndian.Uint16(payload[nttRecipientChainOffset:])
	chainID := vaa.ChainID(recipientChain)

	// Validate chain ID is known
	// ChainID 0 is Unset and invalid
	if chainID == vaa.ChainIDUnset {
		return fmt.Errorf("invalid recipient chain ID: 0 (unset)")
	}

	// Check against known Wormhole chain IDs
	if !slices.Contains(vaa.GetAllNetworkIDs(), chainID) {
		return fmt.Errorf("unknown recipient chain ID: %d", recipientChain)
	}

	return nil
}

// validateAmount checks that the amount in the NTT payload does not exceed the delivered amount.
// This prevents over-claiming by ensuring ntt_amount <= delivered_amount.
// deliveredAmount is the value from tx.Meta.DeliveredAmount (string for XRP drops).
func (w *Watcher) validateAmount(deliveredAmount any, payload []byte) error {
	if len(payload) < nttPayloadMinLength {
		return fmt.Errorf("NTT payload too short for amount validation: got %d bytes", len(payload))
	}

	// Extract NTT amount from payload (8 bytes, big-endian, at offset 5)
	nttAmount := binary.BigEndian.Uint64(payload[nttAmountOffset : nttAmountOffset+8])

	// Parse delivered_amount - for XRP it's a string of drops
	// NTT on XRPL uses XRP (native currency), so delivered_amount must be a string
	deliveredStr, ok := deliveredAmount.(string)
	if !ok {
		return fmt.Errorf("delivered_amount is not a string (got %T), NTT requires XRP payments", deliveredAmount)
	}

	// Parse the delivered amount (XRP drops as string)
	delivered, err := strconv.ParseUint(deliveredStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse delivered_amount %q: %w", deliveredStr, err)
	}

	// Verify NTT amount does not exceed delivered amount
	if nttAmount > delivered {
		return fmt.Errorf("NTT amount %d exceeds delivered amount %d", nttAmount, delivered)
	}

	return nil
}

// TODO: Implement validateAmountDecimals - verify amount uses at most 8 decimal places
// XRPL amounts use a specific precision, and NTT amounts should be trimmed to 8 decimals max.
// func (w *Watcher) validateAmountDecimals(payload []byte) error
