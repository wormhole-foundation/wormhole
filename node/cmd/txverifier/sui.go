package txverifier

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/telemetry"
	txverifier "github.com/certusone/wormhole/node/pkg/txverifier"
	"github.com/certusone/wormhole/node/pkg/version"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

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
	suiRPC                       *string
	suiProcessOnChainEvents      *bool
	suiProcessWormholeScanEvents *bool
	suiEnvironment               *string
	suiDigest                    *string

	// Sui package IDs and emitter addresses
	suiCoreBridgePackageId  *string
	suiTokenBridgeEmitter   *string
	suiTokenBridgePackageId *string
)

var TransferVerifierCmdSui = &cobra.Command{
	Use:   "sui",
	Short: "Transfer Verifier for Sui",
	Run:   runTransferVerifierSui,
}

// CLI parameters
func init() {
	suiRPC = TransferVerifierCmdSui.Flags().String("suiRPC", "", "Sui RPC url")
	suiProcessOnChainEvents = TransferVerifierCmdSui.Flags().Bool("suiProcessOnChainEvents", false, "Indicate whether the Sui transfer verifier should process on-chain events")
	suiProcessWormholeScanEvents = TransferVerifierCmdSui.Flags().Bool("suiProcessWormholeScanEvents", false, "Indicate whether the Sui transfer verifier should process WormholeScan events")
	suiDigest = TransferVerifierCmdSui.Flags().String("suiDigest", "", "If provided, perform transaction verification on this single digest")
	suiEnvironment = TransferVerifierCmdSui.Flags().String("suiEnvironment", "mainnet", "The Sui environment to connect to. Supported values: mainnet, testnet and devnet")

	suiCoreBridgePackageId = TransferVerifierCmdSui.Flags().String("suiCoreBridgePackageId", "", "The Sui Core Bridge package ID. If not provided, the default for the selected environment will be used.")
	suiTokenBridgeEmitter = TransferVerifierCmdSui.Flags().String("suiTokenBridgeEmitter", "", "The Sui Token Bridge emitter address. If not provided, the default for the selected environment will be used.")
	suiTokenBridgePackageId = TransferVerifierCmdSui.Flags().String("suiTokenBridgePackageId", "", "The Sui Token Bridge package ID. If not provided, the default for the selected environment will be used.")
}

func setIfEmpty(param *string, value string) {
	if *param == "" {
		*param = value
	}
}

// Analyse the commandline arguments and prepare the net effect of package and object IDs
func resolveSuiConfiguration() {

	// Set the package IDs and emitter address based on the environment, if they are not provided
	// as CLI args.
	switch *suiEnvironment {
	case "mainnet":
		setIfEmpty(suiCoreBridgePackageId, "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a")
		setIfEmpty(suiTokenBridgePackageId, txverifier.SuiOriginalTokenBridgePackageIds[common.MainNet])
		setIfEmpty(suiTokenBridgeEmitter, "0x"+hex.EncodeToString(sdk.KnownTokenbridgeEmitters[vaa.ChainIDSui]))
	case "testnet":
		setIfEmpty(suiCoreBridgePackageId, "0xf47329f4344f3bf0f8e436e2f7b485466cff300f12a166563995d3888c296a94")
		setIfEmpty(suiTokenBridgePackageId, txverifier.SuiOriginalTokenBridgePackageIds[common.TestNet])
		setIfEmpty(suiTokenBridgeEmitter, "0x"+hex.EncodeToString(sdk.KnownTestnetTokenbridgeEmitters[vaa.ChainIDSui]))
	case "devnet":
		setIfEmpty(suiCoreBridgePackageId, "0x320a40bff834b5ffa12d7f5cc2220dd733dd9e8e91c425800203d06fb2b1fee8")
		setIfEmpty(suiTokenBridgePackageId, txverifier.SuiOriginalTokenBridgePackageIds[common.UnsafeDevNet])
		setIfEmpty(suiTokenBridgeEmitter, "0x"+hex.EncodeToString(sdk.KnownDevnetTokenbridgeEmitters[vaa.ChainIDSui]))
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
	if *suiRPC == "" || *suiCoreBridgePackageId == "" || *suiTokenBridgeEmitter == "" || *suiTokenBridgePackageId == "" {
		logger.Fatal("One or more CLI parameters are empty",
			zap.String("suiRPC", *suiRPC),
			zap.String("suiCoreBridgePackageId", *suiCoreBridgePackageId),
			zap.String("suiTokenBridgeEmitter", *suiTokenBridgeEmitter),
			zap.String("suiTokenBridgePackageId", *suiTokenBridgePackageId))
	}

	logger.Info("Starting Sui transfer verifier")
	logger.Debug("Sui rpc connection", zap.String("url", *suiRPC))
	logger.Debug("Sui core bridge package ID", zap.String("packageId", *suiCoreBridgePackageId))
	logger.Debug("Sui token bridge package ID", zap.String("packageId", *suiTokenBridgePackageId))
	logger.Debug("Sui token bridge emitter", zap.String("address", *suiTokenBridgeEmitter))
	logger.Debug("process on-chain events", zap.Bool("processOnChainEvents", *suiProcessOnChainEvents))
	logger.Debug("process WormholeScan events", zap.Bool("processWormholeScanEvents", *suiProcessWormholeScanEvents))

	suiApiConnection := txverifier.NewSuiApiConnection(*suiRPC)

	// Create a new SuiTransferVerifier
	suiTransferVerifier := txverifier.NewSuiTransferVerifier(*suiCoreBridgePackageId, *suiTokenBridgeEmitter, *suiTokenBridgePackageId, suiApiConnection)

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

	if *suiProcessWormholeScanEvents {
		digests, err := pullDigestsFromWormholeScan(ctx, logger)
		if err != nil {
			logger.Fatal("Error pulling digests from WormholeScan", zap.Error(err))
		}
		// TODO: check the result of each digest against an expected outcome. Some digests
		// link to token attestations, which the transfer verifier doesn't handle.
		for _, digest := range digests {
			_, err := suiTransferVerifier.ProcessDigest(ctx, digest, "", logger)
			if err != nil {
				logger.Error(err.Error())
			}
		}
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
	if *suiProcessOnChainEvents {
		logger.Info("Processing on-chain events")
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

			if len(txDigests) > 0 {
				logger.Info("New events processed", zap.Int("latestTimestamp", latestTimestamp), zap.Int("txDigestCount", len(txDigests)))
			}

		}
	}
}

type WormholeScanResponse struct {
	Operation []struct {
		SourceChain struct {
			Transaction struct {
				TxHash string `json:"txHash"`
			} `json:"transaction"`
		} `json:"sourceChain"`
	} `json:"operations"`
}

// Pulls a bunch of transaction digests from Wormholescan to run through the transfer verifier.
// https://api.wormholescan.io/api/v1/operations?sourceChain=21&appId=PORTAL_TOKEN_BRIDGE
func pullDigestsFromWormholeScan(ctx context.Context, logger *zap.Logger) ([]string, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.wormholescan.io/api/v1/operations?sourceChain=21&appId=PORTAL_TOKEN_BRIDGE", nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, _ := common.SafeRead(resp.Body)

	var wsResp WormholeScanResponse
	err = json.Unmarshal(body, &wsResp)
	if err != nil {
		return nil, err
	}

	digests := make([]string, 0, len(wsResp.Operation))
	for _, operation := range wsResp.Operation {
		digests = append(digests, operation.SourceChain.Transaction.TxHash)
	}

	logger.Info("Pulled digests from WormholeScan", zap.Int("count", len(digests)))
	return digests, nil
}
