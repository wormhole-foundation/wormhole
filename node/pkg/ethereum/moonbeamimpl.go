// This implements the interface to the standard go-ethereum library.

package ethereum

import (
	"context"
	"errors"
	"math/big"
	"strconv"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethHexUtils "github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethEvent "github.com/ethereum/go-ethereum/event"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	common "github.com/certusone/wormhole/node/pkg/common"
	ethAbi "github.com/certusone/wormhole/node/pkg/ethereum/abi"

	"go.uber.org/zap"
)

type MoonbeamImpl struct {
	BaseEth  *EthImpl
	logger   *zap.Logger
	mbClient *ethClient.Client
	mbRpcCon *ethRpc.Client
}

func (e *MoonbeamImpl) SetLogger(l *zap.Logger) {
	e.logger = l
	e.logger.Info("using Moonbeam specific implementation", zap.String("eth_network", e.BaseEth.NetworkName))
}

func (e *MoonbeamImpl) DialContext(ctx context.Context, rawurl string) (err error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// This is used for doing raw eth_ RPC calls.
	e.mbRpcCon, err = ethRpc.DialContext(timeout, rawurl)
	if err != nil {
		return err
	}

	// This is used for doing go-ethereum calls in the polling method.
	e.mbClient, err = ethClient.DialContext(ctx, rawurl)
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

func (e *MoonbeamImpl) SubscribeForBlocks(ctx context.Context, sink chan<- *common.NewBlock) (ethereum.Subscription, error) {
	if e.BaseEth.client == nil {
		panic("client is not initialized!")
	}
	if e.mbRpcCon == nil {
		panic("mbRpcCon is not initialized!")
	}
	if e.mbClient == nil {
		panic("mbClient is not initialized!")
	}	

	latestBlockNumber, err := e.getLatestBlockNumber(ctx)
	if err != nil {
		return nil, err
	}
	currentBlockNumber := latestBlockNumber

	const DELAY_IN_MS = 1000

	timer := time.NewTimer(1 * time.Millisecond) // Start immediately.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				for {
					// See if the next block has been created yet.
					if currentBlockNumber > latestBlockNumber {
						latestBlockNumber, err = e.getLatestBlockNumber(ctx)
						if err != nil {
							e.logger.Error("failed to look up latest block", zap.String("eth_network", e.BaseEth.NetworkName),
								zap.Uint64("block", currentBlockNumber), zap.Error(err))
							break
						}

						if currentBlockNumber > latestBlockNumber {
							e.logger.Info("waiting for new latest block", zap.String("eth_network", e.BaseEth.NetworkName),
								zap.Uint64("current_block", currentBlockNumber), zap.Uint64("latest_block", latestBlockNumber))
							break
						}
					}

					// Fetch the hash every time, in case it changes due to a rollback.
					blockHash, err := e.getBlockHash(ctx, currentBlockNumber)
					if err != nil {
						e.logger.Error("failed to look up hash for current block", zap.String("eth_network", e.BaseEth.NetworkName),
							zap.Uint64("block", currentBlockNumber), zap.Error(err))
						break
					}

					e.logger.Info("checking to see if block is finalized", zap.String("eth_network", e.BaseEth.NetworkName),
						zap.Uint64("block", currentBlockNumber), zap.Stringer("hash", blockHash))
					finalized, err := e.isBlockFinalized(ctx, blockHash.Hex())
					if err != nil {
						e.logger.Error("failed to see if block is finalized", zap.String("eth_network", e.BaseEth.NetworkName),
							zap.Uint64("block", currentBlockNumber), zap.Error(err))
						break
					}
					if !finalized {
						break
					}

					e.logger.Info("block is now finalized", zap.String("eth_network", e.BaseEth.NetworkName),
						zap.Uint64("block", currentBlockNumber), zap.Stringer("hash", blockHash))

					ev, err := e.getBlock(ctx, currentBlockNumber)
					if err != nil {
						e.logger.Error("failed to get finalized block", zap.String("eth_network", e.BaseEth.NetworkName),
							zap.Uint64("block", currentBlockNumber), zap.Error(err))
						// Don't break, move on to the next block.
					} else {
						sink <- &common.NewBlock{
							Number: ev.Number,
							Hash:   blockHash,
						}
					}

					currentBlockNumber += 1
				}

				timer = time.NewTimer(DELAY_IN_MS * time.Millisecond)
			}
		}
	}()

	sub := &MoonbeamSubscription{
		err: make(chan error, 1),
	}

	return sub, err
}

func (e *MoonbeamImpl) getLatestBlockNumber(ctx context.Context) (uint64, error) {
	type Marshaller struct {
		Number *ethHexUtils.Big
	}

	var m Marshaller
	err := e.mbRpcCon.CallContext(ctx, &m, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		e.logger.Error("failed to get current block", zap.String("eth_network", e.BaseEth.NetworkName), zap.Error(err))
		return 0, err
	}

	return strconv.ParseUint(m.Number.String()[2:], 16, 64)
}

func (e *MoonbeamImpl) getBlockHash(ctx context.Context, number uint64) (ethCommon.Hash, error) {
	type Marshaller struct {
		Hash ethCommon.Hash `json:"hash"`
	}

	var m Marshaller
	err := e.mbRpcCon.CallContext(ctx, &m, "eth_getBlockByNumber", strconv.FormatUint(number, 10), false)
	if err != nil {
		e.logger.Error("failed to get current block", zap.String("eth_network", e.BaseEth.NetworkName), zap.Error(err))
		return m.Hash, err
	}

	return m.Hash, nil
}

func (e *MoonbeamImpl) isBlockFinalized(ctx context.Context, hash string) (bool, error) {
	var finalized bool
	err := e.mbRpcCon.CallContext(ctx, &finalized, "moon_isBlockFinalized", hash)
	if err != nil {
		e.logger.Error("failed to check for finality", zap.String("eth_network", e.BaseEth.NetworkName), zap.Error(err))
		return false, err
	}

	return finalized, nil
}

func (e *MoonbeamImpl) getBlock(ctx context.Context, number uint64) (*ethTypes.Header, error) {
	var bnum big.Int
	bnum.SetUint64(number)
	ev, err := e.mbClient.HeaderByNumber(ctx, &bnum)
	if err != nil {
		e.logger.Error("failed to get block", zap.String("eth_network", e.BaseEth.NetworkName), zap.Uint64("block", number), zap.Error(err))
		return ev, err
	}

	return ev, nil
}
