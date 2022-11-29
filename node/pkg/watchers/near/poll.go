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

	var result []*transactionProcessingJob

	chunk, err := e.nearAPI.GetChunk(ctx, chunkHeader)
	if err != nil {
		return nil, err
	}

	txns := chunk.Transactions()

	for _, tx := range txns {
		result = append(result, newTransactionProcessingJob(tx.Hash, tx.SignerId))
	}
	return result, nil
}

// recursivelyReadFinalizedBlocks walks back the blockchain from the startBlock (inclusive)
// until it reaches a block of height stopHeight or less (exclusive). Chunks from all these blocks are put
// into e.chunkProcessingQueque with the chunks from the oldest block first
// if there is an error while walking back the chain, no chunks will be returned
func (e *Watcher) recursivelyReadFinalizedBlocks(logger *zap.Logger, ctx context.Context, startBlock nearapi.Block, stopHeight uint64, chunkSink chan<- nearapi.ChunkHeader, recursionDepth uint) error {

	// SECURITY: Sanity checks for the block header
	if startBlock.Header.Hash == "" || startBlock.Header.Height == 0 || startBlock.Header.PrevBlockHash == "" {
		return errors.New("json parse error")
	}

	// SECURITY: We know that this block is finalized because it is a parent of a finalized block.
	e.finalizer.setFinalized(logger, ctx, startBlock.Header)

	// we want to avoid going too far back because that would increase the likelihood of error somewhere in the recursion stack.
	// If we go back too far, we just report the error and terminate early.
	if recursionDepth > maxFallBehindBlocks {
		e.eventChan <- EVENT_NEAR_WATCHER_TOO_FAR_BEHIND
		return nil
	}

	// recursion + stop condition
	if startBlock.Header.Height-1 > stopHeight {

		prevBlock, err := e.nearAPI.GetBlock(ctx, startBlock.Header.PrevBlockHash)
		if err != nil {
			return err
		}
		err = e.recursivelyReadFinalizedBlocks(logger, ctx, prevBlock, stopHeight, chunkSink, recursionDepth+1)
		if err != nil {
			return err
		}

		logger.Info(
			"block_polled",
			zap.String("log_msg_type", "block_poll"),
			zap.Uint64("height", startBlock.Header.Height),
		)
	}

	chunks := startBlock.ChunkHashes()
	// process chunks after recursion such that youngest chunks get processed first
	for i := 0; i < len(chunks); i++ {
		e.chunkProcessingQueue <- chunks[i]
	}
	return nil
}

// readFinalChunksSince polls the NEAR blockchain for new blocks with height > startHeight, parses out the chunks and places
// them into `chunkSink` in the order they were recorded on the blockchain
// returns the height of the latest final block
func (e *Watcher) ReadFinalChunksSince(logger *zap.Logger, ctx context.Context, startHeight uint64, chunkSink chan<- nearapi.ChunkHeader) (newestFinalHeight uint64, err error) {

	finalBlock, err := e.nearAPI.GetFinalBlock(ctx)
	if err != nil {
		// We can supress this error because this is equivalent to saying that we haven't found any blocks since.
		return startHeight, nil
	}

	newestFinalHeight = finalBlock.Header.Height

	if newestFinalHeight > startHeight {

		logger.Info(
			"polling_attempt",
			zap.String("log_msg_type", "polling_attempt"),
			zap.Uint64("previous_height", startHeight),
			zap.Uint64("newest_final_height", newestFinalHeight),
		)

		err = e.recursivelyReadFinalizedBlocks(logger, ctx, finalBlock, startHeight, chunkSink, 0)
		if err != nil {
			return startHeight, err
		}
	}

	return newestFinalHeight, nil
}
