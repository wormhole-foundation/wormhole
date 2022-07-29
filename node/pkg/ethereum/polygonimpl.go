// This implements polling for polygon by reading Root Contract on Ethereum.
//
// The root chain proxy contract is deployed on Etherum at the following addresses:
//    Mainnet: 0x86E4Dc95c7FBdBf52e33D563BbDB00823894C287
//    Testnet: 0x2890ba17efe978480615e330ecb65333b880928e
//
// To generate the golang abi for the root chain contract:
// - Grab the ABIs from Root Chain contract (not the proxy) (0x17aD93683697CE557Ef7774660394456A7412B00 on Ethereum mainnet) and put it in /tmp/RootChain.abi.
// - mkdir node/pkg/ethereum/abi_root_chain
// - third_party/abigen/abigen --abi /tmp/RootChain.abi --pkg abi_root_chain --out node/pkg/ethereum/abi_root_chain/RootChain.go

package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethHexUtils "github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethEvent "github.com/ethereum/go-ethereum/event"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	common "github.com/certusone/wormhole/node/pkg/common"
	ethAbi "github.com/certusone/wormhole/node/pkg/ethereum/abi"
	rootAbi "github.com/certusone/wormhole/node/pkg/ethereum/abi_root_chain"

	"go.uber.org/zap"
)

type PolygonImpl struct {
	RootChainUrl     string
	RootChainAddress ethCommon.Address
	BaseEth          EthImpl
	DelayInMs        int
	logger           *zap.Logger
	rawClient        *ethRpc.Client
	rootClient       *ethClient.Client
	rootCaller       *rootAbi.AbiRootChainCaller
}

var (
	POLY_RCP_URL           int = 0
	POLY_RCP_CONTRACT_ADDR int = 1
)

func UsePolygonRootChain(extraParams []string) bool {
	return len(extraParams) >= 2 && extraParams[0] != "" && extraParams[1] != ""
}

func (e *PolygonImpl) SetLogger(l *zap.Logger) {
	e.logger = l
	e.logger.Info("RCP: using root chain proxy to detect new blocks", zap.String("eth_network", e.BaseEth.NetworkName),
		zap.String("url", e.RootChainUrl), zap.Stringer("contract_address", e.RootChainAddress), zap.Int("delay_in_ms", e.DelayInMs))
}

func (e *PolygonImpl) DialContext(ctx context.Context, rawurl string) (err error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// This is used for doing raw eth_ RPC calls.
	e.rawClient, err = ethRpc.DialContext(timeout, rawurl)
	if err != nil {
		return err
	}

	// This is used for querying the Root Chain Proxy contract on Ethereum
	e.rootClient, err = ethClient.DialContext(timeout, e.RootChainUrl)
	if err != nil {
		return err
	}

	// This is used for doing all other go-ethereum calls.
	err = e.BaseEth.DialContext(ctx, rawurl)
	return err
}

func (e *PolygonImpl) NewAbiFilterer(address ethCommon.Address) (err error) {
	return e.BaseEth.NewAbiFilterer(address)
}

func (e *PolygonImpl) NewAbiCaller(address ethCommon.Address) (err error) {
	e.rootCaller, err = rootAbi.NewAbiRootChainCaller(e.RootChainAddress, e.rootClient)
	if err != nil {
		return err
	}
	return e.BaseEth.NewAbiCaller(address)
}

func (e *PolygonImpl) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	return e.BaseEth.GetCurrentGuardianSetIndex(ctx)
}

func (e *PolygonImpl) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	return e.BaseEth.GetGuardianSet(ctx, index)
}

func (e *PolygonImpl) WatchLogMessagePublished(ctx, timeout context.Context, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	return e.BaseEth.WatchLogMessagePublished(ctx, timeout, sink)
}

func (e *PolygonImpl) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	return e.BaseEth.TransactionReceipt(ctx, txHash)
}

func (e *PolygonImpl) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	return e.BaseEth.TimeOfBlockByHash(ctx, hash)
}

func (e *PolygonImpl) ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	return e.BaseEth.ParseLogMessagePublished(log)
}

type PolygonPollSubscription struct {
	errOnce   sync.Once
	err       chan error
	quit      chan error
	unsubDone chan struct{}
}

var ErrUnsubscribedForPolygonPolling = errors.New("unsubscribed")

func (sub *PolygonPollSubscription) Err() <-chan error {
	return sub.err
}

func (sub *PolygonPollSubscription) Unsubscribe() {
	sub.errOnce.Do(func() {
		select {
		case sub.quit <- ErrUnsubscribedForPolygonPolling:
			<-sub.unsubDone
		case <-sub.unsubDone:
		}
		close(sub.err)
	})
}

func (e *PolygonImpl) SubscribeForBlocks(ctx context.Context, sink chan<- *common.NewBlock) (ethereum.Subscription, error) {
	if e.BaseEth.client == nil {
		panic("client is not initialized!")
	}
	if e.rawClient == nil {
		panic("rawClient is not initialized!")
	}
	if e.rootClient == nil {
		panic("root client is not initialized!")
	}
	if e.rootCaller == nil {
		panic("root caller is not initialized!")
	}

	sub := &PolygonPollSubscription{
		err: make(chan error, 1),
	}

	opts := &ethBind.CallOpts{Context: ctx}
	lastConfirmedBlockNum, err := e.rootCaller.GetLastChildBlock(opts)
	if err != nil {
		return nil, err
	}

	e.logger.Info("RCP: initial block", zap.String("eth_network", e.BaseEth.NetworkName), zap.Stringer("block", lastConfirmedBlockNum))

	var BIG_ONE = big.NewInt(1)

	// We would like to generate one block immediately.
	lastConfirmedBlockNum.Sub(lastConfirmedBlockNum, BIG_ONE)

	timer := time.NewTimer(time.Millisecond) // Start immediately.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				opts := &ethBind.CallOpts{Context: ctx}
				newConfirmedBlockNum, err := e.rootCaller.GetLastChildBlock(opts)
				if err != nil {
					e.logger.Error("failed to look up latest block", zap.String("eth_network", e.BaseEth.NetworkName), zap.Error(err))
					sub.err <- fmt.Errorf("failed to read latest block: %w", err)
					break
				}

				if newConfirmedBlockNum.Cmp(lastConfirmedBlockNum) > 0 {
					e.logger.Info("RCP: detected new checkpoint", zap.String("eth_network", e.BaseEth.NetworkName), zap.Stringer("block", newConfirmedBlockNum))
					blockNum := big.NewInt(0)
					blockNum.Add(lastConfirmedBlockNum, BIG_ONE)
					for !(blockNum.Cmp(newConfirmedBlockNum) > 0) {
						n := big.Int(*blockNum)
						txHash, err := e.getTxHash(ctx, &n)
						if err != nil {
							e.logger.Error("failed to look up tx hash", zap.String("eth_network", e.BaseEth.NetworkName), zap.Error(err))
							sub.err <- fmt.Errorf("failed to read tx hash: %w", err)
							break
						}

						block := &common.NewBlock{
							Number: &n,
							Hash:   *txHash,
						}

						sink <- block

						blockNum.Add(blockNum, BIG_ONE)
					}

					lastConfirmedBlockNum = newConfirmedBlockNum
				}

				timer = time.NewTimer(time.Duration(e.DelayInMs) * time.Millisecond)
			}
		}
	}()

	return sub, err
}

func (e *PolygonImpl) getTxHash(ctx context.Context, number *big.Int) (*ethCommon.Hash, error) {
	numStr := ethHexUtils.EncodeBig(number)
	type Marshaller struct {
		Hash ethCommon.Hash `json:"hash"`
	}

	var m Marshaller
	err := e.rawClient.CallContext(ctx, &m, "eth_getBlockByNumber", numStr, false)
	if err != nil {
		e.logger.Error("RCP: failed to get block", zap.String("eth_network", e.BaseEth.NetworkName),
			zap.String("requested_block", numStr), zap.Error(err))
		return nil, err
	}

	return &m.Hash, nil
}
