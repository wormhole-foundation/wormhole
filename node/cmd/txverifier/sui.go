package txverifier

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	suiCoreContract         string
	suiTokenBridgeEmitter   string
	suiTokenBridgeAddress   string
	suiProcessInitialEvents *bool
	suiEnvironment          *string
	suiDigest               *string

	suiCoreBridgeStateObjectId  *string
	suiTokenBridgeStateObjectId *string
)

var TransferVerifierCmdSui = &cobra.Command{
	Use:   "sui",
	Short: "Transfer Verifier for Sui",
	Run:   runTransferVerifierSui,
}

// CLI parameters
func init() {
	suiRPC = TransferVerifierCmdSui.Flags().String("suiRPC", "", "Sui RPC url")
	suiProcessInitialEvents = TransferVerifierCmdSui.Flags().Bool("suiProcessInitialEvents", false, "Indicate whether the Sui transfer verifier should process the initial events it fetches")
	suiDigest = TransferVerifierCmdSui.Flags().String("suiDigest", "", "If provided, perform transaction verification on this single digest")
	suiEnvironment = TransferVerifierCmdSui.Flags().String("suiEnvironment", "mainnet", "The Sui environment to connect to. Supported values: mainnet, testnet and devnet")
	suiCoreBridgeStateObjectId = TransferVerifierCmdSui.Flags().String("suiCoreBridgeStateObjectId", "", "The Sui Core Bridge state object ID. If not provided, the default for the selected environment will be used.")
	suiTokenBridgeStateObjectId = TransferVerifierCmdSui.Flags().String("suiTokenBridgeStateObjectId", "", "The Sui Token Bridge state object ID. If not provided, the default for the selected environment will be used.")
}

func setIfEmpty(param *string, value string) {
	if *param == "" {
		*param = value
	}
}

// Analyse the commandline arguments and prepare the net effect of package and object IDs
func resolveSuiConfiguration() {

	// Only set the state object IDs from the static defaults if they weren't overridden
	switch *suiEnvironment {
	case "mainnet":
		setIfEmpty(suiCoreBridgeStateObjectId, "0xaeab97f96cf9877fee2883315d459552b2b921edc16d7ceac6eab944dd88919c")
		setIfEmpty(suiTokenBridgeStateObjectId, txverifier.SuiMainnetStateObjectId)
	case "testnet":
		setIfEmpty(suiCoreBridgeStateObjectId, "0x31358d198147da50db32eda2562951d53973a0c0ad5ed738e9b17d88b213d790")
		setIfEmpty(suiTokenBridgeStateObjectId, txverifier.SuiTestnetStateObjectId)
	case "devnet":
		setIfEmpty(suiCoreBridgeStateObjectId, "0x5a5160ca3c2037f4b4051344096ef7a48ebf4400b3f385e57ea90e1628a8bde0")
		setIfEmpty(suiTokenBridgeStateObjectId, txverifier.SuiDevnetStateObjectId)
	}

	// Create the Sui Api connection, and query the state object to get the token bridge address and emitter.
	suiApiConnection := txverifier.NewSuiApiConnection(*suiRPC)

	// Get core bridge parameters
	coreBridgeStateObject, err := suiApiConnection.GetObject(context.Background(), *suiCoreBridgeStateObjectId)

	if err != nil {
		panic(err)
	}

	objectType, err := coreBridgeStateObject.Type()
	objectTypeParts := strings.Split(objectType, "::")

	if err != nil || len(objectTypeParts) < 3 {
		panic("Error getting core bridge object type")
	}
	suiCoreContract = objectTypeParts[0]

	// Get token bridge parameters
	tokenBridgeStateObject, err := suiApiConnection.GetObject(context.Background(), *suiTokenBridgeStateObjectId)

	if err != nil {
		panic(err)
	}

	suiTokenBridgeEmitter, err = tokenBridgeStateObject.TokenBridgeEmitter()
	if err != nil {
		panic(err)
	}

	suiTokenBridgeAddress, err = tokenBridgeStateObject.TokenBridgePackageId()
	if err != nil {
		panic(err)
	}
}

func runTransferVerifierSui(cmd *cobra.Command, args []string) {
	resolveSuiConfiguration()

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

	// Verify CLI parameters
	if *suiRPC == "" || suiCoreContract == "" || suiTokenBridgeEmitter == "" || suiTokenBridgeAddress == "" {
		logger.Fatal("One or more CLI parameters are empty",
			zap.String("suiRPC", *suiRPC),
			zap.String("suiCoreContract", suiCoreContract),
			zap.String("suiTokenBridgeEmitter", suiTokenBridgeEmitter),
			zap.String("suiTokenBridgeContract", suiTokenBridgeAddress))
	}

	logger.Info("Starting Sui transfer verifier")
	logger.Debug("Sui rpc connection", zap.String("url", *suiRPC))
	logger.Debug("Sui core contract", zap.String("address", suiCoreContract))
	logger.Debug("Sui token bridge contract", zap.String("address", suiTokenBridgeAddress))
	logger.Debug("token bridge event emitter", zap.String("object id", suiTokenBridgeEmitter))
	logger.Debug("process initial events", zap.Bool("processInitialEvents", *suiProcessInitialEvents))

	suiApiConnection := txverifier.NewSuiApiConnection(*suiRPC)

	// Create a new SuiTransferVerifier
	suiTransferVerifier := txverifier.NewSuiTransferVerifier(suiCoreContract, suiTokenBridgeEmitter, suiTokenBridgeAddress, suiApiConnection)

	// Process a single digest and exit
	if *suiDigest != "" {
		logger.Info("Processing single digest", zap.String("txDigeset", *suiDigest))
		valid, err := suiTransferVerifier.ProcessDigest(ctx, *suiDigest, "", logger)

		if err != nil {
			logger.Error("Error validating the digest", zap.Error(err))
		}

		logger.Info("Validation completed", zap.Bool("valid", valid))

		return
	}

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
				_, err = suiTransferVerifier.ProcessDigest(ctx, *event.ID.TxDigest, "", logger)
				if err != nil {
					logger.Error(err.Error())
				}
			}
		}
		logger.Info("Finished processing initial events")
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
			resp, err := suiApiConnection.QueryEvents(ctx, eventFilter, "null", InitialEventFetchLimit, true)
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
				_, err := suiTransferVerifier.ProcessDigest(ctx, txDigest, "", logger)
				if err != nil {
					logger.Error(err.Error())
				}
				logger.Info("Processed new event", zap.String("txDigest", txDigest))
			}

			logger.Info("New events processed", zap.Int("latestTimestamp", latestTimestamp), zap.Int("txDigestCount", len(txDigests)))

		}
	}
}
