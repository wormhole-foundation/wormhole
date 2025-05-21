package node

import (
	"errors"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type ConfigOptions struct {
	FilePath  string
	FileName  string
	EnvPrefix string
}

// InitFileConfig initializes configuration according to the following precedence:
// 1. Command line flags
// 2. Environment variables
// 3. Config file
// 4. Cobra default values
func InitFileConfig(cmd *cobra.Command, options ConfigOptions) error {
	v := viper.New()

	v.SetConfigName(options.FileName)
	v.AddConfigPath(options.FilePath)

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
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
