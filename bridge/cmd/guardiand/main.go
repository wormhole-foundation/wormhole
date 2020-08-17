package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/ethereum"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"

	ipfslog "github.com/ipfs/go-log/v2"
)

var (
	p2pNetworkID = flag.String("network", "/wormhole/dev", "P2P network identifier")
	p2pPort      = flag.Uint("port", 8999, "P2P UDP listener port")
	p2pBootstrap = flag.String("bootstrap", "", "P2P bootstrap peers (comma-separated)")

	nodeKeyPath = flag.String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")

	ethRPC           = flag.String("ethRPC", "", "Ethereum RPC URL")
	ethContract      = flag.String("ethContract", "", "Ethereum bridge contract address")
	ethConfirmations = flag.Uint64("ethConfirmations", 15, "Ethereum confirmation count requirement")

	logLevel = flag.String("loglevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")
)

func main() {
	flag.Parse()

	// Set up logging. The go-log zap wrapper that libp2p uses is compatible with our
	// usage of zap in supervisor, which is nice.
	lvl, err := ipfslog.LevelFromString(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	// FIXME: add hostname to root logger for cleaner console output in multi-node development.
	// The proper way is to change the output format to include the hostname.
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	// Our root logger.
	logger := ipfslog.Logger(fmt.Sprintf("%s-%s", "wormhole", hostname))

	// Override the default go-log config, which uses a magic environment variable.
	ipfslog.SetAllLoggers(lvl)

	// Mute chatty subsystems.
	ipfslog.SetLogLevel("swarm2", "error") // connection errors

	// Verify flags
	if *nodeKeyPath == "" {
		logger.Fatal("Please specify -nodeKey")
	}
	if *ethRPC == "" {
		logger.Fatal("Please specify -ethRPC")
	}

	ethContractAddr := eth_common.HexToAddress(*ethContract)

	// Node's main lifecycle context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ethereum lock event channel
	ec := make(chan *common.ChainLock)

	// Run supervisor.
	supervisor.New(ctx, logger.Desugar(), func(ctx context.Context) error {
		if err := supervisor.Run(ctx, "p2p", p2p); err != nil {
			return err
		}

		watcher := ethereum.NewEthBridgeWatcher(
			*ethRPC, ethContractAddr, *ethConfirmations, ec)

		if err := supervisor.Run(ctx, "eth", watcher.Run); err != nil {
			return err
		}

		supervisor.Signal(ctx, supervisor.SignalHealthy)
		logger.Info("Created services")

		select {}
	}, supervisor.WithPropagatePanic)
	// TODO(leo): only propagate panics in debug mode. We currently need this to properly reset p2p
	// (it leaks its socket and we need to restart the process to fix it)

	select {}
}

