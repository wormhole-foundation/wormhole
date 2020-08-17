package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	eth_common "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

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

var (
	rootCtx       context.Context
	rootCtxCancel context.CancelFunc
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
	rootCtx, rootCtxCancel = context.WithCancel(context.Background())
	defer rootCtxCancel()

	// Ethereum lock event channel
	ec := make(chan *common.ChainLock)

	// Run supervisor.
	supervisor.New(rootCtx, logger.Desugar(), func(ctx context.Context) error {
		if err := supervisor.Run(ctx, "p2p", p2p); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "eth",
			ethereum.NewEthBridgeWatcher(*ethRPC, ethContractAddr, *ethConfirmations, ec).Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "lockups", func(ctx context.Context) error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case k := <-ec:
					supervisor.Logger(ctx).Info("lockup confirmed",
						zap.String("source", hex.EncodeToString(k.SourceAddress[:])),
						zap.String("target", hex.EncodeToString(k.TargetAddress[:])),
						zap.String("amount", k.Amount.String()),
					)
				}
			}
		}); err != nil {
			return err
		}

		logger.Info("Started internal services")
		supervisor.Signal(ctx, supervisor.SignalHealthy)

		select {
		case <-ctx.Done():
			return nil
		}
	})

	select {
	case <-rootCtx.Done():
		logger.Info("root context cancelled, exiting...")
		// TODO: wait for things to shut down gracefully
	}
}
