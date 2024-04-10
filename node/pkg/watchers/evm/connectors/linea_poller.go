// A block is considered finalized on Linea when it is marked finalized by the LineaRollup contract on Ethereum.
//
// For a discussion of finality on Linea, see here:
//    https://www.notion.so/wormholefoundation/Testnet-Info-V2-633e4aa64a634d56a7ce07a103789774?pvs=4#03513c2eb3654d33aff2206a562d25b1
//
// The LineaRollup proxy contract on ethereum is available at the following addresses:
//    Mainnet: 0xd19d4B5d358258f05D7B411E21A1460D11B0876F
//    Testnet: 0xB218f8A4Bc926cF1cA7b3423c154a0D627Bdb7E5
//
// To generate the golang abi for the LineaRollup contract:
// - Grab the ABIs from the LineaRollup contract (not the proxy) (0x934Dd4C63E285551CEceF8459103554D0096c179 on Ethereum mainnet) and put it in /tmp/LineaRollup.abi.
// - mkdir node/pkg/watchers/evm/connectors/lineaabi
// - Install abigen: go install github.com/ethereum/go-ethereum/cmd/abigen@latest
// - abigen --abi /tmp/LineaRollup.abi --pkg lineaabi --out node/pkg/watchers/evm/connectors/lineaabi/LineaRollup.go

package connectors

import (
	"context"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	rollUpAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/lineaabi"

	ethereum "github.com/ethereum/go-ethereum"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	"go.uber.org/zap"
)

// LineaConnector listens for new finalized blocks for Linea by reading the roll up contract on Ethereum.
type LineaConnector struct {
	Connector
	logger *zap.Logger

	// These are used for querying the roll up contract.
	rollUpRawClient *ethRpc.Client
	rollUpClient    *ethClient.Client

	// These are used to subscribe for new block finalized events from the roll up contract.
	rollUpFilterer *rollUpAbi.LineaabiFilterer
	rollUpCaller   *rollUpAbi.LineaabiCaller

	latestBlockNum          uint64
	latestFinalizedBlockNum uint64
}

// NewLineaConnector creates a new Linea poll connector using the specified roll up contract.
func NewLineaConnector(
	ctx context.Context,
	logger *zap.Logger,
	baseConnector Connector,
	rollUpUrl string,
	rollUpAddress string,
) (*LineaConnector, error) {

	rollUpRawClient, err := ethRpc.DialContext(ctx, rollUpUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create roll up raw client for url %s: %w", rollUpUrl, err)
	}

	rollUpClient := ethClient.NewClient(rollUpRawClient)

	addr := ethCommon.HexToAddress(rollUpAddress)
	rollUpFilterer, err := rollUpAbi.NewLineaabiFilterer(addr, rollUpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create roll up filter for url %s: %w", rollUpUrl, err)
	}

	rollUpCaller, err := rollUpAbi.NewLineaabiCaller(addr, rollUpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create roll up caller for url %s: %w", rollUpUrl, err)
	}

	logger.Info("Using roll up for Linea", zap.String("rollUpUrl", rollUpUrl), zap.String("rollUpAddress", rollUpAddress))

	connector := &LineaConnector{
		Connector:       baseConnector,
		logger:          logger,
		rollUpRawClient: rollUpRawClient,
		rollUpClient:    rollUpClient,
		rollUpFilterer:  rollUpFilterer,
		rollUpCaller:    rollUpCaller,
	}

	return connector, nil
}

// SubscribeForBlocks starts polling. It implements the standard connector interface.
func (c *LineaConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Use the standard geth head sink to get latest blocks.
	headSink := make(chan *ethTypes.Header, 2)
	headerSubscription, err := c.Connector.Client().SubscribeNewHead(ctx, headSink)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe for latest blocks: %w", err)
	}

	// Subscribe to data finalized events from the roll up contract.
	dataFinalizedChan := make(chan *rollUpAbi.LineaabiDataFinalized, 2)
	dataFinalizedSub, err := c.rollUpFilterer.WatchDataFinalized(&ethBind.WatchOpts{Context: timeout}, dataFinalizedChan, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe for events from roll up contract: %w", err)
	}

	// Get the current latest block on Linea.
	latestBlock, err := GetBlockByFinality(timeout, c.logger, c.Connector, Latest)
	if err != nil {
		return nil, fmt.Errorf("failed to get current latest block: %w", err)
	}
	c.latestBlockNum = latestBlock.Number.Uint64()

	// Get and publish the current latest finalized block.
	opts := &ethBind.CallOpts{Context: timeout}
	initialBlock, err := c.rollUpCaller.CurrentL2BlockNumber(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial block: %w", err)
	}
	c.latestFinalizedBlockNum = initialBlock.Uint64()

	if c.latestFinalizedBlockNum > c.latestBlockNum {
		return nil, fmt.Errorf("latest finalized block reported by L1 (%d) is ahead of latest block reported by L2 (%d), L2 node seems to be stuck",
			c.latestFinalizedBlockNum, c.latestBlockNum)
	}

	c.logger.Info("queried initial finalized block", zap.Uint64("initialBlock", c.latestFinalizedBlockNum), zap.Uint64("latestBlock", c.latestBlockNum))
	if err = c.postFinalizedAndSafe(ctx, c.latestFinalizedBlockNum, sink); err != nil {
		return nil, fmt.Errorf("failed to post initial block: %w", err)
	}

	common.RunWithScissors(ctx, errC, "linea_block_poller", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				dataFinalizedSub.Unsubscribe()
				return nil
			case err := <-dataFinalizedSub.Err():
				errC <- fmt.Errorf("finalized data watcher posted an error: %w", err)
				dataFinalizedSub.Unsubscribe()
				return nil
			case evt := <-dataFinalizedChan:
				if err := c.processDataFinalizedEvent(ctx, sink, evt); err != nil {
					errC <- fmt.Errorf("failed to process block finalized event: %w", err)
					dataFinalizedSub.Unsubscribe()
					return nil
				}
			case ev := <-headSink:
				if ev == nil {
					c.logger.Error("new latest header event is nil")
					continue
				}
				if ev.Number == nil {
					c.logger.Error("new latest header block number is nil")
					continue
				}
				c.latestBlockNum = ev.Number.Uint64()
				sink <- &NewBlock{
					Number:   ev.Number,
					Time:     ev.Time,
					Hash:     ev.Hash(),
					Finality: Latest,
				}
			}
		}
	})

	return headerSubscription, nil
}

// processDataFinalizedEvent handles a DataFinalized event published by the roll up contract.
func (c *LineaConnector) processDataFinalizedEvent(ctx context.Context, sink chan<- *NewBlock, evt *rollUpAbi.LineaabiDataFinalized) error {
	latestFinalizedBlockNum := evt.LastBlockFinalized.Uint64()
	// Leaving this log info in for now because these events come very infrequently.
	c.logger.Info("processing data finalized event",
		zap.Uint64("latestFinalizedBlockNum", latestFinalizedBlockNum),
		zap.Uint64("prevFinalizedBlockNum", c.latestFinalizedBlockNum),
	)

	if latestFinalizedBlockNum > c.latestBlockNum {
		return fmt.Errorf("latest finalized block reported by L1 (%d) is ahead of latest block reported by L2 (%d), L2 node seems to be stuck",
			latestFinalizedBlockNum, c.latestBlockNum)
	}

	for blockNum := c.latestFinalizedBlockNum + 1; blockNum <= latestFinalizedBlockNum; blockNum++ {
		if err := c.postFinalizedAndSafe(ctx, blockNum, sink); err != nil {
			c.latestFinalizedBlockNum = blockNum - 1
			return fmt.Errorf("failed to post block %d: %w", blockNum, err)
		}
	}

	c.latestFinalizedBlockNum = latestFinalizedBlockNum
	return nil
}

// postFinalizedAndSafe publishes a block as finalized and safe. It takes a block number and looks it up on chain to publish the current values.
func (c *LineaConnector) postFinalizedAndSafe(ctx context.Context, blockNum uint64, sink chan<- *NewBlock) error {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	block, err := GetBlockByNumberUint64(timeout, c.logger, c.Connector, blockNum, Finalized)
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", blockNum, err)
	}

	// Publish the finalized block.
	sink <- block

	// Publish same thing for the safe block.
	sink <- block.Copy(Safe)
	return nil
}
