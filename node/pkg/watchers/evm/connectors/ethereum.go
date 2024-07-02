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

	"go.uber.org/zap"
)

// EthereumBaseConnector implements EVM network query capabilities for go-ethereum based networks and networks supporting
// the standard web3 rpc.
type EthereumBaseConnector struct {
	networkName string
	address     ethCommon.Address
	logger      *zap.Logger
	client      *ethClient.Client
	rawClient   *ethRpc.Client
	filterer    *ethAbi.AbiFilterer
	caller      *ethAbi.AbiCaller
}

func NewEthereumBaseConnector(ctx context.Context, networkName, rawUrl string, address ethCommon.Address, logger *zap.Logger) (*EthereumBaseConnector, error) {
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

	return &EthereumBaseConnector{
		networkName: networkName,
		address:     address,
		logger:      logger,
		client:      client,
		filterer:    filterer,
		caller:      caller,
		rawClient:   rawClient,
	}, nil
}

func (e *EthereumBaseConnector) NetworkName() string {
	return e.networkName
}

func (e *EthereumBaseConnector) ContractAddress() ethCommon.Address {
	return e.address
}

func (e *EthereumBaseConnector) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	return e.caller.GetCurrentGuardianSetIndex(&ethBind.CallOpts{Context: ctx})
}

func (e *EthereumBaseConnector) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	return e.caller.GetGuardianSet(&ethBind.CallOpts{Context: ctx}, index)
}

func (e *EthereumBaseConnector) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return e.filterer.WatchLogMessagePublished(&ethBind.WatchOpts{Context: timeout}, sink, nil)
}

func (e *EthereumBaseConnector) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	return e.client.TransactionReceipt(ctx, txHash)
}

func (e *EthereumBaseConnector) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	block, err := e.client.HeaderByHash(ctx, hash)
	if err != nil {
		return 0, err
	}

	return block.Time, err
}

func (e *EthereumBaseConnector) ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	return e.filterer.ParseLogMessagePublished(log)
}

func (e *EthereumBaseConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	panic("not implemented")
}

func (e *EthereumBaseConnector) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return e.rawClient.CallContext(ctx, result, method, args...)
}

func (e *EthereumBaseConnector) RawBatchCallContext(ctx context.Context, b []ethRpc.BatchElem) error {
	return e.rawClient.BatchCallContext(ctx, b)
}

func (e *EthereumBaseConnector) Client() *ethClient.Client {
	return e.client
}

func (e *EthereumBaseConnector) SubscribeNewHead(ctx context.Context, ch chan<- *ethTypes.Header) (ethereum.Subscription, error) {
	return e.client.SubscribeNewHead(ctx, ch)
}
