package connectors

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	ethEvent "github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"

	ethereum "github.com/ethereum/go-ethereum"

	"go.uber.org/zap"
)

// BatchPollConnector uses batch requests to poll for latest, safe and finalized blocks.
type BatchPollConnector struct {
	Connector
	Delay     time.Duration
	blockFeed ethEvent.Feed
	errFeed   ethEvent.Feed
	batchData []BatchEntry
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

const MAX_GAP_BATCH_SIZE uint64 = 5

func NewBatchPollConnector(ctx context.Context, baseConnector Connector, delay time.Duration) (*BatchPollConnector, error) {
	// Create the batch data in the order we want to report them to the watcher, so finalized is most important, latest is least.
	batchData := []BatchEntry{
		{tag: "finalized", finality: Finalized},
		{tag: "safe", finality: Safe},
		{tag: "latest", finality: Latest},
	}

	connector := &BatchPollConnector{
		Connector: baseConnector,
		Delay:     delay,
		batchData: batchData,
	}
	err := supervisor.Run(ctx, "batchPoller", common.WrapWithScissors(connector.runFromSupervisor, "batchPoller"))
	if err != nil {
		return nil, err
	}
	return connector, nil
}

func (b *BatchPollConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	sub := NewPollSubscription()
	blockSub := b.blockFeed.Subscribe(sink)

	// The feed library does not support error forwarding, so we're emulating that using a custom subscription and
	// an error feed. The feed library can't handle interfaces which is why we post strings.
	innerErrSink := make(chan string, 10)
	innerErrSub := b.errFeed.Subscribe(innerErrSink)

	common.RunWithScissors(ctx, errC, "block_poll_subscribe_for_blocks", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				blockSub.Unsubscribe()
				innerErrSub.Unsubscribe()
				return nil
			case <-sub.quit:
				blockSub.Unsubscribe()
				innerErrSub.Unsubscribe()
				sub.unsubDone <- struct{}{}
				return nil
			case v := <-innerErrSink:
				sub.err <- fmt.Errorf(v)
			}
		}
	})
	return sub, nil
}

func (b *BatchPollConnector) runFromSupervisor(ctx context.Context) error {
	logger := supervisor.Logger(ctx).With(zap.String("eth_network", b.Connector.NetworkName()))
	supervisor.Signal(ctx, supervisor.SignalHealthy)
	return b.run(ctx, logger)
}

func (b *BatchPollConnector) run(ctx context.Context, logger *zap.Logger) error {
	// Get the initial blocks.
	lastBlocks, err := b.getBlocks(ctx, logger)
	if err != nil {
		return err
	}

	timer := time.NewTimer(b.Delay)
	defer timer.Stop()

	errCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			lastBlocks, err = b.pollBlocks(ctx, logger, lastBlocks)
			if err != nil {
				errCount++
				logger.Error("batch polling encountered an error", zap.Int("errCount", errCount), zap.Error(err))
				if errCount > 3 {
					b.errFeed.Send(fmt.Sprint("polling encountered an error: ", err))
					errCount = 0
				}
			} else {
				errCount = 0
			}

			timer.Reset(b.Delay)
		}
	}
}

// pollBlocks polls for the latest blocks (finalized, safe and latest), compares them to the last ones, and publishes any new ones.
// In the case of an error, it returns the last blocks that were passed in, otherwise it returns the new blocks.
func (b *BatchPollConnector) pollBlocks(ctx context.Context, logger *zap.Logger, prevBlocks Blocks) (Blocks, error) {
	newBlocks, err := b.getBlocks(ctx, logger)
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
				gapBlocks, err := b.getBlockRange(ctx, logger, blockNum, batchSize, b.batchData[idx].finality)
				if err != nil {
					// We don't return an error here because we want to go on and check the other finalities.
					logger.Error("failed to get gap blocks", zap.Stringer("finality", b.batchData[idx].finality), zap.Error(err))
					errorFound = true
				} else {
					// Play out the blocks in this batch. If the block number is zero, that means we failed to retrieve it, so we should stop there.
					for _, block := range gapBlocks {
						if block.Number.Uint64() == 0 {
							errorFound = true
							break
						}

						b.blockFeed.Send(block)
						lastPublishedBlock = block
					}
				}

				blockNum += batchSize
			}

			if !errorFound {
				// The original value of newBlocks is still good.
				b.blockFeed.Send(newBlock)
			} else {
				newBlocks[idx] = lastPublishedBlock
			}
		} else if newBlock.Number.Cmp(prevBlocks[idx].Number) < 0 {
			logger.Debug("latest block number went backwards, ignoring it", zap.Stringer("finality", b.batchData[idx].finality), zap.Any("new", newBlock.Number), zap.Any("prev", prevBlocks[idx].Number))
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
		m := &result.result
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
	for idx := 0; idx < int(numBlocks); idx++ {
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
	for idx, result := range results {
		if result.err != nil {
			logger.Error("failed to get block", zap.Int("idx", idx), zap.Stringer("finality", finality), zap.Error(result.err))
			return nil, err
		}

		var n big.Int
		m := &result.result
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
