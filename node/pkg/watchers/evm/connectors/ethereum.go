package connectors

import (
	"context"
	"time"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	ethRpc "github.com/ethereum/go-ethereum/rpc"

	ethereum "github.com/ethereum/go-ethereum"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethEvent "github.com/ethereum/go-ethereum/event"

	"github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"
)

// EthereumConnector implements EVM network query capabilities for go-ethereum based networks and networks supporting
// the standard web3 rpc.
type EthereumConnector struct {
	networkName string
	address     ethCommon.Address
	logger      *zap.Logger
	client      *ethClient.Client
	rawClient   *ethRpc.Client
	filterer    *ethAbi.AbiFilterer
	caller      *ethAbi.AbiCaller
}

func NewEthereumConnector(ctx context.Context, networkName, rawUrl string, address ethCommon.Address, logger *zap.Logger) (*EthereumConnector, error) {
	rawClient, err := ethRpc.DialContext(ctx, rawUrl)
	if err != nil {
		return nil, err
	}

	client := ethClient.NewClient(rawClient)

	filterer, err := ethAbi.NewAbiFilterer(ethCommon.BytesToAddress(address.Bytes()), client)
	if err != nil {
		panic(err)
	}
	caller, err := ethAbi.NewAbiCaller(ethCommon.BytesToAddress(address.Bytes()), client)
	if err != nil {
		panic(err)
	}

	return &EthereumConnector{
		networkName: networkName,
		address:     address,
		logger:      logger.With(zap.String("eth_network", networkName)),
		client:      client,
		filterer:    filterer,
		caller:      caller,
		rawClient:   rawClient,
	}, nil
}

func (e *EthereumConnector) NetworkName() string {
	return e.networkName
}

func (e *EthereumConnector) ContractAddress() ethCommon.Address {
	return e.address
}

func (e *EthereumConnector) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	return e.caller.GetCurrentGuardianSetIndex(&ethBind.CallOpts{Context: ctx})
}

func (e *EthereumConnector) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	return e.caller.GetGuardianSet(&ethBind.CallOpts{Context: ctx}, index)
}

func (e *EthereumConnector) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return e.filterer.WatchLogMessagePublished(&ethBind.WatchOpts{Context: timeout}, sink, nil)
}

func (e *EthereumConnector) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	return e.client.TransactionReceipt(ctx, txHash)
}

func (e *EthereumConnector) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	block, err := e.client.HeaderByHash(ctx, hash)
	if err != nil {
		return 0, err
	}

	return block.Time, err
}

func (e *EthereumConnector) ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	return e.filterer.ParseLogMessagePublished(log)
}

func (e *EthereumConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	headSink := make(chan *ethTypes.Header, 2)
	headerSubscription, err := e.client.SubscribeNewHead(ctx, headSink)
	if err != nil {
		return nil, err
	}

	// The purpose of this is to map events from the geth event channel to the new block event channel.
	common.RunWithScissors(ctx, errC, "eth_connector_subscribe_for_block", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case ev := <-headSink:
				if ev == nil {
					e.logger.Error("new header event is nil")
					continue
				}
				if ev.Number == nil {
					e.logger.Error("new header block number is nil")
					continue
				}
				sink <- &NewBlock{
					Number:   ev.Number,
					Hash:     ev.Hash(),
					Finality: Finalized,
				}
				sink <- &NewBlock{
					Number:   ev.Number,
					Hash:     ev.Hash(),
					Finality: Safe,
				}
				sink <- &NewBlock{
					Number:   ev.Number,
					Hash:     ev.Hash(),
					Finality: Latest,
				}
			}
		}
	})

	return headerSubscription, err
}

func (e *EthereumConnector) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return e.rawClient.CallContext(ctx, result, method, args...)
}

func (e *EthereumConnector) RawBatchCallContext(ctx context.Context, b []ethRpc.BatchElem) error {
	return e.rawClient.BatchCallContext(ctx, b)
}

func (e *EthereumConnector) Client() *ethClient.Client {
	return e.client
}
