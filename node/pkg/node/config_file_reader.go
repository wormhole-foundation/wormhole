package node

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ConfigOptions is used to configure the loading of config parameters by "guardiand node".
type ConfigOptions struct {
	// FilePath is the path to the config file to be loaded, including the file name and extension.
	// If this is specified (either as fully qualified or relative), config parameters will be loaded
	// from that file, in addition to from environment variables and command line arguments.
	// The file may be any of the types supported by Viper (such as .yaml or .json).
	FilePath string

	// EnvPrefix is the prefix to be added to environment variables to load variables that
	// override config file settings. For instance, setting it to "GUARDIAND" will cause it
	// to look for variables like "GUARDIAND_ETHRPC".
	EnvPrefix string
}

// InitFileConfig initializes configuration according to the following precedence:
// 1. Command line flags
// 2. Environment variables
// 3. Config file
// 4. Cobra default values
func InitFileConfig(cmd *cobra.Command, options ConfigOptions) error {
	v := viper.New()

	if options.FilePath != "" {
		v.SetConfigFile(options.FilePath)
		if err := v.ReadInConfig(); err != nil {
			return err
		}
	}

	// Bind flags to environment variables with a common prefix to avoid conflicts
	// Example: --ethRPC will be bound to GUARDIAND_ETHRPC
	v.SetEnvPrefix(options.EnvPrefix)

	// Bind to environment variables
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Determine the naming convention of the flags when represented in the config file
		configName := f.Name

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if err != nil {
				log.Fatalf("failed to bind flag %s to viper: %v", f.Name, err)
			}
		}
	})
}
