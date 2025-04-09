package aztec

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Configuration constants
const (
	// Time intervals
	logProcessingInterval = 10 * time.Second

	// Processing parameters
	defaultBatchSize  = 1
	payloadInitialCap = 13

	// Default starting block
	defaultStartBlock = 0
)

// Watcher monitors the Aztec blockchain for message publications
type Watcher struct {
	// Chain identification
	chainID   vaa.ChainID
	networkID string

	// Connection details
	rpcURL          string
	contractAddress string

	// Communication channels
	msgC     chan<- *common.MessagePublication
	obsvReqC <-chan *gossipv1.ObservationRequest

	// Service state
	readinessSync      readiness.Component
	lastProcessedBlock int
}

// metrics for monitoring
var (
	aztecMessagesConfirmed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_aztec_observations_confirmed_total",
			Help: "Total number of verified observations found for the chain",
		}, []string{"chain_name"})
)

// NewWatcher creates a new Aztec watcher
func NewWatcher(
	chainID vaa.ChainID,
	networkID watchers.NetworkID,
	rpcURL string,
	contractAddress string,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		chainID:            chainID,
		networkID:          string(networkID),
		rpcURL:             rpcURL,
		contractAddress:    contractAddress,
		msgC:               msgC,
		obsvReqC:           obsvReqC,
		readinessSync:      common.MustConvertChainIdToReadinessSyncing(chainID),
		lastProcessedBlock: defaultStartBlock,
	}
}

// Run starts the watcher service and handles the main event loop
func (w *Watcher) Run(ctx context.Context) error {
	logger := supervisor.Logger(ctx)
	logger.Info("Starting Aztec watcher",
		zap.String("rpc", w.rpcURL),
		zap.String("contract", w.contractAddress))

	// Create an error channel and ticker
	errC := make(chan error)
	defer close(errC)

	// Signal that basic initialization is complete
	readiness.SetReady(w.readinessSync)

	// Signal to the supervisor that this runnable has finished initialization
	supervisor.Signal(ctx, supervisor.SignalHealthy)

	// Start the single block processing goroutine
	common.RunWithScissors(ctx, errC, "aztec_events", func(ctx context.Context) error {
		logger.Info("Starting Aztec event processor")

		for {
			select {
			case err := <-errC:
				logger.Error("Worker error detected", zap.Error(err))
				return fmt.Errorf("worker died: %w", err)

			case <-ctx.Done():
				logger.Info("Context done, shutting down")
				return ctx.Err()

			default:
				// Wait before processing more blocks
				time.Sleep(logProcessingInterval)

				// Check for and process new blocks
				if err := w.fetchAndProcessBlocks(ctx, logger); err != nil {
					logger.Error("Error processing blocks", zap.Error(err))
					// Continue instead of returning to maintain service
				}
			}
		}
	})

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// fetchAndProcessBlocks checks for new blocks and processes them if found
func (w *Watcher) fetchAndProcessBlocks(ctx context.Context, logger *zap.Logger) error {
	// Get the latest block number
	latestBlock, err := w.fetchLatestBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("error getting latest block: %w", err)
	}

	// Only process if there are new blocks
	if w.lastProcessedBlock >= latestBlock {
		logger.Debug("No new blocks to process",
			zap.Int("latest", latestBlock),
			zap.Int("lastProcessed", w.lastProcessedBlock))
		return nil
	}

	// Log that we found new blocks to process
	logger.Info("Processing new blocks",
		zap.Int("from", w.lastProcessedBlock),
		zap.Int("to", latestBlock))

	// Process blocks in batches
	return w.processBlockRange(ctx, logger, w.lastProcessedBlock, latestBlock)
}

// processBlockRange processes a range of blocks in batches
func (w *Watcher) processBlockRange(ctx context.Context, logger *zap.Logger, startBlock, endBlock int) error {
	for fromBlock := startBlock; fromBlock <= endBlock; fromBlock += defaultBatchSize {
		// Calculate the end of this batch

		toBlock := min(fromBlock+defaultBatchSize-1, endBlock)

		// Handle quirk for single block ranges (Aztec specific)
		if fromBlock == toBlock {
			toBlock = fromBlock + 1
		}

		// Process this batch
		if err := w.processBatch(ctx, logger, fromBlock, toBlock); err != nil {
			logger.Error("Failed to process batch",
				zap.Int("fromBlock", fromBlock),
				zap.Int("toBlock", toBlock),
				zap.Error(err))
			// Continue with next batch instead of terminating
			continue
		}

		// Update the last processed block
		w.lastProcessedBlock = toBlock
	}

	logger.Info("Completed processing blocks", zap.Int("up_to", w.lastProcessedBlock))
	return nil
}

// processBatch handles fetching and processing logs for a batch of blocks
func (w *Watcher) processBatch(ctx context.Context, logger *zap.Logger, fromBlock, toBlock int) error {
	logger.Info("Processing block batch",
		zap.Int("fromBlock", fromBlock),
		zap.Int("toBlock", toBlock))

	// Get logs for this block range
	logs, err := w.fetchPublicLogs(ctx, fromBlock, toBlock)
	if err != nil {
		return fmt.Errorf("failed to fetch logs: %w", err)
	}

	logger.Info("Processing logs",
		zap.Int("count", len(logs)),
		zap.Int("fromBlock", fromBlock),
		zap.Int("toBlock", toBlock))

	// Process each log
	for _, extLog := range logs {
		if err := w.processLog(ctx, logger, extLog); err != nil {
			logger.Error("Failed to process log",
				zap.Int("block", extLog.ID.BlockNumber),
				zap.Error(err))
			// Continue processing other logs
		}
	}

	return nil
}

// processLog handles processing a single log entry
func (w *Watcher) processLog(ctx context.Context, logger *zap.Logger, extLog ExtendedPublicLog) error {
	// Log basic info
	logger.Info("Log found",
		zap.Int("block", extLog.ID.BlockNumber),
		zap.String("contract", extLog.Log.ContractAddress))

	// Skip empty logs
	if len(extLog.Log.Log) == 0 {
		return nil
	}

	// Extract event parameters
	params, err := w.parseLogParameters(logger, extLog.Log.Log)
	if err != nil {
		return fmt.Errorf("failed to parse log parameters: %w", err)
	}

	// Create message payload
	payload := w.createPayload(logger, extLog.Log.Log[4:])

	// Get block info for transaction ID and timestamp
	blockInfo, err := w.fetchBlockInfo(ctx, extLog.ID.BlockNumber)
	if err != nil {
		logger.Warn("Failed to get block info, using defaults", zap.Error(err))
		blockInfo = BlockInfo{
			TxHash:    "0x0000000000000000000000000000000000000000000000000000000000000000",
			Timestamp: uint64(time.Now().Unix()),
		}
	}

	// Create and publish observation
	return w.publishObservation(logger, params, payload, blockInfo)
}

// parseLogParameters extracts parameters from a log entry
func (w *Watcher) parseLogParameters(logger *zap.Logger, logEntries []string) (LogParameters, error) {
	if len(logEntries) < 4 {
		return LogParameters{}, fmt.Errorf("log has insufficient entries: %d", len(logEntries))
	}

	// First value is the sender
	sender := logEntries[0]
	var senderAddress vaa.Address
	copy(senderAddress[:], sender)
	logger.Info("Sender", zap.String("value", sender))

	// Parse sequence
	sequence, err := hexToUint64(logEntries[1])
	if err != nil {
		return LogParameters{}, fmt.Errorf("failed to parse sequence: %w", err)
	}
	logger.Info("Sequence", zap.Uint64("value", sequence))

	// Parse nonce
	nonce, err := hexToUint64(logEntries[2])
	if err != nil {
		return LogParameters{}, fmt.Errorf("failed to parse nonce: %w", err)
	}
	logger.Info("Nonce", zap.Uint64("value", nonce))

	// Parse consistency level
	consistencyLevel, err := hexToUint64(logEntries[3])
	if err != nil {
		return LogParameters{}, fmt.Errorf("failed to parse consistencyLevel: %w", err)
	}
	logger.Info("ConsistencyLevel", zap.Uint64("value", consistencyLevel))

	return LogParameters{
		SenderAddress:    senderAddress,
		Sequence:         sequence,
		Nonce:            uint32(nonce),
		ConsistencyLevel: uint8(consistencyLevel),
	}, nil
}

// createPayload processes log entries into a byte payload
func (w *Watcher) createPayload(logger *zap.Logger, logEntries []string) []byte {
	payload := make([]byte, 0, payloadInitialCap)

	for i, entry := range logEntries {
		hexStr := strings.TrimPrefix(entry, "0x")

		// Try to decode as hex
		bytes, err := hex.DecodeString(hexStr)
		if err != nil {
			logger.Warn("Failed to decode hex", zap.String("entry", entry), zap.Error(err))
			continue
		}

		// Add to payload
		payload = append(payload, bytes...)

		// Try to interpret as a string for logging
		w.logInterpretedValue(logger, i+4, bytes, entry)
	}

	return payload
}

// logInterpretedValue attempts to interpret bytes as string or number for logging
func (w *Watcher) logInterpretedValue(logger *zap.Logger, index int, bytes []byte, rawHex string) {
	// Trim leading null bytes
	startIndex := 0
	for startIndex < len(bytes) && bytes[startIndex] == 0 {
		startIndex++
	}
	trimmedBytes := bytes[startIndex:]

	// Check if it's a printable string
	if str := string(trimmedBytes); isPrintableString(str) {
		logger.Info("Field as string", zap.Int("index", index), zap.String("value", str))
	} else {
		// Fall back to numeric representation
		logger.Info("Field as number", zap.Int("index", index), zap.String("value", rawHex))
	}
}

// publishObservation creates and publishes a message observation
func (w *Watcher) publishObservation(logger *zap.Logger, params LogParameters, payload []byte, blockInfo BlockInfo) error {
	// Convert transaction hash to byte array for txID
	txID, err := hex.DecodeString(strings.TrimPrefix(blockInfo.TxHash, "0x"))
	if err != nil {
		logger.Error("Failed to decode transaction hash", zap.Error(err))
		// Fall back to default
		txID = []byte{0x0}
	}

	// Create the observation
	observation := &common.MessagePublication{
		TxID:             txID,
		Timestamp:        time.Unix(int64(blockInfo.Timestamp), 0),
		Nonce:            params.Nonce,
		Sequence:         params.Sequence,
		EmitterChain:     w.chainID,
		EmitterAddress:   params.SenderAddress,
		Payload:          payload,
		ConsistencyLevel: params.ConsistencyLevel,
		IsReobservation:  false,
	}

	// Increment metrics
	aztecMessagesConfirmed.WithLabelValues(w.networkID).Inc()

	// Log the observation
	logger.Info("Message observed",
		zap.String("txHash", observation.TxIDString()),
		zap.Time("timestamp", observation.Timestamp),
		zap.Uint32("nonce", observation.Nonce),
		zap.Uint64("sequence", observation.Sequence),
		zap.Stringer("emitter_chain", observation.EmitterChain),
		zap.Stringer("emitter_address", observation.EmitterAddress),
		zap.Binary("payload", observation.Payload),
		zap.Uint8("consistencyLevel", observation.ConsistencyLevel),
	)

	// Send to the message channel
	w.msgC <- observation

	return nil
}

// fetchPublicLogs retrieves logs for a specific block range
func (w *Watcher) fetchPublicLogs(ctx context.Context, fromBlock, toBlock int) ([]ExtendedPublicLog, error) {
	// Create log filter parameter
	logFilter := map[string]any{
		"fromBlock": fromBlock,
		"toBlock":   toBlock,
	}

	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "node_getPublicLogs",
		"params":  []any{logFilter},
		"id":      1,
	}

	// Send the JSON-RPC request
	responseBody, err := w.sendJSONRPCRequest(ctx, payload)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var response JsonRpcResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse logs response: %w", err)
	}

	return response.Result.Logs, nil
}

// fetchLatestBlockNumber gets the current height of the blockchain
func (w *Watcher) fetchLatestBlockNumber(ctx context.Context) (int, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "node_getBlockNumber",
		"params":  []any{},
		"id":      1,
	}

	// Send the request
	responseBody, err := w.sendJSONRPCRequest(ctx, payload)
	if err != nil {
		return 0, err
	}

	// Parse the response
	var response struct {
		Result json.RawMessage `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return 0, fmt.Errorf("failed to parse block number response: %w", err)
	}

	return parseBlockNumber(response.Result)
}

// fetchBlockInfo gets details of a specific block
func (w *Watcher) fetchBlockInfo(ctx context.Context, blockNumber int) (BlockInfo, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "node_getBlock",
		"params":  []any{blockNumber},
		"id":      1,
	}

	// Send the request
	responseBody, err := w.sendJSONRPCRequest(ctx, payload)
	if err != nil {
		return BlockInfo{}, err
	}

	// Parse the response
	var response BlockResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return BlockInfo{}, fmt.Errorf("failed to parse block response: %w", err)
	}

	// Extract the necessary information from the block
	info := BlockInfo{}

	// Get the timestamp from global variables (remove 0x prefix and convert from hex)
	timestampHex := strings.TrimPrefix(response.Result.Header.GlobalVariables.Timestamp, "0x")
	timestamp, err := strconv.ParseUint(timestampHex, 16, 64)
	if err != nil {
		return BlockInfo{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	info.Timestamp = timestamp

	// Get the transaction hash from the first transaction in the block (if available)
	if len(response.Result.Body.TxEffects) > 0 {
		info.TxHash = response.Result.Body.TxEffects[0].TxHash
	} else {
		// If no transactions, use the block's archive root as a fallback identifier
		info.TxHash = response.Result.Archive.Root
	}

	return info, nil
}

// sendJSONRPCRequest sends a JSON-RPC request and returns the response body
func (w *Watcher) sendJSONRPCRequest(ctx context.Context, payload map[string]any) ([]byte, error) {
	// Marshal the payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", w.rpcURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	return body, nil
}

// parseBlockNumber handles different formats of block number in responses
func parseBlockNumber(rawMessage json.RawMessage) (int, error) {
	// Try to unmarshal as string first (hex format)
	var hexStr string
	if err := json.Unmarshal(rawMessage, &hexStr); err == nil {
		// It's a hex string like "0x123"
		parsedNum, err := strconv.ParseInt(hexStr, 0, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse hex block number: %w", err)
		}
		return int(parsedNum), nil
	}

	// Try to unmarshal as number
	var num float64
	if err := json.Unmarshal(rawMessage, &num); err != nil {
		return 0, fmt.Errorf("block number is neither string nor number: %w", err)
	}

	return int(num), nil
}

// hexToUint64 converts a hex string to uint64
func hexToUint64(hexStr string) (uint64, error) {
	// Remove "0x" prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")

	// Parse the hex string to uint64
	value, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert hex to uint64: %s, error: %w", hexStr, err)
	}

	return value, nil
}

// isPrintableString checks if a string contains mostly printable ASCII characters
func isPrintableString(s string) bool {
	printable := 0
	for _, r := range s {
		if r >= 32 && r <= 126 {
			printable++
		}
	}
	return printable >= 3 && float64(printable)/float64(len(s)) > 0.5
}

// LogParameters encapsulates the core parameters from a log
type LogParameters struct {
	SenderAddress    vaa.Address
	Sequence         uint64
	Nonce            uint32
	ConsistencyLevel uint8
}

// Helper struct for block information
type BlockInfo struct {
	TxHash    string
	Timestamp uint64
}

// JSON-RPC related structures
type JsonRpcResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Logs       []ExtendedPublicLog `json:"logs"`
		MaxLogsHit bool                `json:"maxLogsHit"`
	} `json:"result"`
}

type BlockResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  BlockResult `json:"result"`
}

type BlockResult struct {
	Archive BlockArchive `json:"archive"`
	Header  BlockHeader  `json:"header"`
	Body    BlockBody    `json:"body"`
}

type BlockArchive struct {
	Root                   string `json:"root"`
	NextAvailableLeafIndex int    `json:"nextAvailableLeafIndex"`
}

type BlockHeader struct {
	GlobalVariables GlobalVariables `json:"globalVariables"`
	// Other header fields omitted for brevity
}

type GlobalVariables struct {
	ChainID     string `json:"chainId"`
	Version     string `json:"version"`
	BlockNumber string `json:"blockNumber"`
	SlotNumber  string `json:"slotNumber"`
	Timestamp   string `json:"timestamp"`
	Coinbase    string `json:"coinbase"`
	// Other global variables omitted for brevity
}

type BlockBody struct {
	TxEffects []TxEffect `json:"txEffects"`
}

type TxEffect struct {
	TxHash string `json:"txHash"`
	// Other tx effect fields omitted for brevity
}

type LogId struct {
	BlockNumber int `json:"blockNumber"`
	TxIndex     int `json:"txIndex"`
	LogIndex    int `json:"logIndex"`
}

type PublicLog struct {
	ContractAddress string   `json:"contractAddress"`
	Log             []string `json:"log"`
}

type ExtendedPublicLog struct {
	ID  LogId     `json:"id"`
	Log PublicLog `json:"log"`
}
