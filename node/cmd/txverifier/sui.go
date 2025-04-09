package txverifier

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/certusone/wormhole/node/pkg/telemetry"
	txverifier "github.com/certusone/wormhole/node/pkg/txverifier"
	"github.com/certusone/wormhole/node/pkg/version"

	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	InitialEventFetchLimit = 25
	EventQueryInterval     = 2 * time.Second
)

// CLI args
var (
	suiRPC                  *string
	suiCoreContract         *string
	suiTokenBridgeEmitter   *string
	suiTokenBridgeContract  *string
	suiProcessInitialEvents *bool
)

var TransferVerifierCmdSui = &cobra.Command{
	Use:   "sui",
	Short: "Transfer Verifier for Sui",
	Run:   runTransferVerifierSui,
}

// CLI parameters
//
//nolint:errcheck // The MarkFlagRequired calls will cause the script to fail on their own. No need to handle the errors manually.
func init() {
	suiRPC = TransferVerifierCmdSui.Flags().String("suiRPC", "", "Sui RPC url")
	suiCoreContract = TransferVerifierCmdSui.Flags().String("suiCoreContract", "", "Sui core contract address")
	suiTokenBridgeEmitter = TransferVerifierCmdSui.Flags().String("suiTokenBridgeEmitter", "", "Token bridge emitter on Sui")
	suiTokenBridgeContract = TransferVerifierCmdSui.Flags().String("suiTokenBridgeContract", "", "Token bridge contract on Sui")
	suiProcessInitialEvents = TransferVerifierCmdSui.Flags().Bool("suiProcessInitialEvents", false, "Indicate whether the Sui transfer verifier should process the initial events it fetches")

	TransferVerifierCmd.MarkFlagRequired("suiRPC")
	TransferVerifierCmd.MarkFlagRequired("suiCoreContract")
	TransferVerifierCmd.MarkFlagRequired("suiTokenBridgeEmitter")
	TransferVerifierCmd.MarkFlagRequired("suiTokenBridgeContract")
}

func runTransferVerifierSui(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Setup logging
	lvl, err := ipfslog.LevelFromString(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	logger := ipfslog.Logger("wormhole-transfer-verifier-sui").Desugar()

	ipfslog.SetAllLoggers(lvl)

	// Setup logging to Loki if configured
	if *telemetryLokiUrl != "" && *telemetryNodeName != "" {
		labels := map[string]string{
			// Is this required?
			// "network":   *p2pNetworkID,
			"node_name": *telemetryNodeName,
			"version":   version.Version(),
		}

		tm, err := telemetry.NewLokiCloudLogger(
			context.Background(),
			logger,
			*telemetryLokiUrl,
			"transfer-verifier-sui",
			// Private logs are not used in this code
			false,
			labels,
		)
		if err != nil {
			logger.Fatal("Failed to initialize telemetry", zap.Error(err))
		}

		defer tm.Close()
		logger = tm.WrapLogger(logger) // Wrap logger with telemetry logger
	}

	logger.Info("Starting Sui transfer verifier")
	logger.Debug("Sui rpc connection", zap.String("url", *suiRPC))
	logger.Debug("Sui core contract", zap.String("address", *suiCoreContract))
	logger.Debug("Sui token bridge contract", zap.String("address", *suiTokenBridgeContract))
	logger.Debug("token bridge event emitter", zap.String("object id", *suiTokenBridgeEmitter))
	logger.Debug("process initial events", zap.Bool("processInitialEvents", *suiProcessInitialEvents))

	// Verify CLI parameters
	if *suiRPC == "" || *suiCoreContract == "" || *suiTokenBridgeEmitter == "" || *suiTokenBridgeContract == "" {
		logger.Fatal("One or more CLI parameters are empty",
			zap.String("suiRPC", *suiRPC),
			zap.String("suiCoreContract", *suiCoreContract),
			zap.String("suiTokenBridgeEmitter", *suiTokenBridgeEmitter),
			zap.String("suiTokenBridgeContract", *suiTokenBridgeContract))
	}

	suiApiConnection := txverifier.NewSuiApiConnection(*suiRPC)

	// Create a new SuiTransferVerifier
	suiTransferVerifier := txverifier.NewSuiTransferVerifier(*suiCoreContract, *suiTokenBridgeEmitter, *suiTokenBridgeContract, suiApiConnection)

	// Get the event filter
	eventFilter := suiTransferVerifier.GetEventFilter()

	// Initial event fetching
	resp, err := suiApiConnection.QueryEvents(ctx, eventFilter, "null", InitialEventFetchLimit, true)
	if err != nil {
		logger.Fatal("Error in querying initial events", zap.Error(err))
	}

	initialEvents := resp.Result.Data

	// Use the latest timestamp to determine the starting point for live processing
	var latestTimestamp int
	for _, event := range initialEvents {
		if event.Timestamp != nil {
			timestampInt, err := strconv.Atoi(*event.Timestamp)
			if err != nil {
				logger.Error("Error converting timestamp to int", zap.Error(err))
				continue
			}
			if timestampInt > latestTimestamp {
				latestTimestamp = timestampInt
			}
		}
	}
	logger.Info("Initial events fetched", zap.Int("number of initial events", len(initialEvents)), zap.Int("latestTimestamp", latestTimestamp))

	// If specified, process the initial events. This is useful for running a number of digests
	// through the verifier before starting live processing.
	if *suiProcessInitialEvents {
		logger.Info("Processing initial events")
		for _, event := range initialEvents {
			if event.ID.TxDigest != nil {
				_, err = suiTransferVerifier.ProcessDigest(ctx, *event.ID.TxDigest, logger)
				if err != nil {
					logger.Error(err.Error())
				}
			}
		}
	}

	// Ticker for live processing
	ticker := time.NewTicker(EventQueryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Context cancelled")
		case <-ticker.C:
			// Fetch new events
			resp, err := suiApiConnection.QueryEvents(ctx, eventFilter, "null", 25, true)
			if err != nil {
				logger.Error("Error in querying new events", zap.Error(err))
				continue
			}

			newEvents := resp.Result.Data

			// List of transaction digests for transactions in which the WormholeMessage
			// event was emitted.
			var txDigests []string

			// Iterate over all events and get the transaction digests for events younger
			// than latestTimestamp. Also update latestTimestamp.
			for _, event := range newEvents {
				if event.Timestamp != nil {
					timestampInt, err := strconv.Atoi(*event.Timestamp)
					if err != nil {
						logger.Error("Error converting timestamp to int", zap.Error(err))
						continue
					}
					if timestampInt > latestTimestamp {
						latestTimestamp = timestampInt
						if event.ID.TxDigest != nil {
							txDigests = append(txDigests, *event.ID.TxDigest)
						}
					}
				}
			}

			for _, txDigest := range txDigests {
				_, err := suiTransferVerifier.ProcessDigest(ctx, txDigest, logger)
				if err != nil {
					logger.Error(err.Error())
				}
			}

			logger.Info("New events processed", zap.Int("latestTimestamp", latestTimestamp), zap.Int("txDigestCount", len(txDigests)))

		}
	}
}
