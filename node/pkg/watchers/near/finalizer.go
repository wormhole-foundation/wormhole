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
	finalizedBlocksCache, _ := lru.New(workerCountTxProcessing * queueSize)

	return Finalizer{
		finalizedBlocksCache,
		nearAPI,
		eventChan,
		mainnet,
	}
}

func (f Finalizer) isFinalizedCached(logger *zap.Logger, blockHash string) (nearapi.BlockHeader, bool) {
	if err := nearapi.IsWellFormedHash(blockHash); err != nil {
		// SECURITY defense-in-depth: check if block hash is well-formed
		logger.Error("blockHash invalid", zap.String("error_type", "invalid_hash"), zap.String("blockHash", blockHash), zap.Error(err))
		return nearapi.BlockHeader{}, false
	}

	if b, ok := f.finalizedBlocksCache.Get(blockHash); ok {
		blockHeader := b.(nearapi.BlockHeader) //nolint:forcetypeassert
		// SECURITY In blocks < 74473147 message timestamps were computed differently and we don't want to re-observe these messages
		if !f.mainnet || blockHeader.Height > 74473147 {
			return blockHeader, true
		}
	}

	return nearapi.BlockHeader{}, false
}

// isFinalized() checks if a block is finalized by looking at the local cache first. If there is an error during execution it returns false.
// If it is not found in the cache, we walk forward up to nearBlockchainMaxGaps blocks by height,
// starting at the block's height+2 and check if their value of "last_final_block" matches
// the block in question.
// we start at height+2 because NEAR consensus takes at least two blocks to reach finality.
func (f Finalizer) isFinalized(logger *zap.Logger, ctx context.Context, queriedBlockHash string) (nearapi.BlockHeader, bool) {

	logger.Debug("checking block finalization", zap.String("method", "isFinalized"), zap.String("parameters", queriedBlockHash))

	// check cache first
	if block, ok := f.isFinalizedCached(logger, queriedBlockHash); ok {
		return block, true
	}

	logger.Debug("block finalization cache miss", zap.String("method", "isFinalized"), zap.String("parameters", queriedBlockHash))
	f.eventChan <- EVENT_FINALIZED_CACHE_MISS

	queriedBlock, err := f.nearAPI.GetBlock(ctx, queriedBlockHash)
	if err != nil {
		return nearapi.BlockHeader{}, false
	}
	startingBlockHeight := queriedBlock.Header.Height

	for i := 0; i < nearBlockchainMaxGaps; i++ {
		// we start at height+2 because NEAR consensus takes at least two blocks to reach finality.
		blockHeightToQuery := startingBlockHeight + uint64(2+i) // #nosec G115 -- nearBlockchainMaxGaps is 5
		block, err := f.nearAPI.GetBlockByHeight(ctx, blockHeightToQuery)
		if err != nil {
			break
		}

		// SECURITY defense-in-depth check
		if block.Header.Height != blockHeightToQuery {
			// SECURITY violation: Block height is different than what we queried for
			logger.Panic("NEAR RPC Inconsistent", zap.String("error_type", "nearapi_inconsistent"), zap.String("inconsistency", "block_height_result_different_from_query"))
		}

		someFinalBlockHash := block.Header.LastFinalBlock

		// SECURITY defense-in-depth check
		if someFinalBlockHash == "" || block.Header.Height == 0 || block.Header.Timestamp == 0 {
			break
		}

		if queriedBlockHash == someFinalBlockHash {
			f.setFinalized(queriedBlock.Header)
			// block was marked as finalized in the cache, so this should succeed now.
			// We don't return directly because setFinalized() contains some sanity checks.
			return f.isFinalizedCached(logger, queriedBlockHash)
		}
	}
	// it seems like the block has not been finalized yet
	return nearapi.BlockHeader{}, false
}

func (f Finalizer) setFinalized(blockHeader nearapi.BlockHeader) {

	// SECURITY defense-in-depth: don't cache obviously corrupted data.
	if nearapi.IsWellFormedHash(blockHeader.Hash) != nil || blockHeader.Timestamp == 0 || blockHeader.Height == 0 {
		return
	}

	f.finalizedBlocksCache.Add(blockHeader.Hash, blockHeader)
}

func (f Finalizer) setFinalizedHash(logger *zap.Logger, ctx context.Context, blockHash string) error { //nolint Ignore unused function for now; might come in handy later
	logger.Debug("setFinalizedHash()", zap.String("blockHash", blockHash))
	// SECURITY defense-in-depth: don't cache obviously corrupted data.
	if nearapi.IsWellFormedHash(blockHash) != nil {
		return errors.New("blockHash length is not the expected length")
	}

	block, err := f.nearAPI.GetBlock(ctx, blockHash)
	if err != nil {
		return err
	}

	f.finalizedBlocksCache.Add(blockHash, block.Header)
	return nil
}
