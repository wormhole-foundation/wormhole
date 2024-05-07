package connectors

import (
	"context"
	"time"

	celoAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/celoabi"
	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	celoBind "github.com/celo-org/celo-blockchain/accounts/abi/bind"
	celoCommon "github.com/celo-org/celo-blockchain/common"
	celoTypes "github.com/celo-org/celo-blockchain/core/types"
	celoClient "github.com/celo-org/celo-blockchain/ethclient"
	celoRpc "github.com/celo-org/celo-blockchain/rpc"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethEvent "github.com/ethereum/go-ethereum/event"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"
)

// CeloConnector implements EVM network query capabilities for the Celo network. It's almost identical to
// EthereumConnector except it's using the Celo fork and provides shims between their respective types.
type CeloConnector struct {
	networkName string
	address     ethCommon.Address
	logger      *zap.Logger
	client      *celoClient.Client
	rawClient   *celoRpc.Client
	filterer    *celoAbi.AbiFilterer
	caller      *celoAbi.AbiCaller
}

func NewCeloConnector(ctx context.Context, networkName, rawUrl string, address ethCommon.Address, logger *zap.Logger) (*CeloConnector, error) {
	rawClient, err := celoRpc.DialContext(ctx, rawUrl)
	if err != nil {
		return nil, err
	}
	client := celoClient.NewClient(rawClient)

	filterer, err := celoAbi.NewAbiFilterer(celoCommon.BytesToAddress(address.Bytes()), client)
	if err != nil {
		panic(err)
	}
	caller, err := celoAbi.NewAbiCaller(celoCommon.BytesToAddress(address.Bytes()), client)
	if err != nil {
		panic(err)
	}

	return &CeloConnector{
		networkName: networkName,
		address:     address,
		logger:      logger,
		client:      client,
		rawClient:   rawClient,
		filterer:    filterer,
		caller:      caller,
	}, nil
}

func (c *CeloConnector) NetworkName() string {
	return c.networkName
}

func (c *CeloConnector) ContractAddress() ethCommon.Address {
	return c.address
}

func (c *CeloConnector) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	opts := &celoBind.CallOpts{Context: ctx}
	return c.caller.GetCurrentGuardianSetIndex(opts)
}

func (c *CeloConnector) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	opts := &celoBind.CallOpts{Context: ctx}
	celoGs, err := c.caller.GetGuardianSet(opts, index)
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

func (c *CeloConnector) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	messageC := make(chan *celoAbi.AbiLogMessagePublished, 2)
	messageSub, err := c.filterer.WatchLogMessagePublished(&celoBind.WatchOpts{Context: timeout}, messageC, nil)
	if err != nil {
		return messageSub, err
	}

	// The purpose of this is to map events from the Celo log message channel to the Eth log message channel.
	common.RunWithScissors(ctx, errC, "celo_connector_watch_log", func(ctx context.Context) error {
		for {
			select {
			// This will return when the subscription is unsubscribed as the error channel gets closed
			case <-messageSub.Err():
				return nil
			case celoEvent := <-messageC:
				sink <- convertCeloEventToEth(celoEvent)
			}
		}
	})

	return messageSub, err
}

func (c *CeloConnector) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	celoReceipt, err := c.client.TransactionReceipt(ctx, celoCommon.BytesToHash(txHash.Bytes()))
	if err != nil {
		return nil, err
	}

	return convertCeloReceiptToEth(celoReceipt), err
}

func (c *CeloConnector) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	block, err := c.client.HeaderByHash(ctx, celoCommon.BytesToHash(hash.Bytes()))
	if err != nil {
		return 0, err
	}

	return block.Time, err
}

func (c *CeloConnector) ParseLogMessagePublished(ethLog ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	celoEvent, err := c.filterer.ParseLogMessagePublished(*convertCeloLogFromEth(&ethLog))
	if err != nil {
		return nil, err
	}

	return convertCeloEventToEth(celoEvent), err
}

func (c *CeloConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	headSink := make(chan *celoTypes.Header, 2)
	headerSubscription, err := c.client.SubscribeNewHead(ctx, headSink)
	if err != nil {
		return headerSubscription, err
	}

	// The purpose of this is to map events from the Celo event channel to the new block event channel.
	common.RunWithScissors(ctx, errC, "celo_connector_subscribe_for_block", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case ev := <-headSink:
				if ev == nil {
					c.logger.Error("new header event is nil")
					continue
				}
				if ev.Number == nil {
					c.logger.Error("new header block number is nil")
					continue
				}
				hash := ethCommon.BytesToHash(ev.Hash().Bytes())
				sink <- &NewBlock{
					Number:   ev.Number,
					Hash:     hash,
					Time:     ev.Time,
					Finality: Finalized,
				}
				sink <- &NewBlock{
					Number:   ev.Number,
					Hash:     hash,
					Time:     ev.Time,
					Finality: Safe,
				}
				sink <- &NewBlock{
					Number:   ev.Number,
					Hash:     hash,
					Time:     ev.Time,
					Finality: Latest,
				}
			}
		}
	})

	return headerSubscription, err
}

func (c *CeloConnector) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return c.rawClient.CallContext(ctx, result, method, args...)
}

func (c *CeloConnector) RawBatchCallContext(ctx context.Context, b []ethRpc.BatchElem) error {
	celoB := make([]celoRpc.BatchElem, len(b))
	for i, v := range b {
		celoB[i] = celoRpc.BatchElem{
			Method: v.Method,
			Args:   v.Args,
			Result: v.Result,
			Error:  v.Error,
		}
	}
	return c.rawClient.BatchCallContext(ctx, celoB)
}

func (c *CeloConnector) Client() *ethClient.Client {
	panic("unimplemented")
}

func (c *CeloConnector) SubscribeNewHead(ctx context.Context, ch chan<- *ethTypes.Header) (ethereum.Subscription, error) {
	panic("unimplemented")
}

func convertCeloEventToEth(ev *celoAbi.AbiLogMessagePublished) *ethAbi.AbiLogMessagePublished {
	return &ethAbi.AbiLogMessagePublished{
		Sender:           ethCommon.BytesToAddress(ev.Sender.Bytes()),
		Sequence:         ev.Sequence,
		Nonce:            ev.Nonce,
		Payload:          ev.Payload,
		ConsistencyLevel: ev.ConsistencyLevel,
		Raw:              *convertCeloLogToEth(&ev.Raw),
	}
}

func convertCeloLogToEth(l *celoTypes.Log) *ethTypes.Log {
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

func convertCeloReceiptToEth(celoReceipt *celoTypes.Receipt) *ethTypes.Receipt {
	ethLogs := make([]*ethTypes.Log, len(celoReceipt.Logs))
	for n, l := range celoReceipt.Logs {
		ethLogs[n] = convertCeloLogToEth(l)
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

func convertCeloLogFromEth(l *ethTypes.Log) *celoTypes.Log {
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
