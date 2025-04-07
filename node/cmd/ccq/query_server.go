// Note: To generate a signer key file do: guardiand keygen --block-type "CCQ SERVER SIGNING KEY" /path/to/key/file
// You will need to add this key to ccqAllowedRequesters in the guardian configs.

package ccq

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/telemetry"
	promremotew "github.com/certusone/wormhole/node/pkg/telemetry/prom_remote_write"
	"github.com/certusone/wormhole/node/pkg/version"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const CCQ_SERVER_SIGNING_KEY = "CCQ SERVER SIGNING KEY"

var (
	envStr                 *string
	p2pNetworkID           *string
	p2pPort                *uint
	p2pBootstrap           *string
	protectedPeers         []string
	listenAddr             *string
	nodeKeyPath            *string
	signerKeyPath          *string
	permFile               *string
	ethRPC                 *string
	ethContract            *string
	logLevel               *string
	telemetryLokiURL       *string
	telemetryNodeName      *string
	statusAddr             *string
	promRemoteURL          *string
	shutdownDelay1         *uint
	shutdownDelay2         *uint
	monitorPeers           *bool
	gossipAdvertiseAddress *string
	verifyPermissions      *bool
)

const DEV_NETWORK_ID = "/wormhole/dev"

func init() {
	envStr = QueryServerCmd.Flags().String("env", "", "environment (devnet, testnet, mainnet)")
	p2pNetworkID = QueryServerCmd.Flags().String("network", "", "P2P network identifier (optional, overrides default for environment)")
	p2pPort = QueryServerCmd.Flags().Uint("port", 8995, "P2P UDP listener port")
	p2pBootstrap = QueryServerCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (optional for testnet or mainnet, overrides default, required for devnet)")
	QueryServerCmd.Flags().StringSliceVarP(&protectedPeers, "protectedPeers", "", []string{}, "")
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
	promRemoteURL = QueryServerCmd.Flags().String("promRemoteURL", "", "Prometheus remote write URL (Grafana)")
	monitorPeers = QueryServerCmd.Flags().Bool("monitorPeers", false, "Should monitor bootstrap peers and attempt to reconnect")
	gossipAdvertiseAddress = QueryServerCmd.Flags().String("gossipAdvertiseAddress", "", "External IP to advertize on P2P (use if behind a NAT or running in k8s)")
	verifyPermissions = QueryServerCmd.Flags().Bool("verifyPermissions", false, `parse and verify the permissions file and then exit with 0 if success, 1 if failure`)

	// The default health check monitoring is every five seconds, with a five second timeout, and you have to miss two, for 20 seconds total.
	shutdownDelay1 = QueryServerCmd.Flags().Uint("shutdownDelay1", 25, "Seconds to delay after disabling health check on shutdown")

	// The guardians will wait up to 60 seconds before giving up on a request.
	shutdownDelay2 = QueryServerCmd.Flags().Uint("shutdownDelay2", 65, "Seconds to wait after delay1 for pending requests to complete")
}

var QueryServerCmd = &cobra.Command{
	Use:   "query-server",
	Short: "Run the cross-chain query server",
	Run:   runQueryServer,
}

func runQueryServer(cmd *cobra.Command, args []string) {
	env, err := common.ParseEnvironment(*envStr)
	if err != nil || (env != common.UnsafeDevNet && env != common.TestNet && env != common.MainNet) {
		if *envStr == "" {
			fmt.Println("Please specify --env")
		} else {
			fmt.Println("Invalid value for --env, should be devnet, testnet or mainnet", zap.String("val", *envStr))
		}
		os.Exit(1)
	}

	if *verifyPermissions {
		_, err := parseConfigFile(*permFile, env)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	common.SetRestrictiveUmask()

	// Setup logging
	lvl, err := ipfslog.LevelFromString(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	logger := ipfslog.Logger("query-server").Desugar()
	ipfslog.SetAllLoggers(lvl)

	if *p2pNetworkID == "" {
		*p2pNetworkID = p2p.GetNetworkId(env)
	} else if env != common.UnsafeDevNet {
		logger.Warn("overriding default p2p network ID", zap.String("p2pNetworkID", *p2pNetworkID))
	}

	if *p2pNetworkID == DEV_NETWORK_ID && env != common.UnsafeDevNet {
		logger.Fatal("May not set --network to dev unless --env is also dev", zap.String("network", *p2pNetworkID), zap.String("env", *envStr))
	}

	networkID := *p2pNetworkID + "/ccq"

	if *p2pBootstrap == "" {
		*p2pBootstrap, err = p2p.GetCcqBootstrapPeers(env)
		if err != nil {
			logger.Fatal("failed to determine the bootstrap peers from the environment", zap.String("env", string(env)), zap.Error(err))
		}
	} else if env != common.UnsafeDevNet {
		logger.Warn("overriding default p2p bootstrap peers", zap.String("p2pBootstrap", *p2pBootstrap))
	}

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
	if *permFile == "" {
		logger.Fatal("Please specify --permFile")
	}
	if *ethRPC == "" {
		logger.Fatal("Please specify --ethRPC")
	}
	if *ethContract == "" {
		logger.Fatal("Please specify --ethContract")
	}

	permissions, err := NewPermissions(*permFile, env)
	if err != nil {
		logger.Fatal("Failed to load permissions file", zap.String("permFile", *permFile), zap.Error(err))
	}

	loggingMap := NewLoggingMap()

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
	pendingResponses := NewPendingResponses(logger)
	p2p, err := runP2P(ctx, priv, *p2pPort, networkID, *p2pBootstrap, *ethRPC, *ethContract, pendingResponses, logger, *monitorPeers, loggingMap, *gossipAdvertiseAddress, protectedPeers)
	if err != nil {
		logger.Fatal("Failed to start p2p", zap.Error(err))
	}

	// Start the HTTP server
	go func() {
		s := NewHTTPServer(*listenAddr, p2p.topic_req, permissions, signerKey, pendingResponses, logger, env, loggingMap)
		logger.Sugar().Infof("Server listening on %s", *listenAddr)
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server closed unexpectedly", zap.Error(err))
		}
	}()

	// Start the status server
	var statServer *statusServer
	if *statusAddr != "" {
		statServer = NewStatusServer(*statusAddr, logger, env)
		go func() {
			logger.Sugar().Infof("Status server listening on %s", *statusAddr)
			err := statServer.httpServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.Fatal("Status server closed unexpectedly", zap.Error(err))
			}
		}()
	}

	// Start the Prometheus scraper
	usingPromRemoteWrite := *promRemoteURL != ""
	if usingPromRemoteWrite {
		var info promremotew.PromTelemetryInfo
		info.PromRemoteURL = *promRemoteURL
		info.Labels = map[string]string{
			"node_name": *telemetryNodeName,
			"network":   *p2pNetworkID,
			"version":   version.Version(),
			"product":   "ccq_server",
		}

		err := RunPrometheusScraper(ctx, logger, info)
		if err != nil {
			logger.Fatal("Failed to start prometheus scraper", zap.Error(err))
		}
	}

	// Handle SIGTERM
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	go func() {
		<-sigterm
		if statServer != nil && *shutdownDelay1 != 0 {
			logger.Info("Received sigterm. disabling health checks and pausing.")
			statServer.disableHealth()
			time.Sleep(time.Duration(*shutdownDelay1) * time.Second) // #nosec G115 -- Defaults to 25 seconds, overflowing is infeasible
			numPending := 0
			logger.Info("Waiting for any outstanding requests to complete before shutting down.")
			// #nosec G115 -- Defaults to 65 seconds, overflowing is infeasible
			for count := 0; count < int(*shutdownDelay2); count++ {
				time.Sleep(time.Second)
				numPending = pendingResponses.NumPending()
				if numPending == 0 {
					break
				}
			}
			if numPending == 0 {
				logger.Info("Done waiting. shutting down.")
			} else {
				logger.Error("Gave up waiting for pending requests to finish. shutting down anyway.", zap.Int("numStillPending", numPending))
			}
		} else {
			logger.Info("Received sigterm. exiting.")
		}
		cancel()
	}()

	// Start watching for permissions file updates.
	errC := make(chan error)
	permissions.StartWatcher(ctx, logger, errC)

	// Star logging cleanup process.
	loggingMap.Start(ctx, logger, errC)

	// Wait for either a shutdown or a fatal error from the permissions watcher.
	select {
	case <-ctx.Done():
		logger.Info("Context cancelled, exiting...")
		break
	case err := <-errC:
		logger.Error("Encountered an error, exiting", zap.Error(err))
		break
	}

	// Stop the permissions file watcher.
	permissions.StopWatcher()

	// Shutdown p2p. Without this the same host won't properly discover peers until some timeout
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
