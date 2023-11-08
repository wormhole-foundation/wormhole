// Note: To generate a signer key file do: guardiand keygen --block-type "CCQ SERVER SIGNING KEY" /path/to/key/file
// You will need to add this key to ccqAllowedRequesters in the guardian configs.

package ccq

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"os"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/telemetry"
	"github.com/certusone/wormhole/node/pkg/version"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const CCQ_SERVER_SIGNING_KEY = "CCQ SERVER SIGNING KEY"

var (
	envStr            *string
	p2pNetworkID      *string
	p2pPort           *uint
	p2pBootstrap      *string
	listenAddr        *string
	nodeKeyPath       *string
	signerKeyPath     *string
	permFile          *string
	ethRPC            *string
	ethContract       *string
	logLevel          *string
	telemetryLokiURL  *string
	telemetryNodeName *string
	statusAddr        *string
)

const DEV_NETWORK_ID = "/wormhole/dev"

func init() {
	envStr = QueryServerCmd.Flags().String("env", "", "environment (dev, test, prod)")
	p2pNetworkID = QueryServerCmd.Flags().String("network", DEV_NETWORK_ID, "P2P network identifier")
	p2pPort = QueryServerCmd.Flags().Uint("port", 8995, "P2P UDP listener port")
	p2pBootstrap = QueryServerCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (comma-separated)")
	nodeKeyPath = QueryServerCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")
	signerKeyPath = QueryServerCmd.Flags().String("signerKey", "", "Path to key used to sign unsigned queries")
	listenAddr = QueryServerCmd.Flags().String("listenAddr", "[::]:6069", "Listen address for query server (disabled if blank)")
	permFile = QueryServerCmd.Flags().String("permFile", "", "JSON file containing permissions configuration")
	ethRPC = QueryServerCmd.Flags().String("ethRPC", "", "Ethereum RPC for fetching current guardian set")
	ethContract = QueryServerCmd.Flags().String("ethContract", "", "Ethereum core bridge address for fetching current guardian set")
	logLevel = QueryServerCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")
	telemetryLokiURL = QueryServerCmd.Flags().String("telemetryLokiURL", "", "Loki cloud logging URL")
	telemetryNodeName = QueryServerCmd.Flags().String("telemetryNodeName", "", "Node name used in telemetry")
	statusAddr = QueryServerCmd.Flags().String("statusAddr", "[::]:6060", "Listen address for status server (disabled if blank)")
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

	env, err := common.ParseEnvironment(*envStr)
	if err != nil || (env != common.UnsafeDevNet && env != common.TestNet && env != common.MainNet) {
		if *envStr == "" {
			logger.Fatal("Please specify --env")
		}
		logger.Fatal("Invalid value for --env, must be dev, test or prod", zap.String("val", *envStr))
	}

	if *p2pNetworkID == DEV_NETWORK_ID && env != common.UnsafeDevNet {
		logger.Fatal("May not set --network to dev unless --env is also dev", zap.String("network", *p2pNetworkID), zap.String("env", *envStr))
	}

	// Verify flags
	if *nodeKeyPath == "" {
		logger.Fatal("Please specify --nodeKey")
	}
	if *p2pBootstrap == "" {
		logger.Fatal("Please specify --bootstrap")
	}
	if *permFile == "" {
		logger.Fatal("Please specify --permFile")
	}
	if *ethRPC == "" {
		logger.Fatal("Please specify --ethRPC")
	}
	if *ethContract == "" {
		logger.Fatal("Please specify --ethContract")
	}

	permissions, err := parseConfigFile(*permFile)
	if err != nil {
		logger.Fatal("Failed to load permissions file", zap.String("permFile", *permFile), zap.Error(err))
	}

	// Load p2p private key
	var priv crypto.PrivKey
	priv, err = common.GetOrCreateNodeKey(logger, *nodeKeyPath)
	if err != nil {
		logger.Fatal("Failed to load node key", zap.Error(err))
	}

	var signerKey *ecdsa.PrivateKey
	if *signerKeyPath != "" {
		signerKey, err = common.LoadArmoredKey(*signerKeyPath, CCQ_SERVER_SIGNING_KEY, false)
		if err != nil {
			logger.Fatal("Failed to loader signer key", zap.Error(err))
		}

		logger.Info("will sign unsigned requests if api key supports it", zap.Stringer("signingKey", ethCrypto.PubkeyToAddress(signerKey.PublicKey)))
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
		s := NewHTTPServer(*listenAddr, p2p.topic_req, permissions, signerKey, pendingResponses, logger, env)
		logger.Sugar().Infof("Server listening on %s", *listenAddr)
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server closed unexpectedly", zap.Error(err))
		}
	}()

	// Start the status server
	if *statusAddr != "" {
		go func() {
			ss := NewStatusServer(*statusAddr, logger, env)
			logger.Sugar().Infof("Status server listening on %s", *statusAddr)
			err := ss.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.Fatal("Status server closed unexpectedly", zap.Error(err))
			}
		}()
	}

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
