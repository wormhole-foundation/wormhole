package connectors

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	ethereum "github.com/ethereum/go-ethereum"

	"go.uber.org/zap"
)

// BatchPollConnector uses batch requests to poll for latest, safe and finalized blocks.
type BatchPollConnector struct {
	Connector
	logger       *zap.Logger
	Delay        time.Duration
	batchData    []BatchEntry
	generateSafe bool
}

type (
	Blocks []*NewBlock

	BatchEntry struct {
		tag      string
		finality FinalityLevel
	}

	BatchResult struct {
		result BlockMarshaller
		err    error
	}
)

// MAX_GAP_BATCH_SIZE specifies the maximum number of blocks to be requested at once when gap filling.
const MAX_GAP_BATCH_SIZE uint64 = 5

func NewBatchPollConnector(ctx context.Context, logger *zap.Logger, baseConnector Connector, safeSupported bool, delay time.Duration) *BatchPollConnector {
	// Create the batch data in the order we want to report them to the watcher. We always do finalized, but only do safe if requested.
	batchData := []BatchEntry{
		{tag: "finalized", finality: Finalized},
	}

	if safeSupported {
		batchData = append(batchData, BatchEntry{tag: "safe", finality: Safe})
	}

	connector := &BatchPollConnector{
		Connector:    baseConnector,
		logger:       logger,
		Delay:        delay,
		batchData:    batchData,
		generateSafe: !safeSupported,
	}

	return connector
}

func (b *BatchPollConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	// Use the standard geth head sink to get latest blocks. We do this so that we will be notified of rollbacks. The following document
	// indicates that the subscription will receive a replay of all blocks affected by a rollback. This is important for latest because the
	// timestamp cache needs to be updated on a rollback. We can only consider polling for latest if we can guarantee that we won't miss rollbacks.
	// https://ethereum.org/en/developers/tutorials/using-websockets/#subscription-types
	headSink := make(chan *ethTypes.Header, 2)
	headerSubscription, err := b.Connector.SubscribeNewHead(ctx, headSink)
	if err != nil {
		return headerSubscription, fmt.Errorf("failed to subscribe for latest blocks: %w", err)
	}

	// Get the initial blocks.
	lastBlocks, err := b.getBlocks(ctx, b.logger)
	if err != nil {
		b.logger.Error("failed to get initial blocks", zap.Error(err))
		return headerSubscription, fmt.Errorf("failed to get initial blocks: %w", err)
	}

	errCount := 0

	// Publish the initial finalized and safe blocks so we have a starting point for reobservation requests.
	for idx, block := range lastBlocks {
		b.logger.Info(fmt.Sprintf("publishing initial %s block", b.batchData[idx].finality), zap.Uint64("initial_block", block.Number.Uint64()))
		sink <- block
		if b.generateSafe && b.batchData[idx].finality == Finalized {
			safe := block.Copy(Safe)
			b.logger.Info("publishing generated initial safe block", zap.Uint64("initial_block", safe.Number.Uint64()))
			sink <- safe
		}
	}

	common.RunWithScissors(ctx, errC, "block_poll_subscribe_for_blocks", func(ctx context.Context) error {
		timer := time.NewTimer(b.Delay)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-timer.C:
				lastBlocks, err = b.pollBlocks(ctx, sink, lastBlocks)
				if err != nil {
					errCount++
					b.logger.Error("batch polling encountered an error", zap.Int("errCount", errCount), zap.Error(err))
					if errCount > 3 {
						errC <- fmt.Errorf("polling encountered too many errors: %w", err)
						return nil
					}
				} else if errCount != 0 {
					errCount = 0
				}
				timer.Reset(b.Delay)
			case ev := <-headSink:
				if ev == nil {
					b.logger.Error("new latest header event is nil")
					continue
				}
				if ev.Number == nil {
					b.logger.Error("new latest header block number is nil")
					continue
				}
				sink <- &NewBlock{
					Number:   ev.Number,
					Time:     ev.Time,
					Hash:     ev.Hash(),
					Finality: Latest,
				}
			}
		}
	})

	return headerSubscription, nil
}

func (b *BatchPollConnector) GetLatest(ctx context.Context) (latest, finalized, safe uint64, err error) {
	block, err := GetBlockByFinality(ctx, b.Connector, Latest)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get latest block: %w", err)
	}
	latest = block.Number.Uint64()

	block, err = GetBlockByFinality(ctx, b.Connector, Finalized)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get finalized block: %w", err)
	}
	finalized = block.Number.Uint64()

	if b.generateSafe {
		safe = finalized
	} else {
		block, err = GetBlockByFinality(ctx, b.Connector, Safe)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to get safe block: %w", err)
		}
		safe = block.Number.Uint64()
	}

	return
}

// pollBlocks polls for the latest blocks (finalized, safe and latest), compares them to the last ones, and publishes any new ones.
// In the case of an error, it returns the last blocks that were passed in, otherwise it returns the new blocks.
func (b *BatchPollConnector) pollBlocks(ctx context.Context, sink chan<- *NewBlock, prevBlocks Blocks) (Blocks, error) {
	newBlocks, err := b.getBlocks(ctx, b.logger)
	if err != nil {
		return prevBlocks, err
	}

	if len(newBlocks) != len(prevBlocks) {
		panic(fmt.Sprintf("getBlocks returned %d entries when there should be %d", len(newBlocks), len(prevBlocks)))
	}

	for idx, newBlock := range newBlocks {
		if newBlock.Number.Cmp(prevBlocks[idx].Number) > 0 {
			// If there is a gap between prev and new, we have to look up the hashes for the missing ones. Do that in batches.
			newBlockNum := newBlock.Number.Uint64()
			blockNum := prevBlocks[idx].Number.Uint64() + 1
			errorFound := false
			lastPublishedBlock := prevBlocks[idx]
			for blockNum < newBlockNum && !errorFound {
				batchSize := newBlockNum - blockNum
				if batchSize > MAX_GAP_BATCH_SIZE {
					batchSize = MAX_GAP_BATCH_SIZE
				}
				gapBlocks, err := b.getBlockRange(ctx, b.logger, blockNum, batchSize, b.batchData[idx].finality)
				if err != nil {
					// We don't return an error here because we want to go on and check the other finalities.
					b.logger.Error("failed to get gap blocks", zap.Stringer("finality", b.batchData[idx].finality), zap.Error(err))
					errorFound = true
				} else {
					// Play out the blocks in this batch. If the block number is zero, that means we failed to retrieve it, so we should stop there.
					for _, block := range gapBlocks {
						if block.Number.Uint64() == 0 {
							errorFound = true
							break
						}
						sink <- block
						if b.generateSafe && b.batchData[idx].finality == Finalized {
							sink <- block.Copy(Safe)
						}
						lastPublishedBlock = block
					}
				}

				blockNum += batchSize
			}

			if !errorFound {
				// The original value of newBlocks is still good.
				sink <- newBlock
				if b.generateSafe && b.batchData[idx].finality == Finalized {
					sink <- newBlock.Copy(Safe)
				}
			} else {
				newBlocks[idx] = lastPublishedBlock
			}
		} else if newBlock.Number.Cmp(prevBlocks[idx].Number) < 0 {
			b.logger.Debug("latest block number went backwards, ignoring it", zap.Stringer("finality", b.batchData[idx].finality), zap.Any("new", newBlock.Number), zap.Any("prev", prevBlocks[idx].Number))
			newBlocks[idx] = prevBlocks[idx]
		}
	}

	return newBlocks, nil
}

// getBlocks gets the current batch of configured blocks (finalized, safe, latest).
func (b *BatchPollConnector) getBlocks(ctx context.Context, logger *zap.Logger) (Blocks, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	batch := make([]rpc.BatchElem, len(b.batchData))
	results := make([]BatchResult, len(b.batchData))
	for idx, bd := range b.batchData {
		batch[idx] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args: []interface{}{
				bd.tag,
				false, // no full transaction details
			},
			Result: &results[idx].result,
			Error:  results[idx].err,
		}
	}

	err := b.Connector.RawBatchCallContext(timeout, batch)
	if err != nil {
		logger.Error("failed to get blocks", zap.Error(err))
		return nil, err
	}

	ret := make(Blocks, len(b.batchData))
	for idx, result := range results {
		finality := b.batchData[idx].finality
		if result.err != nil {
			logger.Error("failed to get block", zap.Stringer("finality", finality), zap.Error(result.err))
			return nil, err
		}

		var n big.Int
		m := &results[idx].result
		if m.Number == nil {
			logger.Debug("number is nil, treating as zero", zap.Stringer("finality", finality), zap.String("tag", b.batchData[idx].tag))
		} else {
			n = big.Int(*m.Number)
		}

		var l1bn *big.Int
		if m.L1BlockNumber != nil {
			bn := big.Int(*m.L1BlockNumber)
			l1bn = &bn
		}

		ret[idx] = &NewBlock{
			Number:        &n,
			Time:          uint64(m.Time),
			Hash:          m.Hash,
			L1BlockNumber: l1bn,
			Finality:      finality,
		}
	}

	return ret, nil
}

// getBlockRange gets a range of blocks, starting at blockNum, including the next numBlocks. It passes back an array of those blocks.
func (b *BatchPollConnector) getBlockRange(ctx context.Context, logger *zap.Logger, blockNum uint64, numBlocks uint64, finality FinalityLevel) (Blocks, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	batch := make([]rpc.BatchElem, numBlocks)
	results := make([]BatchResult, numBlocks)
	for idx := uint64(0); idx < numBlocks; idx++ {
		batch[idx] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args: []interface{}{
				"0x" + fmt.Sprintf("%x", blockNum),
				false, // no full transaction details
			},
			Result: &results[idx].result,
			Error:  results[idx].err,
		}
		blockNum++
	}

	err := b.Connector.RawBatchCallContext(timeout, batch)
	if err != nil {
		logger.Error("failed to get blocks", zap.Error(err))
		return nil, err
	}

	ret := make(Blocks, numBlocks)
	for idx := range results {
		if results[idx].err != nil {
			logger.Error("failed to get block", zap.Int("idx", idx), zap.Stringer("finality", finality), zap.Error(results[idx].err))
			return nil, err
		}

		var n big.Int
		m := &results[idx].result
		if m.Number == nil {
			logger.Debug("number is nil, treating as zero", zap.Stringer("finality", finality))
		} else {
			n = big.Int(*m.Number)
		}

		var l1bn *big.Int
		if m.L1BlockNumber != nil {
			bn := big.Int(*m.L1BlockNumber)
			l1bn = &bn
		}

		ret[idx] = &NewBlock{
			Number:        &n,
			Time:          uint64(m.Time),
			Hash:          m.Hash,
			L1BlockNumber: l1bn,
			Finality:      finality,
		}
	}

	return ret, nil
}
