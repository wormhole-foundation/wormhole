// On Polygon, a block is considered finalized when it is checkpointed on Ethereum.
// This requires listening to the RootChain contract on Ethereum.
//
// For a discussion on Polygon finality, see here:
//    https://wiki.polygon.technology/docs/pos/heimdall/modules/checkpoint
//
// The RootChain proxy contract on Ethereum is available at the following addresses:
//    Mainnet: 0x86E4Dc95c7FBdBf52e33D563BbDB00823894C287
//    Testnet: 0x2890ba17efe978480615e330ecb65333b880928e
//
// The code for the RootChain contract is available here:
//    https://github.com/maticnetwork/contracts/tree/main/contracts
//
// To generate the golang abi for the root chain contract:
// - Grab the ABIs from the Root Chain contract (not the proxy) (0x17aD93683697CE557Ef7774660394456A7412B00 on Ethereum mainnet) and put it in /tmp/RootChain.abi.
// - mkdir node/pkg/watchers/evm/connectors/polygonabi
// - third_party/abigen/abigen --abi /tmp/RootChain.abi --pkg polygonabi --out node/pkg/watchers/evm/connectors/polygonabi/RootChain.go

package connectors

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	rootAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/polygonabi"

	ethereum "github.com/ethereum/go-ethereum"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	"go.uber.org/zap"
)

type PolygonConnector struct {
	Connector
	logger *zap.Logger

	// These are used for querying the root chain contract.
	rootRawClient *ethRpc.Client
	rootClient    *ethClient.Client

	// These are used to subscribe for new checkpoint events from the root chain contract.
	rootFilterer *rootAbi.AbiRootChainFilterer
	rootCaller   *rootAbi.AbiRootChainCaller
}

func NewPolygonConnector(
	ctx context.Context,
	baseConnector Connector,
	rootChainUrl string,
	rootChainAddress string,
) (*PolygonConnector, error) {

	rootRawClient, err := ethRpc.DialContext(ctx, rootChainUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create root chain raw client for url %s: %w", rootChainUrl, err)
	}

	rootClient := ethClient.NewClient(rootRawClient)

	addr := ethCommon.HexToAddress(rootChainAddress)
	rootFilterer, err := rootAbi.NewAbiRootChainFilterer(addr, rootClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create root chain filter for url %s: %w", rootChainUrl, err)
	}

	rootCaller, err := rootAbi.NewAbiRootChainCaller(addr, rootClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create root chain caller for url %s: %w", rootChainUrl, err)
	}

	logger := supervisor.Logger(ctx).With(zap.String("eth_network", baseConnector.NetworkName()))
	logger.Info("Using checkpointing for Polygon", zap.String("rootChainUrl", rootChainUrl), zap.String("rootChainAddress", rootChainAddress))

	connector := &PolygonConnector{
		Connector:     baseConnector,
		logger:        logger,
		rootRawClient: rootRawClient,
		rootClient:    rootClient,
		rootFilterer:  rootFilterer,
		rootCaller:    rootCaller,
	}

	return connector, nil
}

func (c *PolygonConnector) SubscribeForBlocks(ctx context.Context, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	sub := NewPollSubscription()
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Subscribe to new checkpoint events from the root chain contract.
	messageC := make(chan *rootAbi.AbiRootChainNewHeaderBlock, 2)
	messageSub, err := c.rootFilterer.WatchNewHeaderBlock(&ethBind.WatchOpts{Context: timeout}, messageC, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new checkpoint watcher: %w", err)
	}

	// Get and publish the current latest block.
	opts := &ethBind.CallOpts{Context: ctx}
	initialBlock, err := c.rootCaller.GetLastChildBlock(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial block: %w", err)
	}

	if err = c.postBlock(ctx, initialBlock, sink); err != nil {
		return nil, fmt.Errorf("failed to post initial block: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-messageSub.Err():
				sub.err <- err
			case checkpoint := <-messageC:
				if err := c.processCheckpoint(ctx, sink, checkpoint); err != nil {
					sub.err <- fmt.Errorf("failed to process checkpoint: %w", err)
				}
			}
		}
	}()

	return sub, nil
}

var bigOne = big.NewInt(1)

func (c *PolygonConnector) processCheckpoint(ctx context.Context, sink chan<- *NewBlock, checkpoint *rootAbi.AbiRootChainNewHeaderBlock) error {
	for blockNum := checkpoint.Start; blockNum.Cmp(checkpoint.End) <= 0; blockNum.Add(blockNum, bigOne) {
		if err := c.postBlock(ctx, blockNum, sink); err != nil {
			return fmt.Errorf("failed to post block %s: %w", blockNum.String(), err)
		}
	}

	return nil
}

func (c *PolygonConnector) postBlock(ctx context.Context, blockNum *big.Int, sink chan<- *NewBlock) error {
	if blockNum == nil {
		return fmt.Errorf("blockNum is nil")
	}

	block, err := getBlock(ctx, c.logger, c.Connector, blockNum, false)
	if err != nil {
		return fmt.Errorf("failed to get block %s: %w", blockNum.String(), err)
	}

	sink <- block
	return nil
}
