package evm

import (
	"context"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"

	"go.uber.org/zap"

	ethHexUtil "github.com/ethereum/go-ethereum/common/hexutil"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
)

const CCQ_MAX_BATCH_SIZE = int64(1000)
const CCQ_TIMESTAMP_RANGE_IN_SECONDS = uint64(30 * 60)
const CCQ_BACKFILL_DELAY = 100 * time.Millisecond

type (
	// ccqBackfillRequest represents a request to backfill the cache. It is the payload on the request channel.
	ccqBackfillRequest struct {
		timestamp uint64
	}

	// ccqBatchResult is the result of each query in a batch.
	ccqBatchResult struct {
		result ccqBlockMarshaller
		err    error
	}

	// ccqBlockMarshaller is used to marshal the query results.
	ccqBlockMarshaller struct {
		Number ethHexUtil.Uint64
		Time   ethHexUtil.Uint64 `json:"timestamp"`
		// Hash   ethCommon.Hash    `json:"hash"`
	}
)

// ccqRequestBackfill submits a request to backfill a gap in the timestamp cache. Note that the timestamp passed in should be in seconds, as expected by the timestamp cache.
func (w *Watcher) ccqRequestBackfill(timestamp uint64) {
	select {
	case w.ccqBackfillChannel <- &ccqBackfillRequest{timestamp: timestamp}:
		w.ccqLogger.Debug("published backfill request", zap.Uint64("timestamp", timestamp))
	default:
		// This will get retried next interval.
		w.ccqLogger.Error("failed to post backfill request, will get retried next interval", zap.Uint64("timestamp", timestamp))
	}
}

// ccqBackfillStart initializes the timestamp cache by backfilling some history and starting a routine to handle backfill requests
// when a timestamp is not in the cache. This function does not return errors because we don't want to prevent the watcher from
// coming up if we can't backfill the cache. We just disable backfilling and hope for the best.
func (w *Watcher) ccqBackfillStart(ctx context.Context, errC chan error) {
	if err := w.ccqBackfillInit(ctx); err != nil {
		w.ccqLogger.Error("failed to backfill timestamp cache, disabling backfilling", zap.Error(err))
		w.ccqBackfillCache = false
		return
	}

	common.RunWithScissors(ctx, errC, "ccq_backfiller", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case evt := <-w.ccqBackfillChannel:
				w.ccqPerformBackfill(ctx, evt)
			}
		}
	})
}

// ccqBackfillInit determines the maximum batch size to be used for backfilling the cache. It also loads the initial batch of timestamps.
func (w *Watcher) ccqBackfillInit(ctx context.Context) error {
	// Get the latest block so we can use that as the starting point in our cache.
	latestBlock, err := connectors.GetLatestBlock(ctx, w.ethConn)
	if err != nil {
		return fmt.Errorf("failed to look up latest block: %w", err)
	}
	latestBlockNum := latestBlock.Number.Int64()
	w.ccqLogger.Info("looked up latest block", zap.Int64("latestBlockNum", latestBlockNum), zap.Uint64("timestamp", latestBlock.Time))

	var blocks Blocks
	if w.ccqBatchSize == 0 {
		// Determine the max supported batch size and get the first batch which will start with the latest block and go backwards.
		var err error
		w.ccqBatchSize, blocks, err = ccqBackFillDetermineMaxBatchSize(ctx, w.ccqLogger, w.ethConn, latestBlockNum, CCQ_BACKFILL_DELAY)
		if err != nil {
			return fmt.Errorf("failed to determine max batch size: %w", err)
		}
	} else {
		blocks = append(blocks, Block{BlockNum: latestBlock.Number.Uint64(), Timestamp: latestBlock.Time})
		w.ccqLogger.Info("using existing batch size for timestamp cache", zap.Int64("batchSize", w.ccqBatchSize))
	}

	// We want to start with a half hour in our cache. Get batches until we cover that.
	cutOffTime := latestBlock.Time - CCQ_TIMESTAMP_RANGE_IN_SECONDS
	if latestBlock.Time < CCQ_TIMESTAMP_RANGE_IN_SECONDS {
		// In devnet the timestamps are just integers that start at zero on startup.
		cutOffTime = 0
	}

	if len(blocks) == 0 {
		// This should never happen, but the for loop would panic if it did!
		return fmt.Errorf("list of blocks is empty")
	}

	// Query for more blocks until we go back the desired length of time. The last block in the array will be the oldest, so query starting one before that.
	for blocks[len(blocks)-1].Timestamp > cutOffTime {
		newBlocks, err := w.ccqBackfillGetBlocks(ctx, blocks[len(blocks)-1].BlockNum-1, w.ccqBatchSize)
		if err != nil {
			return fmt.Errorf("failed to get batch starting at %d: %w", blocks[len(blocks)-1].BlockNum-1, err)
		}

		if len(newBlocks) == 0 {
			w.ccqLogger.Warn("failed to read any more blocks, giving up on the backfill")
			break
		}

		blocks = append(blocks, newBlocks...)
		w.ccqLogger.Info("got batch",
			zap.Uint64("oldestBlockNum", newBlocks[len(newBlocks)-1].BlockNum),
			zap.Uint64("latestBlockNum", newBlocks[0].BlockNum),
			zap.Uint64("oldestBlockTimestamp", newBlocks[len(newBlocks)-1].Timestamp),
			zap.Uint64("latestBlockTimestamp", newBlocks[0].Timestamp),
			zap.Stringer("oldestTime", time.Unix(int64(newBlocks[len(newBlocks)-1].Timestamp), 0)), // #nosec G115 -- This conversion is safe indefinitely
			zap.Stringer("latestTime", time.Unix(int64(newBlocks[0].Timestamp), 0)),                // #nosec G115 -- This conversion is safe indefinitely
		)
	}

	w.ccqLogger.Info("adding initial batch to timestamp cache",
		zap.Int64("batchSize", w.ccqBatchSize),
		zap.Int("numBlocks", len(blocks)),
		zap.Uint64("oldestBlockNum", blocks[len(blocks)-1].BlockNum),
		zap.Uint64("latestBlockNum", blocks[0].BlockNum),
		zap.Uint64("oldestBlockTimestamp", blocks[len(blocks)-1].Timestamp),
		zap.Uint64("latestBlockTimestamp", blocks[0].Timestamp),
		zap.Stringer("oldestTime", time.Unix(int64(blocks[len(blocks)-1].Timestamp), 0)), // #nosec G115 -- This conversion is safe indefinitely
		zap.Stringer("latestTime", time.Unix(int64(blocks[0].Timestamp), 0)),             // #nosec G115 -- This conversion is safe indefinitely
	)

	w.ccqTimestampCache.AddBatch(blocks)

	return nil
}

// ccqBackfillConn is defined to allow for testing of ccqBackFillDetermineMaxBatchSize without mocking a full ethereum connection.
type ccqBackfillConn interface {
	RawBatchCallContext(ctx context.Context, b []ethRpc.BatchElem) error
}

// ccqBackFillDetermineMaxBatchSize performs a series of batch queries, starting with a size of 1000 and stepping down by halves, and then back up until we circle in on the maximum batch size supported by the RPC.
func ccqBackFillDetermineMaxBatchSize(ctx context.Context, logger *zap.Logger, conn ccqBackfillConn, latestBlockNum int64, delay time.Duration) (int64, Blocks, error) {
	batchSize := int64(CCQ_MAX_BATCH_SIZE)
	var batch []ethRpc.BatchElem
	var results []ccqBatchResult
	prevFailure := int64(0)
	prevSuccess := int64(0)
	for {
		if latestBlockNum < batchSize {
			batchSize = latestBlockNum
		}
		logger.Info("trying batch size", zap.Int64("batchSize", batchSize))
		batch = make([]ethRpc.BatchElem, batchSize)
		results = make([]ccqBatchResult, batchSize)
		blockNum := latestBlockNum
		for idx := int64(0); idx < batchSize; idx++ {
			batch[idx] = ethRpc.BatchElem{
				Method: "eth_getBlockByNumber",
				Args: []interface{}{
					"0x" + fmt.Sprintf("%x", blockNum),
					false, // no full transaction details
				},
				Result: &results[idx].result,
				Error:  results[idx].err,
			}

			blockNum--
		}

		timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := conn.RawBatchCallContext(timeout, batch)
		cancel()

		if err == nil {
			logger.Info("batch query worked", zap.Int64("batchSize", batchSize))
			if prevFailure == 0 {
				break
			}
			if batchSize+1 >= prevFailure {
				break
			}
			prevSuccess = batchSize
		} else {
			logger.Info("batch query failed", zap.Int64("batchSize", batchSize), zap.Error(err))
			prevFailure = batchSize
		}
		batchSize = (prevFailure + prevSuccess) / 2
		if batchSize == 0 {
			return 0, nil, fmt.Errorf("failed to determine batch size: %w", err)
		}

		time.Sleep(delay)
	}

	// Save the blocks we just retrieved to be used as our starting cache.
	blocks := Blocks{}
	for i := range results {
		if results[i].err != nil {
			return 0, nil, fmt.Errorf("failed to get block: %w", results[i].err)
		}

		m := &results[i].result

		if m.Number != 0 {
			blocks = append(blocks, Block{
				BlockNum: uint64(m.Number),
				// Hash:   m.Hash,
				Timestamp: uint64(m.Time),
			})
		}
	}

	logger.Info("found supported batch size", zap.Int64("batchSize", batchSize), zap.Int("numBlocks", len(blocks)))
	return batchSize, blocks, nil
}

// ccqBackfillGetBlocks gets a range of blocks from the RPC, starting from initialBlockNum and going downward for numBlocks.
func (w *Watcher) ccqBackfillGetBlocks(ctx context.Context, initialBlockNum uint64, numBlocks int64) (Blocks, error) {
	w.ccqLogger.Info("getting batch", zap.Uint64("initialBlockNum", initialBlockNum), zap.Int64("numBlocks", numBlocks))
	batch := make([]ethRpc.BatchElem, numBlocks)
	results := make([]ccqBatchResult, numBlocks)
	blockNum := initialBlockNum
	for idx := int64(0); idx < numBlocks; idx++ {
		batch[idx] = ethRpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args: []interface{}{
				"0x" + fmt.Sprintf("%x", blockNum),
				false, // no full transaction details
			},
			Result: &results[idx].result,
			Error:  results[idx].err,
		}

		blockNum--
	}

	timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	err := w.ethConn.RawBatchCallContext(timeout, batch)
	cancel()
	if err != nil {
		w.ccqLogger.Error("failed to get batch of blocks",
			zap.Uint64("initialBlockNum", initialBlockNum),
			zap.Int64("numBlocks", numBlocks),
			zap.Uint64("finalBlockNum", blockNum),
			zap.Error(err),
		)

		return nil, err
	}

	blocks := Blocks{}
	for i := range results {
		if results[i].err != nil {
			return nil, fmt.Errorf("failed to get block: %w", err)
		}

		m := &results[i].result

		if m.Number != 0 {
			blocks = append(blocks, Block{
				BlockNum: uint64(m.Number),
				// Hash:   m.Hash,
				Timestamp: uint64(m.Time),
			})
		}
	}

	return blocks, nil
}

// ccqPerformBackfill handles a request to backfill the timestamp cache. First it does another lookup to confirm that the backfill is still needed.
// If so, it submits a batch query for all of the requested blocks, up to what will fit in a single batch.
func (w *Watcher) ccqPerformBackfill(ctx context.Context, evt *ccqBackfillRequest) {
	// Things may have changed since the request was posted to the channel. See if we still need to do the backfill.
	firstBlock, lastBlock, found := w.ccqTimestampCache.LookUp(evt.timestamp)
	if found {
		w.ccqLogger.Info("received a backfill request which is now in the cache, ignoring it", zap.Uint64("timestamp", evt.timestamp), zap.Uint64("firstBlock", firstBlock), zap.Uint64("lastBlock", lastBlock))
		return
	}

	numBlocks := int64(lastBlock - firstBlock - 1) // #nosec G115 -- Realistically impossible to overflow
	if numBlocks > w.ccqBatchSize {
		numBlocks = w.ccqBatchSize
	}
	w.ccqLogger.Info("received a backfill request", zap.Uint64("timestamp", evt.timestamp), zap.Uint64("firstBlock", firstBlock), zap.Uint64("lastBlock", lastBlock), zap.Int64("numBlocks", numBlocks))
	blocks, err := w.ccqBackfillGetBlocks(ctx, lastBlock-1, numBlocks)
	if err != nil {
		w.ccqLogger.Error("failed to get backfill batch", zap.Uint64("startingBlock", lastBlock-1), zap.Int64("numBlocks", numBlocks))
		return
	}

	w.ccqLogger.Info("adding backfill batch to timestamp cache",
		zap.Int64("batchSize", w.ccqBatchSize),
		zap.Int("numBlocks", len(blocks)),
		zap.Uint64("oldestBlockNum", blocks[len(blocks)-1].BlockNum),
		zap.Uint64("latestBlockNum", blocks[0].BlockNum),
		zap.Uint64("oldestBlockTimestamp", blocks[len(blocks)-1].Timestamp),
		zap.Uint64("latestBlockTimestamp", blocks[0].Timestamp),
		zap.Stringer("oldestTime", time.Unix(int64(blocks[len(blocks)-1].Timestamp), 0)), // #nosec G115 -- This conversion is safe indefinitely
		zap.Stringer("latestTime", time.Unix(int64(blocks[0].Timestamp), 0)),             // #nosec G115 -- This conversion is safe indefinitely
	)

	w.ccqTimestampCache.AddBatch(blocks)
}
