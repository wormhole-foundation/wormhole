package txverifier

import (
	"github.com/spf13/cobra"
)

var TransferVerifierCmd = &cobra.Command{
	Use:   "transfer-verifier",
	Short: "Transfer Verifier",
}

var (
	// logLevel is a global flag that is used to set the logging level for the TransferVerifierCmd
	logLevel *string
	// telemetryLokiUrl is a global flag that is used to set the Loki cloud logging URL for the TransferVerifierCmd.
	telemetryLokiUrl *string
	// telemetryNodeName is a global flag that is used to set the node name used in telemetry for the TransferVerifierCmd.
	telemetryNodeName *string
)

// init initializes the global flags and subcommands for the TransferVerifierCmd.
// It sets up a persistent flag for logging level with a default value of "info"
// and adds subcommands for EVM and Sui transfer verification.
func init() {
	// Global flags
	logLevel = TransferVerifierCmd.PersistentFlags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")
	telemetryLokiUrl = TransferVerifierCmd.PersistentFlags().String("telemetryLokiUrl", "", "Loki cloud logging URL")
	telemetryNodeName = TransferVerifierCmd.PersistentFlags().String("telemetryNodeName", "", "Node name used in telemetry")

	// Either both loki flags should be present or neither of them.
	TransferVerifierCmd.MarkFlagsRequiredTogether("telemetryLokiUrl", "telemetryNodeName")

	// Subcommands corresponding to chains supported by the Transfer Verifier.
	TransferVerifierCmd.AddCommand(TransferVerifierCmdEvm)
	TransferVerifierCmd.AddCommand(TransferVerifierCmdSui)
}
