package debug

import "github.com/spf13/cobra"

var DebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debugging utilities",
}

func init() {
	DebugCmd.AddCommand(decodeVaaCmd)
}
