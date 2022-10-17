package connectors

import (
	"context"
	"fmt"
	"math/big"
	"time"

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
	Delay        time.Duration
	useFinalized bool
	finalizer    PollFinalizer
	blockFeed    ethEvent.Feed
	errFeed      ethEvent.Feed
}

func NewBlockPollConnector(ctx context.Context, baseConnector Connector, finalizer PollFinalizer, delay time.Duration, useFinalized bool) (*BlockPollConnector, error) {
	connector := &BlockPollConnector{
		Connector:    baseConnector,
		Delay:        delay,
		useFinalized: useFinalized,
		finalizer:    finalizer,
	}
	err := supervisor.Run(ctx, "blockPoller", connector.run)
	if err != nil {
		return nil, err
	}
	return connector, nil
}

func (b *BlockPollConnector) run(ctx context.Context) error {
	logger := supervisor.Logger(ctx).With(zap.String("eth_network", b.Connector.NetworkName()))

	lastBlock, err := b.getBlock(ctx, logger, nil)
	if err != nil {
		return err
	}

	timer := time.NewTimer(time.Millisecond) // Start immediately.
	supervisor.Signal(ctx, supervisor.SignalHealthy)

	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			for count := 0; count < 3; count++ {
				lastBlock, err = b.pollBlocks(ctx, logger, lastBlock)
				if err == nil {
					break
				}
				logger.Error("polling encountered an error", zap.Error(err))
			}

			if err != nil {
				b.errFeed.Send("polling encountered an error")
			}
			timer.Reset(b.Delay)
		}
	}
}

func (b *BlockPollConnector) pollBlocks(ctx context.Context, logger *zap.Logger, lastBlock *NewBlock) (lastPublishedBlock *NewBlock, retErr error) {
	// Some of the testnet providers (like the one we are using for Arbitrum) limit how many transactions we can do. When that happens, the call hangs.
	// Use a timeout so that the call will fail and the runable will get restarted. This should not happen in mainnet, but if it does, we will need to
	// investigate why the runable is dying and fix the underlying problem.

	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	lastPublishedBlock = lastBlock

	// Fetch the latest block on the chain
	// We could do this on every iteration such that if a new block is created while this function is being executed,
	// it would automatically fetch new blocks but in order to reduce API load this will be done on the next iteration.
	latestBlock, err := b.getBlock(timeout, logger, nil)
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
		block, err := b.getBlock(timeout, logger, nextBlockNumber)
		if err != nil {
			logger.Error("failed to fetch next block",
				zap.Uint64("block", nextBlockNumber.Uint64()), zap.Error(err))
			return lastPublishedBlock, fmt.Errorf("failed to fetch next block (%d): %w", nextBlockNumber.Uint64(), err)
		}

		if b.finalizer != nil {
			finalized, err := b.finalizer.IsBlockFinalized(timeout, block)
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

func (b *BlockPollConnector) SubscribeForBlocks(ctx context.Context, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	sub := NewPollSubscription()
	blockSub := b.blockFeed.Subscribe(sink)

	// The feed library does not support error forwarding, so we're emulating that using a custom subscription and
	// an error feed. The feed library can't handle interfaces which is why we post strings.
	innerErrSink := make(chan string, 10)
	innerErrSub := b.errFeed.Subscribe(innerErrSink)

	go func() {
		for {
			select {
			case <-ctx.Done():
				blockSub.Unsubscribe()
				innerErrSub.Unsubscribe()
				return
			case <-sub.quit:
				blockSub.Unsubscribe()
				innerErrSub.Unsubscribe()
				sub.unsubDone <- struct{}{}
				return
			case v := <-innerErrSink:
				sub.err <- fmt.Errorf(v)
			}
		}
	}()
	return sub, nil
}

func (b *BlockPollConnector) getBlock(ctx context.Context, logger *zap.Logger, number *big.Int) (*NewBlock, error) {
	var numStr string
	if number != nil {
		numStr = ethHexUtils.EncodeBig(number)
	} else if b.useFinalized {
		numStr = "finalized"
	} else {
		numStr = "latest"
	}

	type Marshaller struct {
		Number     *ethHexUtils.Big
		Hash       ethCommon.Hash `json:"hash"`
		Difficulty *ethHexUtils.Big
	}

	var m Marshaller
	err := b.Connector.RawCallContext(ctx, &m, "eth_getBlockByNumber", numStr, false)
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
	return &NewBlock{
		Number: &n,
		Hash:   m.Hash,
	}, nil
}
