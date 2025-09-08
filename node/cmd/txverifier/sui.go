package txverifier

import (
	"context"
	"encoding/hex"
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

	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
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
	suiMainnet              *bool
	suiDigest               *string
	suiExecuteHappyList     *bool
)

var TransferVerifierCmdSui = &cobra.Command{
	Use:   "sui",
	Short: "Transfer Verifier for Sui",
	Run:   runTransferVerifierSui,
}

// CLI parameters
func init() {
	suiRPC = TransferVerifierCmdSui.Flags().String("suiRPC", "", "Sui RPC url")
	suiCoreContract = TransferVerifierCmdSui.Flags().String("suiCoreContract", "", "Sui core contract address")
	suiTokenBridgeEmitter = TransferVerifierCmdSui.Flags().String("suiTokenBridgeEmitter", "", "Token bridge emitter on Sui")
	suiTokenBridgeContract = TransferVerifierCmdSui.Flags().String("suiTokenBridgeContract", "", "Token bridge contract on Sui")
	suiProcessInitialEvents = TransferVerifierCmdSui.Flags().Bool("suiProcessInitialEvents", false, "Indicate whether the Sui transfer verifier should process the initial events it fetches")
	suiMainnet = TransferVerifierCmdSui.Flags().Bool("suiMainnet", false, "This flag sets the Sui-related package and object IDs to mainnet defaults, but will be overridden by any additional arguments")
	suiDigest = TransferVerifierCmdSui.Flags().String("suiDigest", "", "If provided, perform transaction verification on this single digest")
	suiExecuteHappyList = TransferVerifierCmdSui.Flags().Bool("suiExecuteHappyList", false, "If provided, run the happy list of digests through the verifier and exit")
}

// Analyse the commandline arguments and prepare the net effect of package and object IDs
func resolveSuiConfiguration() {

	// if Mainnet is specified, set the empty configuration arguments to mainnet defaults
	if *suiMainnet {
		if *suiCoreContract == "" {
			*suiCoreContract = "0x" + sdk.KnownMainnetCoreContracts[vaa.ChainIDSui]
		}
		if *suiTokenBridgeContract == "" {
			*suiTokenBridgeContract = "0x" + sdk.KnownMainnetTokenBridgeContracts[vaa.ChainIDSui]
		}
		if *suiTokenBridgeEmitter == "" {
			*suiTokenBridgeEmitter = "0x" + hex.EncodeToString(sdk.KnownTokenbridgeEmitters[vaa.ChainIDSui])
		}
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
	if *suiRPC == "" || *suiCoreContract == "" || *suiTokenBridgeEmitter == "" || *suiTokenBridgeContract == "" {
		logger.Fatal("One or more CLI parameters are empty",
			zap.String("suiRPC", *suiRPC),
			zap.String("suiCoreContract", *suiCoreContract),
			zap.String("suiTokenBridgeEmitter", *suiTokenBridgeEmitter),
			zap.String("suiTokenBridgeContract", *suiTokenBridgeContract))
	}

	logger.Info("Starting Sui transfer verifier")
	logger.Debug("Sui rpc connection", zap.String("url", *suiRPC))
	logger.Debug("Sui core contract", zap.String("address", *suiCoreContract))
	logger.Debug("Sui token bridge contract", zap.String("address", *suiTokenBridgeContract))
	logger.Debug("token bridge event emitter", zap.String("object id", *suiTokenBridgeEmitter))
	logger.Debug("process initial events", zap.Bool("processInitialEvents", *suiProcessInitialEvents))

	suiApiConnection := txverifier.NewSuiApiConnection(*suiRPC)

	// Create a new SuiTransferVerifier
	suiTransferVerifier := txverifier.NewSuiTransferVerifier(*suiCoreContract, *suiTokenBridgeEmitter, *suiTokenBridgeContract, suiApiConnection)

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

	// Process the happy list of digests and exit
	if *suiExecuteHappyList {
		logger.Info("Processing happy list of digests")
		for _, txDigest := range happyList {
			valid, err := suiTransferVerifier.ProcessDigest(ctx, txDigest, "", logger)
			logger.Info("Processed digest from happy list", zap.String("txDigest", txDigest), zap.Bool("valid", valid), zap.Error(err))
		}
		logger.Info("Finished processing happy list of digests")
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

var happyList = []string{
	"7fTxK1F1EqZzaavr5Hz8LuYSjQSQvJb69nYo5pJBeRGy",
	"8p1hWau231AW5iWd6JiK36mt4qypBaVENKNLiqBZuBBj",
	"89M7fBZX31rgm5vPGwrHeRUbb1KqFSN7jiXVxmAXho9Q",
	"DUnWTgqBphH1US6De88F4p5B5tZeLP2J7ZFcQDG6VdVW",
	"EDVymS6bxJcmoES5R5px8ejdv5Tukq3P9ehTeAcutAQq",
	"368Hq6VktWZHRyWSqDSxFagDqkoi4ZhH9qwtMegC7nCB",
	"zx6iBhajnWVPjnm76cxA1bZ86GS1sXhS84ioy1Ds9Tt",
	"J4LFaMfB2qCHQPGqiYmki6rLEUyqPhYTj5pDcjJZSCpe",
	"EZ1hVx16UobNrybgJ7m5t47qb2ALNwHpdKsiTayoBJbK",
	"4TGB8jK1pgZG33CJoDbhwzeLRz9iEJiGHsUk1uLxF6rZ",
	"CbSiEKrcjbDCCQbz2CBJNT1BsDfx9Ykg8wfYb48rvCsT",
	"3sVEd4bDg9TbxN2qiscUMgdkrTYUaJG97SoEHu9unBJb",
	"9xumNMjpA5zEtTd71C2VaeznQ4g79FJhEyBcMeH7VqUx",
	"7T9bng9aDDVKRzQjp4L8bBks4aZzPj3QSDa7Jqf8yugz",
	"7ckV4P6dML5Dh6BW2cEQYXHiqbeqCX2CEzJfYHSYj2c6",
	"8TYRkSXEZFJzbyt7A74saBs2wt5Bf1Cq8yn8jjqhu7om",
	"FyCdoRe1LAY1jBy5EKPgEa33Erpu9TuUZFXcq6JiipG1",
	"2UM5xGZuRPz1mVNCiy8z6hxqhjo1nuDaaDCLsk7LnvWK",
	"FFFi1Ci7XzjzaV2StDccbvRgy2HEtgigWtzRgMDgfrW2",
	"3LShMH6R4em5thqL3Wa99FNNpjjT8C2ewwLrrkfvDrZG",
	"6bQ2HEFtXEbPkJ6p5RGVZA5RaZyyo5VBUrkxf6QDi8re",
	"56cDPLhrNGSLmn3n2ETbxecpg1UhU6wpzQP2BmuPQB3U",
	"5YnHQrRMzvU7dK6fxKL3C7akS9zZdgtMZiFZAjp7SMUg",
	"3U27U9cgyeoiiZgVyhppBN9UNoQdArTL6d5MZJo8126Y",
	"FHaZNqW1WZiosWSVwQHBNk385RHhdgUx5CEZ43Yspj6A",
	"DagnqqMwEHgFCchkEyQ9k54vUqnxfxRe1cYrxZn7nF2J",
	"89ZvDFFwG53EgeEGbTDaHYke88j8soK3sGPgapTusxvW",
	"6Koh9cuBSAjWmfbWymZGDFYa25bK5sMj7hsvPfzdjo8G",
	"9YVyLU8nRCZLkhzrDVvJ9QCF4MGcDFidiNQ22TjAHsCd",
	"D6v93VqGya6dxTvYimSxW6Xft1owzKy6wqUoh2c2B4HB",
	"8nRwuTQwEoL5nfs7UdQfPS1cTkpcynuJDFEnMFTG7cmL",
	"8dMGuvxZ1vRQdSQ8NJz19UYuwVXeWKDLsqjP2ENUK2nL",
	"7z1yxGjZSeGjnbRuTMCaS99Ht5pYn7ZMiiUXpKtUEAUh",
	"4Yu9wf7DdAwsQRTXB4M7VKnKuaW1CKErwNnL7nMatcpC",
	"EmkXb29jH6wNKeT81tqhcipZ5fjCCe5Zs2wAvbsRBtf7",
	"3hLiPHqs6CcuVE76ARQiUV2kQzdriJ4g3PPiVTMkYpZ3",
	"FJWrzpCAXeYy6bWFa8Vffgu4wWnUNMS2xSLQ7AFY8K22",
	"HwURG9SdcR6uFze6Jkiw5wE2JqQnG9YRRTHkLYUg45PC",
	"DR4ZjKJkwfnZ3o9hkTAncAj1FtkNSpEs1FG8Vw27nFAD",
	"9MY9BZnfsKhoWsEAcqavBDJXMon5n4vYbR3thC1J7LDs",
	"HJueR6bzQDF1zvQrupuN2pp1pyAtLF9VpNsPyinJKvuE",
	"Gr4YeUujWrgKxaDH6RC4S8pigxKuoMciDzSKuryAXiCZ",
	"BA8NHJSQpg8oz1weQt96BgYC6w3HbUpfgyNs5tv3niPj",
	"W8NoZXsAXkfCck7n5KBu4byj3CnLf5P91sz3E2MYfBS",
	"AX8YKn7erVLR9Uie7a5wXgouafvaRMgqXCH6cLpdjJas",
	"F3NKSeDKZ8pVN4nPUhF7WaYM5sVdN3Jujx39eeK6e5Sa",
	"EJvG3PEuYDzhTAc2rxdV4Lg9k1HcSTeMJP8KiFngVvq",
	"CcTGnoavNH55KYpMAYq55kyaWZvR7GpKBY2SKTS2FEFa",
	"34Epseb53i6XmkXJzY5SQRX6ZoT26Rw3oe2TtW6r1DCB",
	"2Eu6GDTEWDoPFjn5DTYHEeMn437AHF4dqchXzo2aEn5U",
}
