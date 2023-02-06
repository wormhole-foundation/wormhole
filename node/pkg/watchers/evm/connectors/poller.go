package connectors

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	ethEvent "github.com/ethereum/go-ethereum/event"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethHexUtils "github.com/ethereum/go-ethereum/common/hexutil"

	"go.uber.org/zap"
)

type PollFinalizer interface {
	IsBlockFinalized(ctx context.Context, block *NewBlock) (bool, error)
}

// BlockPollConnector polls for new blocks instead of subscribing when using SubscribeForBlocks. It allows to specify a
// finalizer which will be used to only return finalized blocks on subscriptions.
type BlockPollConnector struct {
	Connector
	Delay             time.Duration
	useFinalized      bool
	publishSafeBlocks bool
	finalizer         PollFinalizer
	blockFeed         ethEvent.Feed
	errFeed           ethEvent.Feed
}

func NewBlockPollConnector(ctx context.Context, baseConnector Connector, finalizer PollFinalizer, delay time.Duration, useFinalized bool, publishSafeBlocks bool) (*BlockPollConnector, error) {
	if publishSafeBlocks && !useFinalized {
		return nil, fmt.Errorf("publishSafeBlocks may only be enabled if useFinalized is enabled")
	}
	connector := &BlockPollConnector{
		Connector:         baseConnector,
		Delay:             delay,
		useFinalized:      useFinalized,
		publishSafeBlocks: publishSafeBlocks,
		finalizer:         finalizer,
	}
	err := supervisor.Run(ctx, "blockPoller", common.WrapWithScissors(connector.runFromSupervisor, "blockPoller"))
	if err != nil {
		return nil, err
	}
	return connector, nil
}

func (b *BlockPollConnector) runFromSupervisor(ctx context.Context) error {
	logger := supervisor.Logger(ctx).With(zap.String("eth_network", b.Connector.NetworkName()))
	supervisor.Signal(ctx, supervisor.SignalHealthy)
	return b.run(ctx, logger)
}

func (b *BlockPollConnector) run(ctx context.Context, logger *zap.Logger) error {
	lastBlock, err := b.getBlock(ctx, logger, nil, false)
	if err != nil {
		return err
	}

	var lastSafeBlock *NewBlock
	if b.publishSafeBlocks {
		lastSafeBlock, err = b.getBlock(ctx, logger, nil, true)
		if err != nil {
			return err
		}
	}

	timer := time.NewTimer(time.Millisecond) // Start immediately.

	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			for count := 0; count < 3; count++ {
				lastBlock, err = b.pollBlocks(ctx, logger, lastBlock, false)
				if err == nil {
					break
				}
				logger.Error("polling of block encountered an error", zap.Error(err))

				// Wait an interval before trying again. We stay in this loop so that we
				// try up to three times before causing the watcher to restart.
				time.Sleep(b.Delay)
			}

			if err == nil && b.publishSafeBlocks {
				for count := 0; count < 3; count++ {
					lastSafeBlock, err = b.pollBlocks(ctx, logger, lastSafeBlock, true)
					if err == nil {
						break
					}
					logger.Error("polling of safe block encountered an error", zap.Error(err))

					// Same wait as above.
					time.Sleep(b.Delay)
				}
			}

			if err != nil {
				b.errFeed.Send(fmt.Sprint("polling encountered an error: ", err))
			}
			timer.Reset(b.Delay)
		}
	}
}

func (b *BlockPollConnector) pollBlocks(ctx context.Context, logger *zap.Logger, lastBlock *NewBlock, safe bool) (lastPublishedBlock *NewBlock, retErr error) {
	// Some of the testnet providers (like the one we are using for Arbitrum) limit how many transactions we can do. When that happens, the call hangs.
	// Use a timeout so that the call will fail and the runable will get restarted. This should not happen in mainnet, but if it does, we will need to
	// investigate why the runable is dying and fix the underlying problem.

	lastPublishedBlock = lastBlock

	// Fetch the latest block on the chain
	// We could do this on every iteration such that if a new block is created while this function is being executed,
	// it would automatically fetch new blocks but in order to reduce API load this will be done on the next iteration.
	latestBlock, err := b.getBlockWithTimeout(ctx, logger, nil, safe)
	if err != nil {
		logger.Error("failed to look up latest block",
			zap.Uint64("lastSeenBlock", lastBlock.Number.Uint64()), zap.Error(err))
		return lastPublishedBlock, fmt.Errorf("failed to look up latest block: %w", err)
	}
	for {
		if lastPublishedBlock.Number.Cmp(latestBlock.Number) >= 0 {
			// We have to wait for a new block to become available
			return
		}

		// Try to fetch the next block between lastBlock and latestBlock
		nextBlockNumber := new(big.Int).Add(lastPublishedBlock.Number, big.NewInt(1))
		block, err := b.getBlockWithTimeout(ctx, logger, nextBlockNumber, safe)
		if err != nil {
			logger.Error("failed to fetch next block",
				zap.Uint64("block", nextBlockNumber.Uint64()), zap.Error(err))
			return lastPublishedBlock, fmt.Errorf("failed to fetch next block (%d): %w", nextBlockNumber.Uint64(), err)
		}

		if b.finalizer != nil {
			finalized, err := b.isBlockFinalizedWithTimeout(ctx, block)
			if err != nil {
				logger.Error("failed to check block finalization",
					zap.Uint64("block", block.Number.Uint64()), zap.Error(err))
				return lastPublishedBlock, fmt.Errorf("failed to check block finalization (%d): %w", block.Number.Uint64(), err)
			}

			if !finalized {
				break
			}
		}

		b.blockFeed.Send(block)
		lastPublishedBlock = block
	}

	return
}

func (b *BlockPollConnector) getBlockWithTimeout(ctx context.Context, logger *zap.Logger, blockNumber *big.Int, safe bool) (*NewBlock, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return b.getBlock(timeout, logger, blockNumber, safe)
}

func (b *BlockPollConnector) isBlockFinalizedWithTimeout(ctx context.Context, block *NewBlock) (bool, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return b.finalizer.IsBlockFinalized(timeout, block)
}

func (b *BlockPollConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
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

func (b *BlockPollConnector) getBlock(ctx context.Context, logger *zap.Logger, number *big.Int, safe bool) (*NewBlock, error) {
	return getBlock(ctx, logger, b.Connector, number, b.useFinalized, safe)
}

// getBlock is a free function that can be called from other connectors to get a single block.
func getBlock(ctx context.Context, logger *zap.Logger, conn Connector, number *big.Int, useFinalized bool, safe bool) (*NewBlock, error) {
	var numStr string
	if number != nil {
		numStr = ethHexUtils.EncodeBig(number)
	} else if useFinalized {
		if safe {
			numStr = "safe"
		} else {
			numStr = "finalized"
		}
	} else {
		numStr = "latest"
	}

	type Marshaller struct {
		Number *ethHexUtils.Big
		Hash   ethCommon.Hash `json:"hash"`

		// L1BlockNumber is the L1 block number in which an Arbitrum batch containing this block was submitted.
		// This field is only populated when connecting to Arbitrum.
		L1BlockNumber *ethHexUtils.Big
	}

	var m Marshaller
	err := conn.RawCallContext(ctx, &m, "eth_getBlockByNumber", numStr, false)
	if err != nil {
		logger.Error("failed to get block",
			zap.String("requested_block", numStr), zap.Error(err))
		return nil, err
	}
	if m.Number == nil {
		logger.Error("failed to unmarshal block",
			zap.String("requested_block", numStr),
		)
		return nil, fmt.Errorf("failed to unmarshal block: Number is nil")
	}
	n := big.Int(*m.Number)

	var l1bn *big.Int
	if m.L1BlockNumber != nil {
		bn := big.Int(*m.L1BlockNumber)
		l1bn = &bn
	}

	return &NewBlock{
		Number:        &n,
		Hash:          m.Hash,
		L1BlockNumber: l1bn,
		Safe:          safe,
	}, nil
}
