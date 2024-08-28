package transferverifier

import (
	"context"
	"fmt"
	"os"

	txverifier "github.com/certusone/wormhole/node/pkg/transfer-verifier"
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
	// Contract address of the EVM core bridge contract.
	evmCoreContract *string
	// Contract address of the token bridge contract.
	evmTokenBridgeContract *string
	// Height difference between pruning windows (in blocks).
	pruneHeightDelta *uint64
)

// Function to initialize the configuration for the TransferVerifierCmdEvm flags.
func init() {
	// default URL connection for anvil
	evmRpc = TransferVerifierCmdEvm.Flags().String("ethRPC", "ws://localhost:8545", "Ethereum RPC url")
	evmCoreContract = TransferVerifierCmdEvm.Flags().String("ethContract", "", "Ethereum core bridge address for verifying VAAs (required if ethRPC is specified)")
	evmTokenBridgeContract = TransferVerifierCmdEvm.Flags().String("tokenContract", "", "token bridge contract deployed on Ethereum")

	pruneHeightDelta = TransferVerifierCmdEvm.Flags().Uint64("pruneHeightDelta", 10, "The number of blocks for which to retain transaction receipts. Defaults to 10 blocks.")
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
	logger.Info("Starting EVM transfer verifier")

	// Verify CLI parameters
	if *evmRpc == "" || *evmCoreContract == "" || *evmTokenBridgeContract == "" {
		logger.Fatal(
			"One or more CLI parameters are empty",
			zap.String("rpc", *evmRpc),
			zap.String("coreContract", *evmCoreContract),
			zap.String("tokenContract", *evmTokenBridgeContract),
		)
	}

	logger.Debug("EVM rpc connection", zap.String("url", *evmRpc))
	logger.Debug("EVM core contract", zap.String("address", *evmCoreContract))
	logger.Debug("EVM token bridge contract", zap.String("address", *evmTokenBridgeContract))
	logger.Debug("EVM prune config",
		zap.Uint64("height delta", *pruneHeightDelta))

	// Create the RPC connection, context, and channels
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	var ethConnector connectors.Connector
	ethConnector, connectErr := connectors.NewEthereumBaseConnector(ctx, "eth", *evmRpc, common.HexToAddress(*evmCoreContract), logger)
	if connectErr != nil {
		logger.Fatal("could not create new ethereum base connector",
			zap.Error(connectErr))
	}

	// Create main configuration for Transfer Verification
	transferVerifier := txverifier.NewTransferVerifier(
		ethConnector,
		&txverifier.TVAddresses{
			CoreBridgeAddr:  common.HexToAddress(*evmCoreContract),
			TokenBridgeAddr: common.HexToAddress(*evmTokenBridgeContract),
			// TODO: should be a CLI parameter so that we could support other EVM chains
			WrappedNativeAddr: txverifier.WETH_ADDRESS,
		},
		*pruneHeightDelta,
		logger,
	)

	// Set-up for main processing loop

	// Subscription for LogMessagePublished events
	sub := txverifier.NewSubscription(ethConnector.Client(), ethConnector)
	sub.Subscribe(ctx)
	defer sub.Close()

	// MAIN LOOP:
	// - watch for LogMessagePublished events coming from the connector attached to the core bridge.
	// - process the events through the transfer verifier.
	for {
		select {
		case subErr := <-sub.Errors():
			logger.Warn("error on subscription", zap.Error(subErr))

		// Process observed LogMessagePublished events
		case vLog := <-sub.Events():
			transferVerifier.ProcessEvent(ctx, vLog)
		}
	}
}
