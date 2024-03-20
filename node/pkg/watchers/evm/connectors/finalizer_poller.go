package connectors

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/supervisor"

	ethereum "github.com/ethereum/go-ethereum"
	ethEvent "github.com/ethereum/go-ethereum/event"

	"go.uber.org/zap"
)

type PollFinalizer interface {
	IsBlockFinalized(ctx context.Context, block *NewBlock) (bool, error)
}

// FinalizerPollConnector polls for new blocks. It takes a finalizer which will be used to determine when a block is finalized.
type FinalizerPollConnector struct {
	Connector
	logger    *zap.Logger
	Delay     time.Duration
	finalizer PollFinalizer
	blockFeed ethEvent.Feed
	errFeed   ethEvent.Feed
}

func NewFinalizerPollConnector(ctx context.Context, logger *zap.Logger, baseConnector Connector, finalizer PollFinalizer, delay time.Duration) (*FinalizerPollConnector, error) {
	if finalizer == nil {
		panic("finalizer must not be nil")
	}
	connector := &FinalizerPollConnector{
		Connector: baseConnector,
		logger:    logger,
		Delay:     delay,
		finalizer: finalizer,
	}
	err := supervisor.Run(ctx, "blockPoller", common.WrapWithScissors(connector.runFromSupervisor, "blockPoller"))
	if err != nil {
		return nil, err
	}
	return connector, nil
}

func (b *FinalizerPollConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
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

func (b *FinalizerPollConnector) runFromSupervisor(ctx context.Context) error {
	supervisor.Signal(ctx, supervisor.SignalHealthy)
	return b.run(ctx)
}

func (b *FinalizerPollConnector) run(ctx context.Context) error {
	prevLatest, err := GetLatestBlock(ctx, b.logger, b.Connector)
	if err != nil {
		return err
	}

	// Initialize the previous finalized block to latest. This is used to determine where to start looking for finalized blocks. We don't actually publish it.
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
			prevLatest, prevFinalized, err = b.pollBlock(ctx, prevLatest, prevFinalized)
			if err != nil {
				errCount++
				b.logger.Error("polling encountered an error", zap.Int("errCount", errCount), zap.Error(err))
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

// pollBlock poll for the latest block, compares them to the last one, and publishes any new ones.
// In the case of an error, it returns the last block that were passed in, otherwise it returns the new block.
func (b *FinalizerPollConnector) pollBlock(ctx context.Context, prevLatest *NewBlock, prevFinalized *NewBlock) (newLatest *NewBlock, newFinalized *NewBlock, err error) {
	newLatest, err = GetLatestBlock(ctx, b.logger, b.Connector)
	if err != nil {
		err = fmt.Errorf("failed to get latest block: %w", err)
		newLatest = prevLatest
		newFinalized = prevFinalized
		return
	}

	// First see if there new latest ones to publish.
	var block *NewBlock
	if newLatest.Number.Cmp(prevLatest.Number) > 0 {
		// If there is a gap between prev and new, we have to look up the hashes for the missing ones. Do that in batches.
		newBlockNum := newLatest.Number.Uint64()
		for blockNum := prevLatest.Number.Uint64() + 1; blockNum < newBlockNum; blockNum++ {
			block, err = GetBlockByNumberUint64(ctx, b.logger, b.Connector, blockNum, Latest)
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
		b.logger.Debug("latest block number went backwards, ignoring it", zap.Any("newLatest", newLatest), zap.Any("prevLatest", prevLatest))
		newLatest = prevLatest
	}

	// Now see if there might be some newly finalized ones to publish.
	newFinalized = prevFinalized
	if newLatest.Number.Cmp(prevFinalized.Number) > 0 {
		var finalized bool
		// If there is a gap between prev and new, we have to look up the hashes for the missing ones. Do that in batches.
		newBlockNum := newLatest.Number.Uint64()
		for blockNum := prevFinalized.Number.Uint64() + 1; blockNum <= newBlockNum; blockNum++ {
			block, err = GetBlockByNumberUint64(ctx, b.logger, b.Connector, blockNum, Finalized)
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

func GetLatestBlock(ctx context.Context, logger *zap.Logger, conn Connector) (*NewBlock, error) {
	return GetBlockByFinality(ctx, logger, conn, Latest)
}

func GetBlockByFinality(ctx context.Context, logger *zap.Logger, conn Connector, blockFinality FinalityLevel) (*NewBlock, error) {
	return GetBlock(ctx, logger, conn, blockFinality.String(), blockFinality)
}

func GetBlockByNumberUint64(ctx context.Context, logger *zap.Logger, conn Connector, blockNum uint64, blockFinality FinalityLevel) (*NewBlock, error) {
	return GetBlock(ctx, logger, conn, "0x"+fmt.Sprintf("%x", blockNum), blockFinality)
}

func GetBlock(ctx context.Context, logger *zap.Logger, conn Connector, str string, blockFinality FinalityLevel) (*NewBlock, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var m BlockMarshaller
	err := conn.RawCallContext(timeout, &m, "eth_getBlockByNumber", str, false)
	if err != nil {
		logger.Error("failed to get block",
			zap.String("requested_block", str), zap.Error(err))
		return nil, err
	}
	if m.Number == nil {
		logger.Error("failed to unmarshal block",
			zap.String("requested_block", str),
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
		Time:          uint64(m.Time),
		Hash:          m.Hash,
		L1BlockNumber: l1bn,
		Finality:      blockFinality,
	}, nil
}

func (b *FinalizerPollConnector) isBlockFinalized(ctx context.Context, block *NewBlock) (bool, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return b.finalizer.IsBlockFinalized(timeout, block)
}
