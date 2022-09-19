// This implements polling for log events.

// It works by using the finalizer in the polling implementation to check for log events on each new block.

package evm

import (
	"context"
	"errors"
	"fmt"
	ethereum2 "github.com/certusone/wormhole/node/pkg/evm/connectors"
	ethAbi "github.com/certusone/wormhole/node/pkg/evm/connectors/ethabi"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethEvent "github.com/ethereum/go-ethereum/event"

	common "github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"
)

type GetLogsImpl struct {
	*PollImpl
	Query  *GetLogsQuery
	logger *zap.Logger
}

func NewGetLogsImpl(networkName string, contract ethCommon.Address, delayInMs int) *GetLogsImpl {
	query := &GetLogsQuery{ContractAddress: contract}
	return &GetLogsImpl{PollImpl: &PollImpl{EthereumConnector: ethereum2.EthereumConnector{NetworkName: networkName}, Finalizer: query, DelayInMs: delayInMs}, Query: query}
}

func (e *GetLogsImpl) SetLogger(l *zap.Logger) {
	e.logger = l
	e.logger.Info("using eth_getLogs api to retreive log events", zap.String("eth_network", e.PollImpl.EthereumConnector.NetworkName))
	e.PollImpl.SetLogger(l)
}

func (e *GetLogsImpl) DialContext(ctx context.Context, rawurl string) (err error) {
	e.Query.poller = e.PollImpl
	return e.PollImpl.DialContext(ctx, rawurl)
}

type GetLogsPollSubscription struct {
	errOnce   sync.Once
	err       chan error
	quit      chan error
	unsubDone chan struct{}
}

var ErrUnsubscribedForGetLogs = errors.New("unsubscribed")

func (sub *GetLogsPollSubscription) Err() <-chan error {
	return sub.err
}

func (sub *GetLogsPollSubscription) Unsubscribe() {
	sub.errOnce.Do(func() {
		select {
		case sub.quit <- ErrUnsubscribedForGetLogs:
			<-sub.unsubDone
		case <-sub.unsubDone:
		}
		close(sub.err)
	})
}

func (e *GetLogsImpl) WatchLogMessagePublished(ctx, timeout context.Context, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	e.Query.sink = sink

	e.Query.sub = &GetLogsPollSubscription{
		err: make(chan error, 1),
	}

	return e.Query.sub, nil
}

type GetLogsQuery struct {
	logger          *zap.Logger
	networkName     string
	ContractAddress ethCommon.Address
	prevBlockNum    *big.Int
	client          *ethClient.Client
	poller          *PollImpl
	sink            chan<- *ethAbi.AbiLogMessagePublished
	sub             *GetLogsPollSubscription
}

func (f *GetLogsQuery) SetLogger(l *zap.Logger, netName string) {
	f.logger = l
	f.networkName = netName
}

func (f *GetLogsQuery) DialContext(ctx context.Context, rawurl string) (err error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	f.client, err = ethClient.DialContext(timeout, rawurl)
	return err
}

var (
	getLogsBigOne       = big.NewInt(1)
	logsLogMessageTopic = ethCommon.HexToHash("0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2")
)

// This doesn't actually check finality, instead it queries for new log events.
func (f *GetLogsQuery) IsBlockFinalized(ctx context.Context, block *common.NewBlock) (bool, error) {
	if f.prevBlockNum == nil {
		f.prevBlockNum = new(big.Int).Set(block.Number)
	} else {
		f.prevBlockNum.Add(f.prevBlockNum, getLogsBigOne)
	}

	filter := ethereum.FilterQuery{
		FromBlock: f.prevBlockNum,
		ToBlock:   block.Number,
		Addresses: []ethCommon.Address{f.ContractAddress},
	}

	*f.prevBlockNum = *block.Number

	logs, err := f.client.FilterLogs(ctx, filter)
	if err != nil {
		f.logger.Error("GetLogsQuery: query of eth_getLogs failed",
			zap.String("eth_network", f.networkName),
			zap.Stringer("FromBlock", filter.FromBlock),
			zap.Stringer("ToBlock", filter.ToBlock),
			zap.Error(err),
		)

		f.sub.err <- fmt.Errorf("GetLogsQuery: failed to query for log messages: %w", err)
		return true, nil // We still return true here, because we don't want this error flagged against the poller.
	}

	if len(logs) == 0 {
		return true, nil
	}

	for _, log := range logs {
		if log.Topics[0] == logsLogMessageTopic {
			ev, err := f.poller.ParseLogMessagePublished(log)
			if err != nil {
				f.logger.Error("GetLogsQuery: failed to parse log entry",
					zap.String("eth_network", f.networkName),
					zap.Stringer("FromBlock", filter.FromBlock),
					zap.Stringer("ToBlock", filter.ToBlock),
					zap.Error(err),
				)

				f.sub.err <- fmt.Errorf("failed to parse log message: %w", err)
				continue
			}

			f.sink <- ev
		}
	}

	return true, nil
}
