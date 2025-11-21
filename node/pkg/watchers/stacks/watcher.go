package stacks

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

/// OVERVIEW
// The Stacks watcher monitors the Stacks blockchain for cross-chain Wormhole message events.
// It uses Bitcoin blocks (burn blocks) as the anchor point for confirmation and processes
// Stacks blocks that are anchored to confirmed Bitcoin blocks.
//
// Core Components and Process Flow:
// - Public Methods:
//    - Run: Main entry point that starts the block poller and observation request handler
//    - Reobserve: Implements reobservation support for previously emitted messages
//
// - Execution Flow:
//    - runBlockPoller: Polls for new Bitcoin blocks and processes confirmed blocks
//    - process...: (`processCoreEvent` is the main/final function)
//      - Bitcoin Block → Stacks Blocks → Transactions → Events → Message Data
//
// API Interaction, aka fetch methods are in `fetch.go`.

// Safe overflow checking constants for BigInt validation
var (
	maxUint32BigInt = big.NewInt(math.MaxUint32)
	maxUint64BigInt = new(big.Int).SetUint64(math.MaxUint64)
	maxUint8BigInt  = big.NewInt(math.MaxUint8)
	maxInt64        = uint64(math.MaxInt64)
)

type (
	Watcher struct {
		rpcURL        string
		rpcAuthToken  string
		stateContract string

		bitcoinBlockPollInterval time.Duration

		msgC          chan<- *common.MessagePublication
		obsvReqC      <-chan *gossipv1.ObservationRequest
		readinessSync readiness.Component

		nakamotoBitcoinHeight atomic.Uint64 // We can't process blocks before this height

		stableBitcoinHeight    atomic.Uint64
		latestBitcoinHeight    atomic.Uint64
		processedBitcoinHeight atomic.Uint64
	}

	MessageData struct {
		EmitterAddress   vaa.Address
		Nonce            uint32
		Sequence         uint64
		ConsistencyLevel uint8
		Payload          []byte
	}
)

func NewWatcher(
	rpcURL string,
	rpcAuthToken string,
	contract string,
	bitcoinBlockPollInterval time.Duration,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	w := &Watcher{
		rpcURL:                   rpcURL,
		rpcAuthToken:             rpcAuthToken,
		stateContract:            contract,
		bitcoinBlockPollInterval: bitcoinBlockPollInterval,
		msgC:                     msgC,
		obsvReqC:                 obsvReqC,
		readinessSync:            common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDStacks),
	}

	w.latestBitcoinHeight.Store(0)
	w.processedBitcoinHeight.Store(0)

	return w
}

/// WATCHER PUBLIC METHODS

func (w *Watcher) Run(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	logger.Info("Starting Stacks watcher",
		zap.String("rpc_url", w.rpcURL),
		zap.String("contract", w.stateContract))

	errC := make(chan error)

	// Start block poller
	common.RunWithScissors(ctx, errC, "stacksBlockPoller", w.runBlockPoller)

	// Handle observation requests
	common.RunWithScissors(ctx, errC, "stacksObsvReqWorker", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case req := <-w.obsvReqC:

				if req.ChainId != uint32(vaa.ChainIDStacks) {
					logger.Error("Unexpected chain ID",
						zap.Uint32("chain_id", req.ChainId))
					continue
				}

				logger.Info("Received Stacks observation request",
					zap.String("tx_hash", hex.EncodeToString(req.TxHash)),
					zap.Int64("timestamp", req.Timestamp))

				numObservations, err := w.Reobserve(ctx, vaa.ChainIDStacks, req.TxHash, "")
				if err != nil {
					logger.Error("Failed to process observation request",
						zap.String("tx_hash", hex.EncodeToString(req.TxHash)),
						zap.Uint32("num_observations", numObservations),
						zap.Error(err),
					)
					continue
				}

				logger.Info("Reobserved transactions",
					zap.String("tx_hash", hex.EncodeToString(req.TxHash)),
					zap.Uint32("num_observations", numObservations),
				)
			}
		}
	})

	// Set initial readiness state
	readiness.SetReady(w.readinessSync)

	// Wait for error or context cancellation
	select {
	case <-ctx.Done():
		logger.Info("Context cancelled, stopping Stacks watcher")
		return nil
	case err := <-errC:
		return err
	}
}

// Reobserve implements the interfaces.Reobserver interface.
func (w *Watcher) Reobserve(ctx context.Context, chainID vaa.ChainID, txID []byte, customEndpoint string) (uint32, error) {
	logger := supervisor.Logger(ctx)

	// Verify this request is for our chain
	if chainID != vaa.ChainIDStacks {
		return 0, fmt.Errorf("unexpected chain ID: %v", chainID)
	}

	txIdString := hex.EncodeToString(txID)
	logger.Info("Received reobservation request",
		zap.String("tx_id", txIdString),
		zap.String("custom_endpoint", customEndpoint))

	// Process the transaction
	count, err := w.reobserveStacksTransactionByTxId(ctx, txIdString, logger)
	if err != nil {
		logger.Error("Failed to reobserve transaction",
			zap.String("tx_id", txIdString),
			zap.Error(err))
		return 0, err
	}

	return count, nil
}

/// RUN

// Polls for new Bitcoin (burn) blocks and processes confirmed blocks
func (w *Watcher) runBlockPoller(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	logger.Info("Starting Stacks block poller",
		zap.String("rpc_url", w.rpcURL),
		zap.String("contract", w.stateContract),
		zap.Duration("poll_interval", w.bitcoinBlockPollInterval))

	poxInfo, err := w.fetchPoxInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch PoX info: %w", err)
	}

	var nakamotoEpoch *StacksV2PoxEpoch
	for _, epoch := range poxInfo.Epochs {
		if epoch.EpochID == "Epoch30" {
			nakamotoEpoch = &epoch
			break
		}
	}

	if nakamotoEpoch == nil {
		return fmt.Errorf("failed to find Nakamoto epoch (Epoch30) in PoX info")
	}

	w.nakamotoBitcoinHeight.Store(nakamotoEpoch.StartHeight)

	nodeInfo, err := w.fetchNodeInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch node info: %w", err)
	}

	// Set to stable or nakamoto height, whichever is higher
	// Act as if all blocks up to the stable burn block height have been processed
	if nakamotoEpoch.StartHeight > nodeInfo.StableBurnBlockHeight {
		w.processedBitcoinHeight.Store(nakamotoEpoch.StartHeight)
	} else {
		w.processedBitcoinHeight.Store(nodeInfo.StableBurnBlockHeight)
	}

	logger.Info("Initialized Stacks watcher with stable Bitcoin (burn) block",
		zap.Uint64("stable_bitcoin_block_height", nodeInfo.StableBurnBlockHeight))

	// Convert StableBurnBlockHeight to int64 with overflow check
	stableHeight := nodeInfo.StableBurnBlockHeight
	if stableHeight > maxInt64 {
		return fmt.Errorf("stable burn block height %d exceeds maximum int64 value", stableHeight)
	}

	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDStacks, &gossipv1.Heartbeat_Network{
		Height:          int64(stableHeight), // #nosec G115 -- checked above
		ContractAddress: w.stateContract,
	})

	timer := time.NewTimer(w.bitcoinBlockPollInterval)
	defer timer.Stop()

	// Poll loop
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			nodeInfo, err := w.fetchNodeInfo(ctx)
			if err != nil {
				logger.Error("Failed to fetch node info",
					zap.Error(err))
				timer.Reset(w.bitcoinBlockPollInterval)
				continue
			}

			previousStableBitcoinHeight := w.stableBitcoinHeight.Load()

			// We have a new stable Bitcoin (burn) block height
			if nodeInfo.StableBurnBlockHeight > previousStableBitcoinHeight {
				logger.Info("Found new stable Bitcoin (burn) block",
					zap.Uint64("previous_stable_height", previousStableBitcoinHeight),
					zap.Uint64("stable_height", nodeInfo.StableBurnBlockHeight))

				w.stableBitcoinHeight.Store(nodeInfo.StableBurnBlockHeight)

				// Convert StableBurnBlockHeight to int64 with overflow check
				newStableHeight := nodeInfo.StableBurnBlockHeight
				if newStableHeight > maxInt64 {
					logger.Error("Stable burn block height exceeds maximum int64 value",
						zap.Uint64("height", newStableHeight))
					timer.Reset(w.bitcoinBlockPollInterval)
					continue
				}

				p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDStacks, &gossipv1.Heartbeat_Network{
					Height:          int64(newStableHeight), // #nosec G115 -- checked above
					ContractAddress: w.stateContract,
				})

				bitcoinFromHeight := w.processedBitcoinHeight.Load() + 1

				logger.Info("Processing Bitcoin (burn) blocks",
					zap.Uint64("from_height", bitcoinFromHeight),
					zap.Uint64("to_height", nodeInfo.StableBurnBlockHeight))

				// Processing loop
				for height := bitcoinFromHeight; height <= nodeInfo.StableBurnBlockHeight; height++ {
					tenure, err := w.fetchTenureBlocksByBurnHeight(ctx, height)
					if err != nil {
						logger.Error("Failed to fetch Bitcoin (burn) block",
							zap.Uint64("height", height),
							zap.Error(err))
						break
					}

					w.processBitcoinBlock(ctx, tenure, logger)
					w.processedBitcoinHeight.Store(height)
				}
			}

			timer.Reset(w.bitcoinBlockPollInterval)
		}
	}
}

/// PROCESS

// Processes all Stacks blocks anchored to the given Bitcoin (burn) block
func (w *Watcher) processBitcoinBlock(ctx context.Context, tenureBlocks *StacksV3TenureBlocksResponse, logger *zap.Logger) {
	logger.Info("Processing Bitcoin (burn) block",
		zap.Uint64("bitcoin_block_height", tenureBlocks.BurnBlockHeight),
		zap.String("bitcoin_block_hash", tenureBlocks.BurnBlockHash))

	// Process each Stacks block anchored to this burn block
	for _, block := range tenureBlocks.StacksBlocks {
		logger.Info("Processing Stacks block", zap.String("stacks_block_id", block.BlockId))

		// Fetch and process the Stacks block
		if err := w.processStacksBlock(ctx, block.BlockId, logger); err != nil {
			logger.Error("Failed to process Stacks block",
				zap.String("stacks_block_id", block.BlockId),
				zap.Error(err))
			// Continue processing other blocks even if one fails
		}
	}
}

// Fetches and processes all transactions in a Stacks block
func (w *Watcher) processStacksBlock(ctx context.Context, blockHash string, logger *zap.Logger) error {
	replay, err := w.fetchStacksBlockReplay(ctx, blockHash)
	if err != nil {
		return fmt.Errorf("failed to fetch Stacks block replay: %w", err)
	}

	for _, tx := range replay.Transactions {
		if _, err := w.processStacksTransaction(ctx, &tx, replay, false, logger); err != nil {
			logger.Error("Failed to process transaction",
				zap.String("tx_id", tx.TxId),
				zap.Error(err))
			// Continue processing other transactions even if one fails
		}
	}

	return nil
}

// Processes a single transaction from a Stacks block
func (w *Watcher) processStacksTransaction(_ context.Context, tx *StacksV3TenureBlockTransaction, replay *StacksV3TenureBlockReplayResponse, isReobservation bool, logger *zap.Logger) (uint32, error) {
	logger.Info("Processing Stacks transaction", zap.String("tx_id", tx.TxId))

	// non-okay response
	if !strings.HasPrefix(tx.ResultHex, "0x07") { // (ok) is 0x07...
		return 0, fmt.Errorf("transaction %s failed due to response hex: %s", tx.TxId, tx.ResultHex)
	}

	// abort_by_response
	if !isTransactionResultCommitted(tx.Result) {
		return 0, fmt.Errorf("transaction %s failed due to response: %v", tx.TxId, tx.Result)
	}

	// abort_by_post_condition
	if tx.PostConditionAborted {
		return 0, fmt.Errorf("transaction %s failed due to post-condition aborted", tx.TxId)
	}

	// other runtime error
	if tx.VmError != nil {
		return 0, fmt.Errorf("transaction %s failed due to runtime error: %s", tx.TxId, *tx.VmError)
	}

	// success

	wormholeEvents := uint32(0)
	for _, event := range tx.Events {
		// Skip events that don't match our criteria
		if !event.Committed ||
			event.Type != "contract_event" ||
			event.ContractEvent == nil ||
			event.ContractEvent.ContractIdentifier != w.stateContract ||
			event.ContractEvent.Topic != "print" {
			continue
		}

		logger.Info("Found Wormhole message event",
			zap.String("tx_id", tx.TxId),
			zap.Uint64("event_index", event.EventIndex))

		hexStr := strings.TrimPrefix(event.ContractEvent.RawValue, "0x")
		hexBytes, err := hex.DecodeString(hexStr)
		if err != nil {
			logger.Error("Failed to decode raw value hex",
				zap.String("tx_id", tx.TxId),
				zap.Uint64("event_index", event.EventIndex),
				zap.String("hex", event.ContractEvent.RawValue),
				zap.Error(err))
			continue
		}

		clarityValue, err := DecodeClarityValue(bytes.NewReader(hexBytes))
		if err != nil {
			logger.Error("Failed to decode clarity value",
				zap.String("tx_id", tx.TxId),
				zap.Uint64("event_index", event.EventIndex),
				zap.Error(err))
			continue
		}

		logger.Debug("Decoded clarity value",
			zap.String("tx_id", tx.TxId),
			zap.Uint64("event_index", event.EventIndex),
			zap.String("type", fmt.Sprintf("%T", clarityValue)))

		// Process the core event
		if err := w.processCoreEvent(clarityValue, tx.TxId, replay.Timestamp, isReobservation); err == nil {
			wormholeEvents++
		} else {
			logger.Error("Failed to process core event",
				zap.String("tx_id", tx.TxId),
				zap.Uint64("event_index", event.EventIndex),
				zap.Error(err))
			// Continue processing other events even if one fails
		}
	}

	logger.Info("Finished processing transaction events",
		zap.String("tx_id", tx.TxId),
		zap.Uint32("wormhole_events_processed", wormholeEvents))

	return wormholeEvents, nil
}

// Processes a single transaction by its txid (used for reobservations)
func (w *Watcher) reobserveStacksTransactionByTxId(ctx context.Context, txId string, logger *zap.Logger) (uint32, error) {
	logger.Info("Processing transaction by txid", zap.String("tx_id", txId))

	transaction, err := w.fetchStacksTransactionByTxId(ctx, txId)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	replay, err := w.fetchStacksBlockReplay(ctx, transaction.IndexBlockHash)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch block replay: %w", err)
	}

	stableBitcoinBlockHeight := w.stableBitcoinHeight.Load()
	if replay.BlockHeight > stableBitcoinBlockHeight {
		return 0, fmt.Errorf("block replay height %d is greater than stable Bitcoin (burn) block height %d", replay.BlockHeight, stableBitcoinBlockHeight)
	}

	var tx *StacksV3TenureBlockTransaction
	for i := range replay.Transactions {
		if replay.Transactions[i].TxId == txId {
			tx = &replay.Transactions[i]
			break
		}
	}

	if tx == nil {
		return 0, fmt.Errorf("transaction %s not found in block replay", txId)
	}

	// Process the transaction using the same processing function used in polling
	count, err := w.processStacksTransaction(ctx, tx, replay, true, logger)
	if err != nil {
		return 0, fmt.Errorf("failed to process transaction: %w", err)
	}

	return count, nil
}

// Processes a core contract event tuple and extracts message fields
func (w *Watcher) processCoreEvent(clarityValue ClarityValue, txId string, timestamp uint64, isReobservation bool) error {
	// Cast to tuple
	eventTuple, isTuple := clarityValue.(*Tuple)
	if !isTuple {
		return fmt.Errorf("expected tuple type but got %T", clarityValue)
	}

	// Extract the event name
	eventName, err := extractEventName(eventTuple)
	if err != nil {
		return fmt.Errorf("failed to extract event name: %w", err)
	}

	// Check if this is a post-message event
	if eventName != "post-message" {
		return fmt.Errorf("expected 'post-message' event but got '%s'", eventName)
	}

	// Extract the core message fields
	msgData, err := extractMessageData(eventTuple)
	if err != nil {
		return fmt.Errorf("failed to extract message data: %w", err)
	}

	// Convert txId to bytes
	txIdBytes, err := hex.DecodeString(strings.TrimPrefix(txId, "0x"))
	if err != nil {
		return fmt.Errorf("failed to decode transaction ID hex: %w", err)
	}

	// Convert timestamp to int64 with overflow check
	if timestamp > maxInt64 {
		return fmt.Errorf("timestamp %d exceeds maximum int64 value", timestamp)
	}

	// Create the complete MessagePublication
	msgPub := &common.MessagePublication{
		TxID:             txIdBytes,
		Timestamp:        time.Unix(int64(timestamp), 0), // #nosec G115 -- checked above
		EmitterChain:     vaa.ChainIDStacks,
		EmitterAddress:   msgData.EmitterAddress,
		ConsistencyLevel: msgData.ConsistencyLevel,
		Nonce:            msgData.Nonce,
		Payload:          msgData.Payload,
		Sequence:         msgData.Sequence,
		IsReobservation:  isReobservation,
	}

	// Submit the message to the channel for processing
	w.msgC <- msgPub

	return nil
}

/// HELPERS

func isTransactionResultCommitted(result map[string]interface{}) bool {
	if result == nil {
		return false
	}

	response, parsed := result["Response"].(map[string]interface{})
	if !parsed {
		return false
	}

	committed, parsed := response["committed"].(bool)
	return parsed && committed
}

// Extracts the event name from an event tuple
func extractEventName(eventTuple *Tuple) (string, error) {
	eventNameVal, ok := eventTuple.Values["event"]
	if !ok {
		return "", fmt.Errorf("missing 'event' field in tuple")
	}

	// Check if event is a StringASCII or StringUTF8
	var eventName string
	if strVal, ok := eventNameVal.(*StringASCII); ok {
		eventName = strVal.Value
	} else if strVal, ok := eventNameVal.(*StringUTF8); ok {
		eventName = strVal.Value
	} else {
		return "", fmt.Errorf("'event' field is not a string type: %T", eventNameVal)
	}

	return eventName, nil
}

// Extracts core message fields from an event tuple
func extractMessageData(eventTuple *Tuple) (*MessageData, error) {
	// Get the data field which should contain the message
	dataVal, ok := eventTuple.Values["data"]
	if !ok {
		return nil, fmt.Errorf("missing 'data' field in tuple")
	}

	// Cast data to tuple
	msgTuple, ok := dataVal.(*Tuple)
	if !ok {
		return nil, fmt.Errorf("'data' field is not a tuple: %T", dataVal)
	}

	// Extract message fields
	emitterVal, ok := msgTuple.Values["emitter"]
	if !ok {
		return nil, fmt.Errorf("missing 'emitter' field in message")
	}

	emitterBuffer, ok := emitterVal.(*ClarityBuffer)
	if !ok || emitterBuffer.Length != 32 {
		return nil, fmt.Errorf("'emitter' field is not a 32-byte buffer: %T", emitterVal)
	}

	// Convert buffer to wormhole address
	emitterAddr := vaa.Address{}
	copy(emitterAddr[:], emitterBuffer.Data[:])

	nonceVal, ok := msgTuple.Values["nonce"]
	if !ok {
		return nil, fmt.Errorf("missing 'nonce' field in message")
	}

	nonceUint, ok := nonceVal.(*UInt128)
	if !ok || nonceUint.Value.Cmp(maxUint32BigInt) > 0 {
		return nil, fmt.Errorf("invalid 'nonce' field: %T", nonceVal)
	}

	sequenceVal, ok := msgTuple.Values["sequence"]
	if !ok {
		return nil, fmt.Errorf("missing 'sequence' field in message")
	}

	sequenceUint, ok := sequenceVal.(*UInt128)
	if !ok || sequenceUint.Value.Cmp(maxUint64BigInt) > 0 {
		return nil, fmt.Errorf("invalid 'sequence' field: %T", sequenceVal)
	}

	consistencyLevelVal, ok := msgTuple.Values["consistency-level"]
	if !ok {
		return nil, fmt.Errorf("missing 'consistency-level' field in message")
	}

	consistencyLevelUint, ok := consistencyLevelVal.(*UInt128)
	if !ok || consistencyLevelUint.Value.Cmp(maxUint8BigInt) > 0 {
		return nil, fmt.Errorf("invalid 'consistency-level' field: %T", consistencyLevelVal)
	}

	payloadVal, ok := msgTuple.Values["payload"]
	if !ok {
		return nil, fmt.Errorf("missing 'payload' field in message")
	}

	payload, ok := payloadVal.(*ClarityBuffer)
	if !ok || payload.Length > 8192 {
		return nil, fmt.Errorf("invalid 'payload' field: %T", payloadVal)
	}

	// Extract values with safe conversions (already validated above against max values)
	nonceValue := nonceUint.Value.Uint64()
	consistencyLevelValue := consistencyLevelUint.Value.Uint64()

	// Return just the core message fields
	return &MessageData{
		EmitterAddress:   emitterAddr,
		Nonce:            uint32(nonceValue), // #nosec G115 -- validated against maxUint32BigInt above
		Sequence:         sequenceUint.Value.Uint64(),
		ConsistencyLevel: uint8(consistencyLevelValue), // #nosec G115 -- validated against maxUint8BigInt above
		Payload:          payload.Data,
	}, nil
}
