package node

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func NewTestRootCommand() *cobra.Command {
	var ethRPC *string
	var solRPC *string

	// Define test configuration
	testConfig := ConfigOptions{
		FilePath:  "testdata",
		FileName:  "test",
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
			fmt.Fprintln(out, "solRPC:", *solRPC)
		},
	}

	ethRPC = rootCmd.Flags().String("ethRPC", "", "Ethereum RPC URL")
	solRPC = rootCmd.Flags().String("solRPC", "", "Solana RPC URL")

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
		wantOutput := "ethRPC: ws://eth-config-file:8545\nsolRPC: ws://sol-config-file:8545\n"

		assert.Equal(t, wantOutput, gotOutput, "expected ethRPC to use the config file default")
	})

	t.Run("env var", func(t *testing.T) {
		os.Setenv("TEST_GUARDIAN_ETHRPC", "ws://eth-env-var:8545")
		defer os.Unsetenv("TEST_GUARDIAN_ETHRPC")

		cmd := NewTestRootCommand()
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.Execute()

		gotOutput := output.String()
		wantOutput := "ethRPC: ws://eth-env-var:8545\nsolRPC: ws://sol-config-file:8545\n"

		assert.Equal(t, wantOutput, gotOutput, "expected ethRPC to use the environment variable and solRPC to use the config file default")
	})

	// Set ethRPC with a flag
	t.Run("flag", func(t *testing.T) {
		os.Setenv("TEST_GUARDIAN_ETHRPC", "ws://eth-env-var:8545")
		defer os.Unsetenv("TEST_GUARDIAN_ETHRPC")

		cmd := NewTestRootCommand()
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.SetArgs([]string{
			"--ethRPC",
			"ws://eth-flag:8545",
			"--solRPC",
			"ws://sol-flag:8545",
		})
		cmd.Execute()

		gotOutput := output.String()
		wantOutput := "ethRPC: ws://eth-flag:8545\nsolRPC: ws://sol-flag:8545\n"

		assert.Equal(t, wantOutput, gotOutput, "expected the ethRPC to use the flag value and solRPC to use the flag value")
	})
}
