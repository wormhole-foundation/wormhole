// This implements the interface to the standard go-ethereum library.

package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethHexUtils "github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethEvent "github.com/ethereum/go-ethereum/event"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	common "github.com/certusone/wormhole/node/pkg/common"
	ethAbi "github.com/certusone/wormhole/node/pkg/ethereum/abi"

	"go.uber.org/zap"
)

type MoonbeamImpl struct {
	BaseEth  *EthImpl
	logger   *zap.Logger
	rawClient *ethRpc.Client
}

func (e *MoonbeamImpl) SetLogger(l *zap.Logger) {
	e.logger = l
	e.logger.Info("using Moonbeam specific implementation", zap.String("eth_network", e.BaseEth.NetworkName))
}

func (e *MoonbeamImpl) DialContext(ctx context.Context, rawurl string) (err error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// This is used for doing raw eth_ RPC calls.
	e.rawClient, err = ethRpc.DialContext(timeout, rawurl)
	if err != nil {
		return err
	}

	// This is used for doing all other go-ethereum calls.
	return e.BaseEth.DialContext(ctx, rawurl)
}

func (e *MoonbeamImpl) NewAbiFilterer(address ethCommon.Address) (err error) {
	return e.BaseEth.NewAbiFilterer(address)
}

func (e *MoonbeamImpl) NewAbiCaller(address ethCommon.Address) (err error) {
	return e.BaseEth.NewAbiCaller(address)
}

func (e *MoonbeamImpl) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	return e.BaseEth.GetCurrentGuardianSetIndex(ctx)
}

func (e *MoonbeamImpl) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	return e.BaseEth.GetGuardianSet(ctx, index)
}

func (e *MoonbeamImpl) WatchLogMessagePublished(ctx, timeout context.Context, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	return e.BaseEth.WatchLogMessagePublished(ctx, timeout, sink)
}

func (e *MoonbeamImpl) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	return e.BaseEth.TransactionReceipt(ctx, txHash)
}

func (e *MoonbeamImpl) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	return e.BaseEth.TimeOfBlockByHash(ctx, hash)
}

func (e *MoonbeamImpl) ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	return e.BaseEth.ParseLogMessagePublished(log)
}

type MoonbeamSubscription struct {
	errOnce   sync.Once
	err       chan error
	quit      chan error
	unsubDone chan struct{}
}

func (sub *MoonbeamSubscription) Err() <-chan error {
	return sub.err
}

var errUnsubscribed = errors.New("unsubscribed")

func (sub *MoonbeamSubscription) Unsubscribe() {
	sub.errOnce.Do(func() {
		select {
		case sub.quit <- errUnsubscribed:
			<-sub.unsubDone
		case <-sub.unsubDone:
		}
		close(sub.err)
	})
}

// Moonbeam can publish blocks before they are marked final. This means we need to sit on the block until a special "is finalized"
// query returns true. The assumption is that every block number will eventually be published and finalized, it's just that the contents
// of the block (and therefore the hash) might change if there is a rollback. Therefore rather than subscribing for headers from geth,
// we use a polling mechanism to get the next expected block, and keep doing it until it is marked final.

func (e *MoonbeamImpl) SubscribeForBlocks(ctx context.Context, sink chan<- *common.NewBlock) (ethereum.Subscription, error) {
	if e.BaseEth.client == nil {
		panic("client is not initialized!")
	}
	if e.rawClient == nil {
		panic("rawClient is not initialized!")
	}

	sub := &MoonbeamSubscription{
		err: make(chan error, 1),
	}

	latestBlock, err := e.getBlock(ctx, nil)
	if err != nil {
		return nil, err
	}
	currentBlockNumber := *latestBlock.Number

	const DELAY_IN_MS = 250
	var BIG_ONE = big.NewInt(1)

	timer := time.NewTimer(time.Millisecond) // Start immediately.
	go func() {
		var errorCount int
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				var errorOccurred bool
				for {
					var block *common.NewBlock
					var err error
					errorOccurred = false

					// See if the next block has been created yet.
					if currentBlockNumber.Cmp(latestBlock.Number) > 0 {
						latestBlock, err = e.getBlock(ctx, nil)
						if err != nil {
							errorOccurred = true
							e.logger.Error("failed to look up latest block", zap.String("eth_network", e.BaseEth.NetworkName),
								zap.Uint64("block", currentBlockNumber.Uint64()), zap.Error(err))
							break
						}

						if currentBlockNumber.Cmp(latestBlock.Number) > 0 {
							// We have to wait for this block to become available.
							break
						}

						if currentBlockNumber.Cmp(latestBlock.Number) == 0 {
							block = latestBlock
						}
					}

					// Fetch the hash every time, in case it changes due to a rollback. The only exception is if we just got it above.
					if block == nil {
						block, err = e.getBlock(ctx, &currentBlockNumber)
						if err != nil {
							errorOccurred = true
							e.logger.Error("failed to get current block", zap.String("eth_network", e.BaseEth.NetworkName),
								zap.Uint64("block", currentBlockNumber.Uint64()), zap.Error(err))
							break
						}
					}

					finalized, err := e.isBlockFinalized(ctx, block.Hash.Hex())
					if err != nil {
						errorOccurred = true
						e.logger.Error("failed to see if block is finalized", zap.String("eth_network", e.BaseEth.NetworkName),
							zap.Uint64("block", currentBlockNumber.Uint64()), zap.Error(err))
						break
					}
					if !finalized {
						break
					}

					sink <- block
					currentBlockNumber.Add(&currentBlockNumber, BIG_ONE)
				}

				if errorOccurred {
					errorCount++
					if errorCount > 1 {
						sub.err <- fmt.Errorf("polling encountered multiple errors")
					}
				} else {
					errorCount = 0
				}

				timer = time.NewTimer(DELAY_IN_MS * time.Millisecond)
			}
		}
	}()

	return sub, err
}

func (e *MoonbeamImpl) getBlock(ctx context.Context, number *big.Int) (*common.NewBlock, error) {
	var numStr string
	if number != nil {
		numStr = ethHexUtils.EncodeBig(number)
	} else {
		numStr = "latest"
	}

	type Marshaller struct {
		Number *ethHexUtils.Big
		Hash ethCommon.Hash `json:"hash"`
	}

	var m Marshaller
	err := e.rawClient.CallContext(ctx, &m, "eth_getBlockByNumber", numStr, false)
	if err != nil {
		e.logger.Error("failed to get block", zap.String("eth_network", e.BaseEth.NetworkName),
			zap.String("requested_block", numStr), zap.Error(err))
		return nil, err
	}

	n := big.Int(*m.Number)
	return &common.NewBlock{
		Number: &n,
		Hash:   m.Hash,
	}, nil
}

func (e *MoonbeamImpl) isBlockFinalized(ctx context.Context, hash string) (bool, error) {
	var finalized bool
	err := e.rawClient.CallContext(ctx, &finalized, "moon_isBlockFinalized", hash)
	if err != nil {
		e.logger.Error("failed to check for finality", zap.String("eth_network", e.BaseEth.NetworkName), zap.Error(err))
		return false, err
	}

	return finalized, nil
}
