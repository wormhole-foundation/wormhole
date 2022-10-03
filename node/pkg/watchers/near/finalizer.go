package near

import (
	"context"

	"github.com/certusone/wormhole/node/pkg/watchers/near/nearapi"
	lru "github.com/hashicorp/golang-lru"
	"go.uber.org/zap"
)

type Finalizer struct {
	// internal cache of which blocks have been finalized, mapping blockHack => blockTimestamp.
	// The timestamp is persisted because we'll need it again later.
	// thread-safe
	finalizedBlocksCache *lru.Cache
	nearAPI              nearapi.NearAPI
	eventChan            chan eventType
	mainnet              bool
}

func newFinalizer(eventChan chan eventType, nearAPI nearapi.NearAPI, mainnet bool) Finalizer {
	finalizedBlocksCache, _ := lru.New(workerCountTxProcessing * quequeSize)

	return Finalizer{
		finalizedBlocksCache,
		nearAPI,
		eventChan,
		mainnet,
	}
}

func (f Finalizer) isFinalizedCached(logger *zap.Logger, block_hash string) (bool, nearapi.BlockHeader) {
	if len(block_hash) != 44 {
		// SECURITY defense-in-depth: block hashes should be 44 characters in length
		logger.Error("blockHash length != 44", zap.String("block_hash", block_hash))
		return false, nearapi.BlockHeader{}
	}

	// check if block_hash is in cache already and if so, return early
	if b, ok := f.finalizedBlocksCache.Get(block_hash); ok {
		block := b.(nearapi.BlockHeader)
		return true, block
	}

	return false, nearapi.BlockHeader{}
}

// isFinalized() checks if a block is finalized by looking at the local cache first. If therer is an error during execution it returns false.
// If it is not found in the cache, we walk forward up to nearBlockchainMaxGaps blocks by height,
// starting at the block's height+2 and check if their value of "last_final_block" matches
// the block in question.
// we start at height+2 because NEAR consensus takes at least two blocks to reach finality.
func (f Finalizer) isFinalized(logger *zap.Logger, ctx context.Context, blockHash string) (bool, nearapi.BlockHeader) {

	logger.Debug("checking block finalization", zap.String("method", "isFinalized"), zap.String("parameters", blockHash))

	// check cache first
	if ok, block := f.isFinalizedCached(logger, blockHash); ok {
		return true, block
	}

	logger.Debug("block finalization cache miss", zap.String("method", "isFinalized"), zap.String("parameters", blockHash))
	f.eventChan <- EVENT_FINALIZED_CACHE_MISS

	startingBlock, err := f.nearAPI.GetBlock(ctx, blockHash)
	if err != nil {
		return false, nearapi.BlockHeader{}
	}
	startingBlockHeight := startingBlock.Header.Height

	for i := 0; i < nearBlockchainMaxGaps; i++ {
		block, err := f.nearAPI.GetBlockByHeight(ctx, startingBlockHeight+uint64(2+i))
		if err != nil {
			break
		}
		blockHeader := block.Header
		someFinalBlockHash := block.Header.LastFinalBlock

		// SECURITY check that return values are not invalid.
		if someFinalBlockHash == "" || blockHeader.Height == 0 || blockHeader.Timestamp == 0 {
			break
		}
		if blockHeader.Height != startingBlockHeight+uint64(2+i) {
			// SECURITY violation: Block height is different than what we queried for
			logger.Panic("NEAR RPC Inconsistent", zap.String("inconsistency", "block_height_result_different_from_query"))
		}
		f.setFinalized(blockHeader)

		if blockHash == someFinalBlockHash {
			// block was marked as finalized in the cache, so this should succeed now.
			// We don't return directly because setFinalized() contains some sanity checks.
			return f.isFinalizedCached(logger, blockHash)
		}
	}
	// it seems like the block has not been finalized yet
	return false, nearapi.BlockHeader{}
}

func (f Finalizer) setFinalized(blockHeader nearapi.BlockHeader) {

	// SECURITY defense-in-depth: don't cache obviously corrupted data.
	if len(blockHeader.Hash) != nearapi.BlockHashLen || blockHeader.Timestamp == 0 || blockHeader.Height == 0 {
		return
	}

	// SECURITY In blocks < 74473147 message timestamps were computed differently and we don't want to re-observe these messages
	if f.mainnet && blockHeader.Height < 74473147 {
		return
	}

	f.finalizedBlocksCache.Add(blockHeader.Hash, blockHeader)
}
