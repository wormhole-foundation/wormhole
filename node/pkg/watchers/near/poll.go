package near

import (
	"context"
	"errors"

	"github.com/certusone/wormhole/node/pkg/watchers/near/nearapi"
	"go.uber.org/zap"
)

// fetchAndParseChunk goes through all transactions in a chunk and returns a list of transactionProcessingJob
func (e *Watcher) fetchAndParseChunk(logger *zap.Logger, ctx context.Context, chunkHeader nearapi.ChunkHeader) ([]*transactionProcessingJob, error) {
	logger.Debug("near.fetchAndParseChunk", zap.String("chunk_hash", chunkHeader.Hash))

	chunk, err := e.nearAPI.GetChunk(ctx, chunkHeader)
	if err != nil {
		return nil, err
	}

	txns := chunk.Transactions()

	result := make([]*transactionProcessingJob, len(txns))
	for i, tx := range txns {
		result[i] = newTransactionProcessingJob(tx.Hash, tx.SignerId, false)
	}
	return result, nil
}

// recursivelyReadFinalizedBlocks walks back the blockchain from the startBlock (inclusive)
// until it reaches a block of height stopHeight or less (exclusive). Chunks from all these blocks are put
// into chunkSink with the chunks from the oldest block first
// if there is an error while walking back the chain, no chunks will be returned
func (e *Watcher) recursivelyReadFinalizedBlocks(logger *zap.Logger, ctx context.Context, startBlock nearapi.Block, stopHeight uint64, chunkSink chan<- nearapi.ChunkHeader, recursionDepth uint) error {

	// SECURITY: Sanity checks for the block header
	if startBlock.Header.Hash == "" || startBlock.Header.Height == 0 || startBlock.Header.PrevBlockHash == "" {
		return errors.New("json parse error")
	}

	// SECURITY: We know that this block is finalized because it is a parent of a finalized block.
	e.finalizer.setFinalized(startBlock.Header)

	logger.Debug(
		"block_polled",
		zap.String("log_msg_type", "block_poll"),
		zap.Uint64("height", startBlock.Header.Height),
		zap.String("block_hash", startBlock.Header.Hash),
	)

	// we want to avoid going too far back because that would increase the likelihood of error somewhere in the recursion stack.
	// If we go back too far, we just report the error and terminate early.
	if recursionDepth > maxFallBehindBlocks {
		e.eventChan <- EVENT_NEAR_WATCHER_TOO_FAR_BEHIND
		return errors.New("recursivelyReadFinalizedBlocks: maxFallBehindBlocks")
	}

	// recursion + stop condition
	if startBlock.Header.Height-1 > stopHeight {

		prevBlock, err := e.nearAPI.GetBlock(ctx, startBlock.Header.PrevBlockHash)
		if err != nil {
			return err
		}
		err = e.recursivelyReadFinalizedBlocks(logger, ctx, prevBlock, stopHeight, chunkSink, recursionDepth+1)
		if err != nil {
			// only log error because we still want to process the blocks up until the one that made the error
			logger.Debug("recursivelyReadFinalizedBlocks error", zap.Error(err))
		}
	}

	chunks := startBlock.ChunkHashes()
	// process chunks after recursion such that youngest chunks get processed first
	for i := 0; i < len(chunks); i++ {
		chunkSink <- chunks[i]
	}
	return nil
}

// ReadFinalChunksSince polls the NEAR blockchain for new blocks with height > startHeight, parses out the chunks and places
// them into `chunkSink` in the order they were recorded on the blockchain
// returns the height of the latest final block
func (e *Watcher) ReadFinalChunksSince(logger *zap.Logger, ctx context.Context, startHeight uint64, chunkSink chan<- nearapi.ChunkHeader) (newestFinalHeight uint64, err error) {

	finalBlock, err := e.nearAPI.GetFinalBlock(ctx)
	if err != nil {
		// We can suppress this error because this is equivalent to saying that we haven't found any blocks since.
		return startHeight, nil
	}

	newestFinalHeight = finalBlock.Header.Height

	if newestFinalHeight > startHeight {

		logger.Debug(
			"polling_attempt",
			zap.String("log_msg_type", "polling_attempt"),
			zap.Uint64("previous_height", startHeight),
			zap.Uint64("newest_final_height", newestFinalHeight),
		)

		err = e.recursivelyReadFinalizedBlocks(logger, ctx, finalBlock, startHeight, chunkSink, 0)
		if err != nil {
			logger.Debug("recursivelyReadFinalizedBlocks error", zap.Error(err))
			return startHeight, err
		}
	}

	return newestFinalHeight, nil
}
