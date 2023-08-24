package ccq

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/telemetry"
	"github.com/certusone/wormhole/node/pkg/version"
	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	p2pNetworkID      *string
	p2pPort           *uint
	p2pBootstrap      *string
	listenAddr        *string
	nodeKeyPath       *string
	ethRPC            *string
	ethContract       *string
	logLevel          *string
	telemetryLokiURL  *string
	telemetryNodeName *string
)

func init() {
	p2pNetworkID = QueryServerCmd.Flags().String("network", "/wormhole/dev", "P2P network identifier")
	p2pPort = QueryServerCmd.Flags().Uint("port", 8996, "P2P UDP listener port")
	p2pBootstrap = QueryServerCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (comma-separated)")
	listenAddr = QueryServerCmd.Flags().String("listenAddr", "[::]:6069", "Listen address for query server (disabled if blank)")
	nodeKeyPath = QueryServerCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")
	ethRPC = QueryServerCmd.Flags().String("ethRPC", "", "Ethereum RPC for fetching current guardian set")
	ethContract = QueryServerCmd.Flags().String("ethContract", "", "Ethereum core bridge address for fetching current guardian set")
	logLevel = QueryServerCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")
	telemetryLokiURL = QueryServerCmd.Flags().String("telemetryLokiURL", "", "Loki cloud logging URL")
	telemetryNodeName = QueryServerCmd.Flags().String("telemetryNodeName", "", "Node name used in telemetry")
}

var QueryServerCmd = &cobra.Command{
	Use:   "query-server",
	Short: "Run the cross-chain query server",
	Run:   runQueryServer,
}

func runQueryServer(cmd *cobra.Command, args []string) {
	common.SetRestrictiveUmask()

	// Setup logging
	lvl, err := ipfslog.LevelFromString(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	logger := ipfslog.Logger("query-server").Desugar()
	ipfslog.SetAllLoggers(lvl)

	if *telemetryLokiURL != "" {
		logger.Info("Using Loki telemetry logger")
		if *telemetryNodeName == "" {
			logger.Fatal("if --telemetryLokiURL is specified --telemetryNodeName must be specified")
		}
		labels := map[string]string{
			"network":   *p2pNetworkID,
			"node_name": *telemetryNodeName,
			"version":   version.Version(),
		}

		tm, err := telemetry.NewLokiCloudLogger(context.Background(), logger, *telemetryLokiURL, "ccq_server", true, labels)
		if err != nil {
			logger.Fatal("Failed to initialize telemetry", zap.Error(err))
		}

		defer tm.Close()
		logger = tm.WrapLogger(logger) // Wrap logger with telemetry logger
	}

	// Verify flags
	if *nodeKeyPath == "" {
		logger.Fatal("Please specify --nodeKey")
	}
	if *p2pBootstrap == "" {
		logger.Fatal("Please specify --bootstrap")
	}
	if *ethRPC == "" {
		logger.Fatal("Please specify --ethRPC")
	}
	if *ethContract == "" {
		logger.Fatal("Please specify --ethContract")
	}

	// Load p2p private key
	var priv crypto.PrivKey
	priv, err = common.GetOrCreateNodeKey(logger, *nodeKeyPath)
	if err != nil {
		logger.Fatal("Failed to load node key", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run p2p
	pendingResponses := NewPendingResponses()
	p2p, err := runP2P(ctx, priv, *p2pPort, *p2pNetworkID, *p2pBootstrap, *ethRPC, *ethContract, pendingResponses, logger)
	if err != nil {
		logger.Fatal("Failed to start p2p", zap.Error(err))
	}

	// Start the HTTP server
	go func() {
		s := NewHTTPServer(*listenAddr, p2p.topic_req, pendingResponses)
		logger.Sugar().Infof("Server listening on %s", *listenAddr)
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server closed unexpectedly", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("Context cancelled, exiting...")

	// Cleanly shutdown
	// Without this the same host won't properly discover peers until some timeout
	p2p.sub.Cancel()
	if err := p2p.topic_req.Close(); err != nil {
		logger.Error("Error closing the request topic", zap.Error(err))
	}
	if err := p2p.topic_resp.Close(); err != nil {
		logger.Error("Error closing the response topic", zap.Error(err))
	}
	if err := p2p.host.Close(); err != nil {
		logger.Error("Error closing the host", zap.Error(err))
	}
}
