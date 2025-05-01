package txverifier

import (
	"context"
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
	// Receipt hash to analyze.
	hash *string
)

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
	hash = TransferVerifierCmdEvm.Flags().String("hash", "", "A receipt hash to evaluate. The tool will exit after processing the receipt.")

	TransferVerifierCmd.MarkFlagRequired("rpcUrl")
	TransferVerifierCmd.MarkFlagRequired("coreContract")
	TransferVerifierCmd.MarkFlagRequired("tokenContract")
	TransferVerifierCmd.MarkFlagRequired("wrappedNativeContract")
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

	// Single-shot mode: process a single receipt hash.
	if len(*hash) > 0 {
		receiptHash := common.HexToHash(*hash)
		result, err := transferVerifier.TransferIsValid(ctx, receiptHash, nil)
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
			valid, err := transferVerifier.TransferIsValid(ctx, vLog.Raw.TxHash, nil)
			if err != nil {
				logger.Debug("error when processing event",
					zap.Error(err),
					zap.Bool("result", valid),
					zap.String("txHash", vLog.Raw.TxHash.String()))
			}
			if !valid {
				logger.Error("token transfer is invalid",
					zap.String("txHash", vLog.Raw.TxHash.String()))
			}
			logger.Debug("done processing", zap.Bool("result", valid), zap.String("txHash", vLog.Raw.TxHash.String()))
		}
	}
}
