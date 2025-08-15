package txverifier

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/certusone/wormhole/node/pkg/telemetry"
	txverifier "github.com/certusone/wormhole/node/pkg/txverifier"
	"github.com/certusone/wormhole/node/pkg/version"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/ethereum/go-ethereum/common"
	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var TransferVerifierCmdEvm = &cobra.Command{
	Use:   "evm",
	Short: "Transfer Verifier for EVM-based chains",
	Run:   runTransferVerifierEvm,
}

// Configuration variables for EVM interactions.
var (
	// RPC endpoint URL for interacting with an EVM node.
	evmRpc *string
	// Contract address of the core bridge.
	evmCoreContract *string
	// Contract address of the token bridge.
	evmTokenBridgeContract *string
	// Contract address of the wrapped native asset, e.g. WETH for Ethereum
	wrappedNativeContract *string
	// Height difference between pruning windows (in blocks).
	pruneHeightDelta *uint64
	// Transaction hash to analyze.
	hash *string
	// Perform sanity checks
	sanity *bool
)

var ErrInvariant = errors.New("invariant violation")

// sanityCheck represents a test case for the transfer verifier that can be replayed
// against mainnet to ensure that the tool has not regressed.
type sanityCheck struct {
	txHash   common.Hash
	msgValid bool
	err      error
}

// Function to initialize the configuration for the TransferVerifierCmdEvm flags.
//
//nolint:errcheck // The MarkFlagRequired calls will cause the script to fail on their own. No need to handle the errors manually.
func init() {
	evmRpc = TransferVerifierCmdEvm.Flags().String("rpcUrl", "ws://localhost:8546", "RPC url")
	evmCoreContract = TransferVerifierCmdEvm.Flags().String("coreContract", "", "core bridge address")
	evmTokenBridgeContract = TransferVerifierCmdEvm.Flags().String("tokenContract", "", "token bridge")
	wrappedNativeContract = TransferVerifierCmdEvm.Flags().String("wrappedNativeContract", "", "wrapped native address (e.g. WETH on Ethereum)")
	pruneHeightDelta = TransferVerifierCmdEvm.Flags().Uint64("pruneHeightDelta", 10, "The number of blocks for which to retain transaction receipts. Defaults to 10 blocks.")
	// Allows testing the tool on a single receipt.
	hash = TransferVerifierCmdEvm.Flags().String("hash", "", "A transaction hash to evaluate. The tool will exit after processing the receipt.")
	sanity = TransferVerifierCmdEvm.Flags().Bool("sanity", false, "Sanity check: evaluate a hard-coded set of receipts for testing. A fatal error is logged if the results don't match what was expected.")

	TransferVerifierCmd.MarkFlagRequired("rpcUrl")
	TransferVerifierCmd.MarkFlagRequired("coreContract")
	TransferVerifierCmd.MarkFlagRequired("tokenContract")

	// EVM-only configuration
	TransferVerifierCmdEvm.MarkFlagRequired("wrappedNativeContract")
	TransferVerifierCmdEvm.MarkFlagsMutuallyExclusive("hash", "sanity")
}

// Note: logger.Error should be reserved only for conditions that break the
// invariants of the Token Bridge
func runTransferVerifierEvm(cmd *cobra.Command, args []string) {

	// Setup logging
	lvl, logErr := ipfslog.LevelFromString(*logLevel)
	if logErr != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	logger := ipfslog.Logger("wormhole-transfer-verifier").Desugar()
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
			// Note: the product name parameter here is representing a per-chain configuration, so 'eth' is used
			// rather than 'evm'. This allows us to distinguish this instance from other EVM chains that may be added in
			// the future.
			"transfer-verifier-eth",
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

	logger.Info("Starting EVM transfer verifier")

	logger.Debug("EVM rpc connection", zap.String("url", *evmRpc))
	logger.Debug("EVM core contract", zap.String("address", *evmCoreContract))
	logger.Debug("EVM token bridge contract", zap.String("address", *evmTokenBridgeContract))
	logger.Debug("EVM wrapped native asset contract", zap.String("address", *wrappedNativeContract))
	logger.Debug("EVM prune config",
		zap.Uint64("height delta", *pruneHeightDelta))

	// Create the RPC connection, context, and channels
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	var evmConnector connectors.Connector
	evmConnector, connectErr := connectors.NewEthereumBaseConnector(ctx, "eth", *evmRpc, common.HexToAddress(*evmCoreContract), logger)
	if connectErr != nil {
		logger.Fatal("could not create new evm base connector",
			zap.Error(connectErr))
	}

	// Create main configuration for Transfer Verification
	transferVerifier, err := txverifier.NewTransferVerifier(
		ctx,
		evmConnector,
		&txverifier.TVAddresses{
			CoreBridgeAddr:    common.HexToAddress(*evmCoreContract),
			TokenBridgeAddr:   common.HexToAddress(*evmTokenBridgeContract),
			WrappedNativeAddr: common.HexToAddress(*wrappedNativeContract),
		},
		*pruneHeightDelta,
		logger,
	)

	if err != nil {
		logger.Fatal("could not create new transfer verifier", zap.Error(err))
	}

	// Do sanity checks, then quit.
	if *sanity {
		// Ensure each check gives the correct result and error type.
		// The (false, nil) case is not tested here because it requires a core bridge exploit; this can
		// be tested using the integration tests in Tilt.
		for i, check := range sanityChecks {
			logger.Info(fmt.Sprintf("Running sanity check %d for txHash %s", i, check.txHash))
			valid, err := transferVerifier.TransferIsValid(ctx, "", check.txHash, nil)
			logger.Debug("done processing", zap.Bool("result", valid), zap.String("txHash", check.txHash.String()))

			if err != nil {
				logger.Debug("could not validate",
					zap.Error(err),
					zap.Bool("result", valid),
					zap.String("txHash", check.txHash.String()))

				if !errors.Is(err, check.err) {
					logger.Fatal(fmt.Sprintf("Sanity check %d failed (wrong error) for txHash %s", i, check.txHash))
				}
			} else {
				// We got nil but we expected an error.
				if check.err != nil {
					logger.Fatal(fmt.Sprintf(
						"Sanity check had nil error for txHash %s, expected %s",
						check.txHash,
						check.err,
					))
				}
			}

			// Ensure that the right error type was returned
			if valid != check.msgValid {
				logger.Fatal(fmt.Sprintf("Sanity check %d failed (wrong result) for txHash %s", i, check.txHash))
			}
		}
		logger.Info("Sanity checks successful. Exiting.")
		os.Exit(0)
	}

	// Single-shot mode: process a single transaction hash, then quit.
	if len(*hash) > 0 {
		receiptHash := common.HexToHash(*hash)
		result, err := transferVerifier.TransferIsValid(ctx, "", receiptHash, nil)
		if err != nil {
			logger.Error("could not verify transfer", zap.Error(err))
			os.Exit(1)
		}
		logger.Info("result", zap.Bool("valid", result))
		os.Exit(0)
	}

	// Set-up for main processing loop

	// Subscription for LogMessagePublished events
	sub := txverifier.NewSubscription(evmConnector.Client(), evmConnector)
	sub.Subscribe(ctx)
	defer sub.Close()

	// MAIN LOOP:
	// - watch for LogMessagePublished events coming from the connector attached to the core bridge.
	// - process the events through the transfer verifier.
	for {
		select {
		case <-ctx.Done():
			logger.Info("context cancelled, exiting")
			return
		case subErr := <-sub.Errors():
			logger.Warn("error on subscription", zap.Error(subErr))

		// Process observed LogMessagePublished events
		case vLog := <-sub.Events():
			valid, err := transferVerifier.TransferIsValid(ctx, "", vLog.Raw.TxHash, nil)
			if err != nil {
				logger.Debug("could not validate",
					zap.Error(err),
					zap.Bool("result", valid),
					zap.String("txHash", vLog.Raw.TxHash.String()))
				continue
			}
			if !valid {
				// False + nil means that an invariant was violated.
				logger.Error(
					// WARNING: This error string is used by the integration tests for detecting
					// violations. If it is changed, the "error pattern" in the devnet files must
					// be updated to match.
					ErrInvariant.Error(),
					zap.String("txHash", vLog.Raw.TxHash.String()),
				)
			}
			logger.Debug("done processing", zap.Bool("result", valid), zap.String("txHash", vLog.Raw.TxHash.String()))
		}
	}
}

// A list of receipts that have revealed bugs during testing. These can be replayed while developing the
// package to ensure that there are no regressions introduced when processing live data.
var sanityChecks = []sanityCheck{
	// Message publication with wrapped asset
	{
		common.HexToHash(`0xa3e0bdf8896a0e1f1552eaa346a914d655a4f94a94739c4ffe86a941a47ec7a8`),
		true,
		nil,
	},

	// Message publication with a native deposit
	{
		common.HexToHash(`0x173a027bb960fa2e2e2275c66649264c1b961ffae0fbb4082efdf329a701979a`),
		true,
		nil,
	},

	// Many transfers, one event with no topics, and a LogMessagePublished event.
	// Unrelated to the Token Bridge.
	{
		common.HexToHash(`0x27acebf817c3c244adb47cd3867620d9a30691c0587c4f484878fa896068b4d5`),
		false,
		txverifier.ErrNoMsgsFromTokenBridge,
	},

	// Mayan Swift transfer. Should be successfully parsed and ultimately skipped.
	{
		common.HexToHash(`0xdfa07c6910e3650faa999986c4e85a0160eb7039f3697e4143a4a737e4036edd`),
		false,
		txverifier.ErrNoMsgsFromTokenBridge,
	},
	// Token Transfer with Payload of a wrapped asset.
	{
		common.HexToHash(`0xb6a993373786c962c864d57c77944b2c58056250e09fc6a15c87d473e5cfe206`),
		true,
		nil,
	},
	// An NFT transfer. Ensures that ERC721 transfers are not interpreted as ERC20 transfers that the program should analyze.
	{
		common.HexToHash(`0x5550571b9e7cee04db0e93b75cd6df655d356e3a9913c392a075d5e50dda1f2c`),
		false,
		txverifier.ErrNoMsgsFromTokenBridge,
	},
	// This is a deflationary token, and in this case the amount out of the token bridge is greater than
	// the amount transferred in. This occurs because the balanceOf() function implementation for this
	// token is a ratio of supply rather than a fixed number that always increases after a transfer.
	// As a result, this shows up as an invariant violation.
	{
		common.HexToHash(`0x3b592b8ecbfe2f1b650ebf08806d3309cab601794e2a1f0312c9ec230fca75bd`),
		false,
		nil,
	},
	// Receipt contains an unusual Deposit event with too much data. Ultimately it
	// has no message from the token bridge.
	{
		common.HexToHash(`0xdd372c2e4f3626f62ed0199ba84458dfe64fa594fd7bcea3a503b29c2ec2fa2c`),
		false,
		txverifier.ErrNoMsgsFromTokenBridge,
	},
}
