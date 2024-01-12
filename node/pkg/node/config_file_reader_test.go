package node

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/spf13/cobra"
)



func NewTestRootCommand() *cobra.Command {
	var ethRPC *string

	// Define test configuration
	testConfig := ConfigOptions{
		FilePath: "testdata",
		FileName: "test",
		EnvPrefix: "TEST_GUARDIAN",
	}

	rootCmd := &cobra.Command{
		Use:   "config_file_reader_test",
		Short: "Unit test to test config file reader",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize configuration using Viper
			return InitFileConfig(cmd, testConfig) // Adjust the filename as needed
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Working with OutOrStdout/OutOrStderr allows us to unit test our command easier
			out := cmd.OutOrStdout()

			// Print the final resolved value from binding cobra flags and viper config
			fmt.Fprintln(out, "ethRPC:", *ethRPC)
		},
	}

	ethRPC = rootCmd.Flags().String("ethRPC", "", "Ethereum RPC URL")

	return rootCmd
}

func TestInitFileConfig(t *testing.T) {
	// Set ethRPC with config file
	t.Run("config file", func(t *testing.T) {
		cmd := NewTestRootCommand()
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.Execute()

		gotOutput := output.String()
		wantOutput := "ethRPC: ws://eth-devnet:8545\n"

		assert.Equal(t, wantOutput, gotOutput, "expected ethRPC from the config file default")
	})

	// Set ethRPC with a flag
	t.Run("flag", func(t *testing.T) {
		cmd := NewTestRootCommand()
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.SetArgs([]string{"--ethRPC", "ws://eth-devnet2:8545"})
		cmd.Execute()

		gotOutput := output.String()
		wantOutput := "ethRPC: ws://eth-devnet2:8545\n"

		assert.Equal(t, wantOutput, gotOutput, "expected the ethRPC to use the flag value")
	})
}
