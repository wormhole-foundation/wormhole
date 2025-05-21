package cmd

import (
	"fmt"
	"os"

	"github.com/certusone/wormhole/node/cmd/ccq"
	"github.com/certusone/wormhole/node/cmd/debug"
	"github.com/certusone/wormhole/node/cmd/spy"
	txverifier "github.com/certusone/wormhole/node/cmd/txverifier"
	"github.com/certusone/wormhole/node/pkg/version"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"

	"github.com/certusone/wormhole/node/cmd/guardiand"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "guardiand",
	Short: "Wormhole guardian node",
}

// Top-level version subcommand
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display binary version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Version())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.guardiand.yaml)")
	rootCmd.AddCommand(guardiand.NodeCmd)
	rootCmd.AddCommand(spy.SpyCmd)
	rootCmd.AddCommand(txverifier.TransferVerifierCmd)
	rootCmd.AddCommand(ccq.QueryServerCmd)
	rootCmd.AddCommand(guardiand.KeygenCmd)
	rootCmd.AddCommand(guardiand.AdminCmd)
	rootCmd.AddCommand(guardiand.TemplateCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(debug.DebugCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".guardiand" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".guardiand.yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
