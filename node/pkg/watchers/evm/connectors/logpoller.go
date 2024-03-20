package connectors

import (
	"context"
	"fmt"
	"math/big"
	"time"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethEvent "github.com/ethereum/go-ethereum/event"

	"github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"
)

// LogPollConnector pulls logs on each new block event when subscribing using WatchLogMessagePublished instead of using
// a websocket connection. It can be used in conjunction with a BlockPollConnector and Finalizer to only return
// finalized message log events.
type LogPollConnector struct {
	Connector
	logger      *zap.Logger
	client      *ethClient.Client
	messageFeed ethEvent.Feed
	errFeed     ethEvent.Feed

	prevBlockNum *big.Int
}

func NewLogPollConnector(ctx context.Context, logger *zap.Logger, baseConnector Connector, client *ethClient.Client) (*LogPollConnector, error) {
	connector := &LogPollConnector{
		Connector: baseConnector,
		logger:    logger,
		client:    client,
	}
	// The supervisor will keep the poller running
	err := supervisor.Run(ctx, "logPoller", common.WrapWithScissors(connector.run, "logPoller"))
	if err != nil {
		return nil, err
	}
	return connector, nil
}

func (l *LogPollConnector) run(ctx context.Context) error {
	blockChan := make(chan *NewBlock)
	errC := make(chan error)

	sub, err := l.SubscribeForBlocks(ctx, errC, blockChan)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	supervisor.Signal(ctx, supervisor.SignalHealthy)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sub.Err():
			return err
		case err := <-errC:
			return err
		case block := <-blockChan:
			if err := l.processBlock(ctx, block); err != nil {
				l.errFeed.Send(err.Error())
			}
		}
	}
}

func (l *LogPollConnector) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	sub := NewPollSubscription()
	messageSub := l.messageFeed.Subscribe(sink)

	// The feed library does not support error forwarding, so we're emulating that using a custom subscription and
	// an error feed.
	innerErrSink := make(chan string, 10)
	innerErrSub := l.errFeed.Subscribe(innerErrSink)

	common.RunWithScissors(ctx, errC, "log_poll_watch_log", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				messageSub.Unsubscribe()
				innerErrSub.Unsubscribe()
				return nil
			case <-sub.quit:
				messageSub.Unsubscribe()
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

var (
	getLogsBigOne       = big.NewInt(1)
	logsLogMessageTopic = ethCommon.HexToHash("0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2")
)

func (l *LogPollConnector) processBlock(ctx context.Context, block *NewBlock) error {
	if l.prevBlockNum == nil {
		l.prevBlockNum = new(big.Int).Set(block.Number)
	} else {
		l.prevBlockNum.Add(l.prevBlockNum, getLogsBigOne)
	}

	filter := ethereum.FilterQuery{
		FromBlock: l.prevBlockNum,
		ToBlock:   block.Number,
		Addresses: []ethCommon.Address{l.ContractAddress()},
	}

	*l.prevBlockNum = *block.Number

	tCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	logs, err := l.client.FilterLogs(tCtx, filter)
	if err != nil {
		l.logger.Error("GetLogsQuery: query of eth_getLogs failed",
			zap.Stringer("FromBlock", filter.FromBlock),
			zap.Stringer("ToBlock", filter.ToBlock),
			zap.Error(err),
		)

		return fmt.Errorf("GetLogsQuery: failed to query for log messages: %w", err)
	}

	if len(logs) == 0 {
		return nil
	}

	for _, log := range logs {
		if log.Topics[0] != logsLogMessageTopic {
			continue
		}
		ev, err := l.ParseLogMessagePublished(log)
		if err != nil {
			l.logger.Error("GetLogsQuery: failed to parse log entry",
				zap.Stringer("FromBlock", filter.FromBlock),
				zap.Stringer("ToBlock", filter.ToBlock),
				zap.Error(err),
			)

			l.errFeed.Send(fmt.Errorf("failed to parse log message: %w", err))
			continue
		}

		l.messageFeed.Send(ev)
	}

	return nil
}
