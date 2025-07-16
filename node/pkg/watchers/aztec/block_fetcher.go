package aztec

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"go.uber.org/zap"
)

// BlockFetcher defines the interface for retrieving Aztec chain data
type BlockFetcher interface {
	FetchPublicLogs(ctx context.Context, fromBlock, toBlock int) ([]ExtendedPublicLog, error)
	FetchBlock(ctx context.Context, blockNumber int) (BlockInfo, error)
}

// aztecBlockFetcher is the implementation of BlockFetcher
type aztecBlockFetcher struct {
	rpcClient *rpc.Client
	logger    *zap.Logger
}

// NewAztecBlockFetcher creates a new block fetcher
func NewAztecBlockFetcher(ctx context.Context, rpcURL string, logger *zap.Logger) (BlockFetcher, error) {
	// Create a new RPC client
	client, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %v", err)
	}

	return &aztecBlockFetcher{
		rpcClient: client,
		logger:    logger,
	}, nil
}

// Guardians should set ARCHIVER_MAX_LOGS to a high number in the Aztec node
// since we don't support pagination in the client.
// FetchPublicLogs gets logs for a specific block range
func (f *aztecBlockFetcher) FetchPublicLogs(ctx context.Context, fromBlock, toBlock int) ([]ExtendedPublicLog, error) {
	f.logger.Debug("Fetching logs",
		zap.Int("fromBlock", fromBlock),
		zap.Int("toBlock", toBlock))

	// Prepare the filter arguments
	logFilter := map[string]interface{}{
		"fromBlock": fromBlock,
		"toBlock":   toBlock,
	}

	// Create a variable to hold the result
	var result struct {
		Logs       []ExtendedPublicLog `json:"logs"`
		MaxLogsHit bool                `json:"maxLogsHit"`
	}

	// Make the RPC call
	err := f.rpcClient.CallContext(ctx, &result, "node_getPublicLogs", logFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch public logs: %v", err)
	}

	return result.Logs, nil
}

// FetchBlock gets info for a specific block
func (f *aztecBlockFetcher) FetchBlock(ctx context.Context, blockNumber int) (BlockInfo, error) {
	// Create variables to hold the result
	var blockResult BlockResult

	// Make the RPC call
	err := f.rpcClient.CallContext(ctx, &blockResult, "node_getBlock", blockNumber)
	if err != nil {
		return BlockInfo{}, fmt.Errorf("failed to fetch block info: %v", err)
	}

	info := BlockInfo{}

	// Set the block hash using the archive root
	info.archiveRoot = blockResult.Archive.Root

	// Set the parent hash using lastArchive.root
	info.parentArchiveRoot = blockResult.Header.LastArchive.Root

	// Get the timestamp from global variables (remove 0x prefix and convert from hex)
	timestampHex := strings.TrimPrefix(blockResult.Header.GlobalVariables.Timestamp, "0x")
	if timestampHex == "" {
		// Handle empty timestamp (typically for genesis block)
		if blockNumber == 0 {
			// Use a default timestamp for genesis block
			info.Timestamp = 0 // Or any appropriate value
			f.logger.Debug("Genesis block has no timestamp, using default value")
		} else {
			// Use current time as fallback for non-genesis blocks
			unixTime := time.Now().Unix()
			if unixTime < 0 {
				// Handle negative timestamp - this shouldn't happen in practice
				// but gosec wants us to check for it
				info.Timestamp = 0
			} else {
				info.Timestamp = uint64(unixTime)
			}
			f.logger.Warn("Block has empty timestamp, using current time",
				zap.Int("blockNumber", blockNumber))
		}
	} else {
		// Parse the timestamp normally
		timestamp, err := strconv.ParseUint(timestampHex, 16, 64)
		if err != nil {
			return BlockInfo{}, fmt.Errorf("failed parsing timestamp: %v", err)
		}
		info.Timestamp = timestamp
	}

	// Default transaction hash
	info.TxHash = "0x0"

	// Store transaction hashes by index for log processing
	info.TxHashesByIndex = make(map[int]string)
	for i, txEffect := range blockResult.Body.TxEffects {
		info.TxHashesByIndex[i] = txEffect.TxHash
	}

	// Log the block hash and parent hash for debugging
	f.logger.Debug("Fetched block info",
		zap.Int("blockNumber", blockNumber),
		zap.String("archiveRoot", info.archiveRoot),
		zap.String("parentArchiveRoot", info.parentArchiveRoot),
		zap.Int("txCount", len(blockResult.Body.TxEffects)))

	return info, nil
}
