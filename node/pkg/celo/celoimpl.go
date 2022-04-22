// This implements the interface to the "go-ethereum-ish" library used by Celo.

package celo

import (
	"context"

	celoBind "github.com/celo-org/celo-blockchain/accounts/abi/bind"
	celoCommon "github.com/celo-org/celo-blockchain/common"
	celoTypes "github.com/celo-org/celo-blockchain/core/types"
	celoClient "github.com/celo-org/celo-blockchain/ethclient"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethEvent "github.com/ethereum/go-ethereum/event"

	celoAbi "github.com/certusone/wormhole/node/pkg/celo/abi"
	ethAbi "github.com/certusone/wormhole/node/pkg/ethereum/abi"

	"go.uber.org/zap"
)

type CeloImpl struct {
	NetworkName string
	logger      *zap.Logger
	client      *celoClient.Client
	filterer    *celoAbi.AbiFilterer
	caller      *celoAbi.AbiCaller
}

func (e *CeloImpl) SetLogger(l *zap.Logger) {
	e.logger = l
	e.logger.Info("using celo specific ethereum library", zap.String("eth_network", e.NetworkName))
}

func (e *CeloImpl) DialContext(ctx context.Context, rawurl string) (err error) {
	e.client, err = celoClient.DialContext(ctx, rawurl)
	return
}

func (e *CeloImpl) NewAbiFilterer(address ethCommon.Address) (err error) {
	e.filterer, err = celoAbi.NewAbiFilterer(celoCommon.BytesToAddress(address.Bytes()), e.client)
	return
}

func (e *CeloImpl) NewAbiCaller(address ethCommon.Address) (err error) {
	e.caller, err = celoAbi.NewAbiCaller(celoCommon.BytesToAddress(address.Bytes()), e.client)
	return
}

func (e *CeloImpl) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	if e.caller == nil {
		panic("caller is not initialized!")
	}

	opts := &celoBind.CallOpts{Context: ctx}
	return e.caller.GetCurrentGuardianSetIndex(opts)
}

func (e *CeloImpl) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	if e.caller == nil {
		panic("caller is not initialized!")
	}

	opts := &celoBind.CallOpts{Context: ctx}
	celoGs, err := e.caller.GetGuardianSet(opts, index)
	if err != nil {
		return ethAbi.StructsGuardianSet{}, err
	}

	ethKeys := make([]ethCommon.Address, len(celoGs.Keys))
	for n, k := range celoGs.Keys {
		ethKeys[n] = ethCommon.BytesToAddress(k.Bytes())
	}

	return ethAbi.StructsGuardianSet{
		Keys:           ethKeys,
		ExpirationTime: celoGs.ExpirationTime,
	}, err
}

func (e *CeloImpl) WatchLogMessagePublished(ctx, timeout context.Context, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	if e.filterer == nil {
		panic("filterer is not initialized!")
	}

	messageC := make(chan *celoAbi.AbiLogMessagePublished, 2)
	messageSub, err := e.filterer.WatchLogMessagePublished(&celoBind.WatchOpts{Context: timeout}, messageC, nil)
	if err != nil {
		return messageSub, err
	}

	// The purpose of this is to map events from the Celo log message channel to the Eth log message channel.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case celoEvent := <-messageC:
				sink <- convertEventToEth(celoEvent)
			}
		}
	}()

	return messageSub, err
}

func (e *CeloImpl) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	if e.client == nil {
		panic("client is not initialized!")
	}

	celoReceipt, err := e.client.TransactionReceipt(ctx, celoCommon.BytesToHash(txHash.Bytes()))
	if err != nil {
		return nil, err
	}

	return convertReceiptToEth(celoReceipt), err
}

func (e *CeloImpl) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	if e.client == nil {
		panic("client is not initialized!")
	}

	block, err := e.client.BlockByHash(ctx, celoCommon.BytesToHash(hash.Bytes()))
	if err != nil {
		return 0, err
	}

	return block.Time(), err
}

func (e *CeloImpl) ParseLogMessagePublished(ethLog ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	if e.filterer == nil {
		panic("filterer is not initialized!")
	}

	celoEvent, err := e.filterer.ParseLogMessagePublished(*convertLogFromEth(&ethLog))
	if err != nil {
		return nil, err
	}

	return convertEventToEth(celoEvent), err
}

func (e *CeloImpl) SubscribeNewHead(ctx context.Context, sink chan<- *ethTypes.Header) (ethereum.Subscription, error) {
	if e.client == nil {
		panic("client is not initialized!")
	}

	headSink := make(chan *celoTypes.Header, 2)
	headerSubscription, err := e.client.SubscribeNewHead(ctx, headSink)
	if err != nil {
		return headerSubscription, err
	}

	// The purpose of this is to map events from the Celo event channel to the Eth event channel.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case celoEvent := <-headSink:
				sink <- convertNewHeadEventToEth(celoEvent)
			}
		}
	}()

	return headerSubscription, err
}

func convertEventToEth(ev *celoAbi.AbiLogMessagePublished) *ethAbi.AbiLogMessagePublished {
	return &ethAbi.AbiLogMessagePublished{
		Sender:           ethCommon.BytesToAddress(ev.Sender.Bytes()),
		Sequence:         ev.Sequence,
		Nonce:            ev.Nonce,
		Payload:          ev.Payload,
		ConsistencyLevel: ev.ConsistencyLevel,
		Raw:              *convertLogToEth(&ev.Raw),
	}
}

func convertLogToEth(l *celoTypes.Log) *ethTypes.Log {
	topics := make([]ethCommon.Hash, len(l.Topics))
	for n, t := range l.Topics {
		topics[n] = ethCommon.BytesToHash(t.Bytes())
	}

	return &ethTypes.Log{
		Address:     ethCommon.BytesToAddress(l.Address.Bytes()),
		Topics:      topics,
		Data:        l.Data,
		BlockNumber: l.BlockNumber,
		TxHash:      ethCommon.BytesToHash(l.TxHash.Bytes()),
		TxIndex:     l.TxIndex,
		BlockHash:   ethCommon.BytesToHash(l.BlockHash.Bytes()),
		Index:       l.Index,
		Removed:     l.Removed,
	}
}

func convertReceiptToEth(celoReceipt *celoTypes.Receipt) *ethTypes.Receipt {
	ethLogs := make([]*ethTypes.Log, len(celoReceipt.Logs))
	for n, l := range celoReceipt.Logs {
		ethLogs[n] = convertLogToEth(l)
	}

	return &ethTypes.Receipt{
		Type:              celoReceipt.Type,
		PostState:         celoReceipt.PostState,
		Status:            celoReceipt.Status,
		CumulativeGasUsed: celoReceipt.CumulativeGasUsed,
		Bloom:             ethTypes.BytesToBloom(celoReceipt.Bloom.Bytes()),
		Logs:              ethLogs,
		TxHash:            ethCommon.BytesToHash(celoReceipt.TxHash.Bytes()),
		ContractAddress:   ethCommon.BytesToAddress(celoReceipt.ContractAddress.Bytes()),
		GasUsed:           celoReceipt.GasUsed,
		BlockHash:         ethCommon.BytesToHash(celoReceipt.BlockHash.Bytes()),
		BlockNumber:       celoReceipt.BlockNumber,
		TransactionIndex:  celoReceipt.TransactionIndex,
	}
}

func convertLogFromEth(l *ethTypes.Log) *celoTypes.Log {
	topics := make([]celoCommon.Hash, len(l.Topics))
	for n, t := range l.Topics {
		topics[n] = celoCommon.BytesToHash(t.Bytes())
	}

	return &celoTypes.Log{
		Address:     celoCommon.BytesToAddress(l.Address.Bytes()),
		Topics:      topics,
		Data:        l.Data,
		BlockNumber: l.BlockNumber,
		TxHash:      celoCommon.BytesToHash(l.TxHash.Bytes()),
		TxIndex:     l.TxIndex,
		BlockHash:   celoCommon.BytesToHash(l.BlockHash.Bytes()),
		Index:       l.Index,
		Removed:     l.Removed,
	}
}

func convertNewHeadEventToEth(ev *celoTypes.Header) *ethTypes.Header {
	return &ethTypes.Header{
		ParentHash: ethCommon.BytesToHash(ev.ParentHash.Bytes()),
		// Celo doesn't have UncleHash.
		Coinbase:    ethCommon.BytesToAddress(ev.Coinbase.Bytes()),
		Root:        ethCommon.BytesToHash(ev.Root.Bytes()),
		TxHash:      ethCommon.BytesToHash(ev.TxHash.Bytes()),
		ReceiptHash: ethCommon.BytesToHash(ev.ReceiptHash.Bytes()),
		Bloom:       ethTypes.BytesToBloom(ev.Bloom.Bytes()),
		// Celo doesn't have Difficulty.
		Number: ev.Number,
		// Celo doesn't have GasLimit.
		GasUsed: ev.GasUsed,
		Time:    ev.Time,
		Extra:   ev.Extra,
		// Celo doesn't have MixDigest.
		// Celo doesn't have Nonce.
		// Celo doesn't have BaseFee.
	}
}
