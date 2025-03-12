// This tool can be used to verify that an EVM endpoint works properly with go-ethereum websocket subscriptions.
// It can subscribe to latest blocks as well as log events from the Wormhole core contract and just logs them out.
//
// To run this tool, do:
//   go run wstest.go --rpc <websocketEndpoint> [--contract <wormholeCodeContractAddress>] [--blocks]
//
// where
//   --contract` subscribes to log events from the specified Wormhole core contract
//   --blocks subscribes to the latest blocks.
//
// To listen to log events from the SeiEVM test endpoint (what this was originally written for) do:
//   go run wstest.go --rpc wss://evm-ws-testnet.sei-apis.com --contract 0xBB73cB66C26740F31d1FabDC6b7A46a038A300dd

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
)

var (
	rpc      = flag.String("rpc", "", "Websocket URL, this parameter is required")
	contract = flag.String("contract", "", "Core contract address, leave blank to not subscribe to the core contract")
	blocks   = flag.Bool("blocks", false, "Also subscribe to new blocks, default is false")
)

func main() {
	flag.Parse()
	logger, _ := zap.NewDevelopment()
	if *rpc == "" {
		logger.Fatal(`The "--rpc" parameter is required`)
	}
	if *contract == "" && !*blocks {
		logger.Fatal(`Must specify either "--contract" or "--blocks" or both`)
	}

	logger.Info("Connecting to websocket endpoint", zap.String("webSocket", *rpc))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rawClient, err := ethRpc.DialContext(ctx, *rpc)
	if err != nil {
		logger.Fatal("Failed to connect to RPC", zap.Error(err))
	}

	client := ethClient.NewClient(rawClient)

	errC := make(chan error)

	if *blocks {
		logger.Info("Subscribing for latest blocks")
		headSink := make(chan *ethTypes.Header, 2)
		headerSubscription, err := client.SubscribeNewHead(ctx, headSink)
		if err != nil {
			logger.Fatal("Failed to subscribe to latest blocks", zap.Error(err))
		}

		go func() {
			logger.Info("Waiting for latest block events")
			defer headerSubscription.Unsubscribe()
			for {
				select {
				case <-ctx.Done():
					return
				case err := <-headerSubscription.Err():
					errC <- fmt.Errorf("block subscription failed: %w", err) // nolint:channelcheck // The watcher will exit anyway
					return
				case block := <-headSink:
					// These two pointers should have been checked before the event was placed on the channel, but just being safe.
					if block == nil {
						logger.Error("New header event is nil")
						continue
					}
					logger.Info("Received a new block", zap.Any("block", block))
				}
			}
		}()
	}

	if *contract != "" {
		logger.Info("Subscribing to log events from contract", zap.String("contractAddr", *contract))
		filterer, err := ethAbi.NewAbiFilterer(ethCommon.BytesToAddress(ethCommon.HexToAddress(*contract).Bytes()), client)
		if err != nil {
			logger.Fatal("Failed to create filter", zap.Error(err))
		}

		timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		messageC := make(chan *ethAbi.AbiLogMessagePublished, 2)
		messageSub, err := filterer.WatchLogMessagePublished(&ethBind.WatchOpts{Context: timeout}, messageC, nil)
		if err != nil {
			logger.Fatal("Failed to subscribe to events", zap.Error(err))
		}
		defer messageSub.Unsubscribe()

		logger.Info("Waiting for log events from contract")
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case err := <-messageSub.Err():
					errC <- fmt.Errorf("message subscription failed: %w", err) // nolint:channelcheck // The watcher will exit anyway
					return
				case ev := <-messageC:
					logger.Info("Received a log event from the contract", zap.Any("ev", ev))
				}
			}
		}()
	}

	// Wait for SIGTERM.
	logger.Info("Waiting for sigterm.")
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	go func() {
		<-sigterm
		logger.Info("Received sigterm. exiting.")
		cancel()
	}()

	// Wait for either a shutdown or a fatal error from the permissions watcher.
	select {
	case <-ctx.Done():
		logger.Info("Context cancelled, exiting...")
		break
	case err := <-errC:
		logger.Error("Encountered an error, exiting", zap.Error(err))
		break
	}

}
