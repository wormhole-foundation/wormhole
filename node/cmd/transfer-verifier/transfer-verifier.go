package transferverifier

/*
	TODOs:
*/

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
)

// init initializes the global flags and subcommands for the TransferVerifierCmd.
// It sets up a persistent flag for logging level with a default value of "info"
// and adds subcommands for EVM and Sui transfer verification.
func init() {
	// Global flags
	logLevel = TransferVerifierCmd.PersistentFlags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")

	// Subcommands
	TransferVerifierCmd.AddCommand(TransferVerifierCmdEvm)
	TransferVerifierCmd.AddCommand(TransferVerifierCmdSui)
}
