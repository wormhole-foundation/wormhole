package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
)

const wsStr = "wss://evm-ws-testnet.sei-apis.com"

// This is for the websocket proxy that we are using in testnet (but which is not robust enough to use in mainnet).
//const wsStr = "ws://localhost:8080"

const contractAddrStr = "0xBB73cB66C26740F31d1FabDC6b7A46a038A300dd"

func main() {
	logger, _ := zap.NewDevelopment()
	logger.Info("Connecting to Sei EVM", zap.String("webSocket", wsStr), zap.String("contractAddr", contractAddrStr))
	ctx := context.Background()

	rawClient, err := ethRpc.DialContext(ctx, wsStr)
	if err != nil {
		logger.Fatal("Failed to connect to RPC", zap.Error(err))
	}

	client := ethClient.NewClient(rawClient)

	logger.Info("Creating filter for log events from contract")
	filterer, err := ethAbi.NewAbiFilterer(ethCommon.BytesToAddress(ethCommon.HexToAddress(contractAddrStr).Bytes()), client)
	if err != nil {
		logger.Fatal("Failed to create filter", zap.Error(err))
	}

	logger.Info("Subscribing to log events from contract")
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	messageC := make(chan *ethabi.AbiLogMessagePublished, 2)
	messageSub, err := filterer.WatchLogMessagePublished(&ethBind.WatchOpts{Context: timeout}, messageC, nil)
	if err != nil {
		logger.Fatal("Failed to subscribe to events", zap.Error(err))
	}
	defer messageSub.Unsubscribe()

	logger.Info("Waiting for log events from contract")
	for {
		select {
		case <-ctx.Done():
			break
		case err := <-messageSub.Err():
			logger.Error("Message subscription failed", zap.Error(err))
			break
		case ev := <-messageC:
			logger.Info("Received a log event from the contract", zap.Any("ev", ev))
		}
	}

}
