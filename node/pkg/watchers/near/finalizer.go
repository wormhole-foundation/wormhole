package near

import (
	"context"
	"errors"

	"github.com/certusone/wormhole/node/pkg/watchers/near/nearapi"
	lru "github.com/hashicorp/golang-lru"
	"go.uber.org/zap"
)

type Finalizer struct {
	// internal cache of which blocks have been finalized, mapping blockHack => blockTimestamp.
	// The timestamp is persisted because we'll need it again later.
	// thread-safe
	finalizedBlocksCache *lru.Cache
	nearAPI              nearapi.NearApi
	eventChan            chan eventType
	mainnet              bool
}

func newFinalizer(eventChan chan eventType, nearAPI nearapi.NearApi, mainnet bool) Finalizer {
	finalizedBlocksCache, _ := lru.New(workerCountTxProcessing * quequeSize)

	return Finalizer{
		finalizedBlocksCache,
		nearAPI,
		eventChan,
		mainnet,
	}
}

func (f Finalizer) isFinalizedCached(logger *zap.Logger, ctx context.Context, blockHash string) (bool, nearapi.BlockHeader) {
	if nearapi.IsValidHash(blockHash) != nil {
		// SECURITY defense-in-depth: block hashes should be 44 characters in length
		logger.Error("blockHash length is not the expected length", zap.String("blockHash", blockHash))
		return false, nearapi.BlockHeader{}
	}

	if b, ok := f.finalizedBlocksCache.Get(blockHash); ok {
		blockHeader := b.(nearapi.BlockHeader)
		// SECURITY In blocks < 74473147 message timestamps were computed differently and we don't want to re-observe these messages
		if !f.mainnet || blockHeader.Height > 74473147 {
			return true, blockHeader
		}
	}

	return false, nearapi.BlockHeader{}
}

// isFinalized() checks if a block is finalized by looking at the local cache first. If therer is an error during execution it returns false.
// If it is not found in the cache, we walk forward up to nearBlockchainMaxGaps blocks by height,
// starting at the block's height+2 and check if their value of "last_final_block" matches
// the block in question.
// we start at height+2 because NEAR consensus takes at least two blocks to reach finality.
func (f Finalizer) isFinalized(logger *zap.Logger, ctx context.Context, queriedBlockHash string) (bool, nearapi.BlockHeader) {

	logger.Debug("checking block finalization", zap.String("method", "isFinalized"), zap.String("parameters", queriedBlockHash))

	// check cache first
	if ok, block := f.isFinalizedCached(logger, ctx, queriedBlockHash); ok {
		return true, block
	}

	logger.Debug("block finalization cache miss", zap.String("method", "isFinalized"), zap.String("parameters", queriedBlockHash))
	f.eventChan <- EVENT_FINALIZED_CACHE_MISS

	queriedBlock, err := f.nearAPI.GetBlock(ctx, queriedBlockHash)
	if err != nil {
		return false, nearapi.BlockHeader{}
	}
	startingBlockHeight := queriedBlock.Header.Height

	for i := 0; i < nearBlockchainMaxGaps; i++ {
		block, err := f.nearAPI.GetBlockByHeight(ctx, startingBlockHeight+uint64(2+i))
		if err != nil {
			break
		}

		// SECURITY defense-in-depth check
		if block.Header.Height != startingBlockHeight+uint64(2+i) {
			// SECURITY violation: Block height is different than what we queried for
			logger.Panic("NEAR RPC Inconsistent", zap.String("inconsistency", "block_height_result_different_from_query"))
		}

		someFinalBlockHash := block.Header.LastFinalBlock

		// SECURITY defense-in-depth check
		if someFinalBlockHash == "" || block.Header.Height == 0 || block.Header.Timestamp == 0 {
			break
		}

		if queriedBlockHash == someFinalBlockHash {
			f.setFinalized(logger, ctx, queriedBlock.Header)
			// block was marked as finalized in the cache, so this should succeed now.
			// We don't return directly because setFinalized() contains some sanity checks.
			return f.isFinalizedCached(logger, ctx, queriedBlockHash)
		}
	}
	// it seems like the block has not been finalized yet
	return false, nearapi.BlockHeader{}
}

func (f Finalizer) setFinalized(logger *zap.Logger, ctx context.Context, blockHeader nearapi.BlockHeader) {

	// SECURITY defense-in-depth: don't cache obviously corrupted data.
	if nearapi.IsValidHash(blockHeader.Hash) != nil || blockHeader.Timestamp == 0 || blockHeader.Height == 0 {
		return
	}

	f.finalizedBlocksCache.Add(blockHeader.Hash, blockHeader)
}

func (f Finalizer) setFinalizedHash(logger *zap.Logger, ctx context.Context, blockHash string) error { //nolint Ignore unused function for now; might come in handy later
	logger.Debug("setFinalizedHash()", zap.String("blockHash", blockHash))
	// SECURITY defense-in-depth: don't cache obviously corrupted data.
	if nearapi.IsValidHash(blockHash) != nil {
		return errors.New("blockHash length is not the expected length")
	}

	block, err := f.nearAPI.GetBlock(ctx, blockHash)
	if err != nil {
		return err
	}

	f.finalizedBlocksCache.Add(blockHash, block.Header)
	return nil
}
