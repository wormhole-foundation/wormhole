// This implements the interface to the standard go-ethereum library.

package ethereum

import (
	"context"

	ethereum "github.com/ethereum/go-ethereum"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethEvent "github.com/ethereum/go-ethereum/event"

	ethAbi "github.com/certusone/wormhole/node/pkg/ethereum/abi"

	"go.uber.org/zap"
)

type EthImpl struct {
	NetworkName string
	client      *ethClient.Client
	filterer    *ethAbi.AbiFilterer
	caller      *ethAbi.AbiCaller
}

func (e *EthImpl) SetLogger(_l *zap.Logger) {}

func (e *EthImpl) DialContext(ctx context.Context, rawurl string) (err error) {
	e.client, err = ethClient.DialContext(ctx, rawurl)
	return
}

func (e *EthImpl) NewAbiFilterer(address ethCommon.Address) (err error) {
	e.filterer, err = ethAbi.NewAbiFilterer(address, e.client)
	return
}

func (e *EthImpl) NewAbiCaller(address ethCommon.Address) (err error) {
	e.caller, err = ethAbi.NewAbiCaller(address, e.client)
	return
}

func (e *EthImpl) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	if e.caller == nil {
		panic("caller is not initialized!")
	}

	opts := &ethBind.CallOpts{Context: ctx}
	return e.caller.GetCurrentGuardianSetIndex(opts)
}

func (e *EthImpl) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	if e.caller == nil {
		panic("caller is not initialized!")
	}

	opts := &ethBind.CallOpts{Context: ctx}
	return e.caller.GetGuardianSet(opts, index)
}

func (e *EthImpl) WatchLogMessagePublished(_ctx, timeout context.Context, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	if e.filterer == nil {
		panic("filterer is not initialized!")
	}

	return e.filterer.WatchLogMessagePublished(&ethBind.WatchOpts{Context: timeout}, sink, nil)
}

func (e *EthImpl) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	if e.client == nil {
		panic("client is not initialized!")
	}

	return e.client.TransactionReceipt(ctx, txHash)
}

func (e *EthImpl) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	if e.client == nil {
		panic("client is not initialized!")
	}

	block, err := e.client.BlockByHash(ctx, hash)
	if err != nil {
		return 0, err
	}

	return block.Time(), err
}

func (e *EthImpl) ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	if e.filterer == nil {
		panic("filterer is not initialized!")
	}

	return e.filterer.ParseLogMessagePublished(log)
}

func (e *EthImpl) SubscribeNewHead(ctx context.Context, sink chan<- *ethTypes.Header) (ethereum.Subscription, error) {
	if e.client == nil {
		panic("client is not initialized!")
	}

	return e.client.SubscribeNewHead(ctx, sink)
}
