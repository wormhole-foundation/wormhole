package connectors

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/supervisor"

	ethereum "github.com/ethereum/go-ethereum"
	ethHexUtils "github.com/ethereum/go-ethereum/common/hexutil"
	ethEvent "github.com/ethereum/go-ethereum/event"

	"go.uber.org/zap"
)

type PollFinalizer interface {
	IsBlockFinalized(ctx context.Context, block *NewBlock) (bool, error)
}

// BlockPollConnector polls for new blocks instead of subscribing when using SubscribeForBlocks. It allows to specify a
// finalizer which will be used to only return finalized blocks on subscriptions.
type BlockPollConnector struct {
	Connector
	Delay     time.Duration
	finalizer PollFinalizer
	blockFeed ethEvent.Feed
	errFeed   ethEvent.Feed
}

func NewBlockPollConnector(ctx context.Context, baseConnector Connector, finalizer PollFinalizer, delay time.Duration) (*BlockPollConnector, error) {
	if finalizer == nil {
		panic("finalizer must not be nil; Use finalizers.NewDefaultFinalizer() if you want to have no finalizer.")
	}
	connector := &BlockPollConnector{
		Connector: baseConnector,
		Delay:     delay,
		finalizer: finalizer,
	}
	err := supervisor.Run(ctx, "blockPoller", common.WrapWithScissors(connector.runFromSupervisor, "blockPoller"))
	if err != nil {
		return nil, err
	}
	return connector, nil
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

func (b *BlockPollConnector) runFromSupervisor(ctx context.Context) error {
	logger := supervisor.Logger(ctx).With(zap.String("eth_network", b.Connector.NetworkName()))
	supervisor.Signal(ctx, supervisor.SignalHealthy)
	return b.run(ctx, logger)
}

func (b *BlockPollConnector) run(ctx context.Context, logger *zap.Logger) error {
	prevLatest, err := getBlockByTag(ctx, logger, b.Connector, "latest", Latest)
	if err != nil {
		return err
	}

	prevFinalized := &NewBlock{
		Number:   prevLatest.Number,
		Hash:     prevLatest.Hash,
		Finality: Finalized,
	}

	timer := time.NewTimer(b.Delay)
	defer timer.Stop()

	errCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			prevLatest, prevFinalized, err = b.pollBlock(ctx, logger, prevLatest, prevFinalized)
			if err != nil {
				errCount++
				logger.Error("polling encountered an error", zap.Int("errCount", errCount), zap.Error(err))
				if errCount > 3 {
					b.errFeed.Send(fmt.Sprint("polling encountered an error: ", err))
				}
			} else {
				errCount = 0
			}

			timer.Reset(b.Delay)
		}
	}
}

// pollBlock poll for the latest block, compares them to the last one, and publishes any new ones.
// In the case of an error, it returns the last block that were passed in, otherwise it returns the new block.
func (b *BlockPollConnector) pollBlock(ctx context.Context, logger *zap.Logger, prevLatest *NewBlock, prevFinalized *NewBlock) (newLatest *NewBlock, newFinalized *NewBlock, err error) {
	newLatest, err = getBlockByTag(ctx, logger, b.Connector, "latest", Latest)
	if err != nil {
		err = fmt.Errorf("failed to get latest block: %w", err)
		newLatest = prevLatest
		newFinalized = prevFinalized
		return
	}

	// First see if there might be some newly finalized ones to publish
	var block *NewBlock
	if newLatest.Number.Cmp(prevLatest.Number) > 0 {
		// If there is a gap between prev and new, we have to look up the transaction hashes for the missing ones. Do that in batches.
		newBlockNum := newLatest.Number.Uint64()
		for blockNum := prevLatest.Number.Uint64() + 1; blockNum < newBlockNum; blockNum++ {
			block, err = getBlockByNumberUint64(ctx, logger, b.Connector, blockNum, Latest)
			if err != nil {
				err = fmt.Errorf("failed to get gap block: %w", err)
				newLatest = prevLatest
				newFinalized = prevFinalized
				return
			}

			b.blockFeed.Send(block)
		}

		b.blockFeed.Send(newLatest)
	} else if newLatest.Number.Cmp(prevLatest.Number) < 0 {
		logger.Error("latest block number went backwards, ignoring it", zap.Any("newLatest", newLatest), zap.Any("prevLatest", prevLatest))
		newLatest = prevLatest
	}

	newFinalized = prevFinalized
	if newLatest.Number.Cmp(prevFinalized.Number) > 0 {
		var finalized bool
		// If there is a gap between prev and new, we have to look up the transaction hashes for the missing ones. Do that in batches.
		newBlockNum := newLatest.Number.Uint64()
		for blockNum := prevFinalized.Number.Uint64() + 1; blockNum <= newBlockNum; blockNum++ {
			block, err = getBlockByNumberUint64(ctx, logger, b.Connector, blockNum, Finalized)
			if err != nil {
				err = fmt.Errorf("failed to get gap block: %w", err)
				newLatest = prevLatest
				newFinalized = prevFinalized
				return
			}

			finalized, err = b.isBlockFinalized(ctx, block)
			if err != nil {
				err = fmt.Errorf("failed to check finality on block: %w", err)
				newLatest = prevLatest
				newFinalized = prevFinalized
				return
			}

			if !finalized {
				break
			}

			b.blockFeed.Send(block.Copy(Safe))
			b.blockFeed.Send(block.Copy(Finalized))
			newFinalized = block
		}
	}

	return
}

func getBlockByNumberUint64(ctx context.Context, logger *zap.Logger, conn Connector, blockNum uint64, desiredFinality FinalityLevel) (*NewBlock, error) {
	return getBlockByTag(ctx, logger, conn, "0x"+fmt.Sprintf("%x", blockNum), desiredFinality)
}

func getBlockByNumberBigInt(ctx context.Context, logger *zap.Logger, conn Connector, blockNum *big.Int, desiredFinality FinalityLevel) (*NewBlock, error) {
	return getBlockByTag(ctx, logger, conn, ethHexUtils.EncodeBig(blockNum), desiredFinality)
}

func getBlockByTag(ctx context.Context, logger *zap.Logger, conn Connector, tag string, desiredFinality FinalityLevel) (*NewBlock, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var m BlockMarshaller
	err := conn.RawCallContext(timeout, &m, "eth_getBlockByNumber", tag, false)
	if err != nil {
		logger.Error("failed to get block",
			zap.String("requested_block", tag), zap.Error(err))
		return nil, err
	}
	if m.Number == nil {
		logger.Error("failed to unmarshal block",
			zap.String("requested_block", tag),
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
		Finality:      desiredFinality,
	}, nil
}

func (b *BlockPollConnector) isBlockFinalized(ctx context.Context, block *NewBlock) (bool, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return b.finalizer.IsBlockFinalized(timeout, block)
}
