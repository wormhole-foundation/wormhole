package node

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type ConfigOptions struct {
	FilePath  string
	FileName  string
	EnvPrefix string
}

func InitFileConfig(cmd *cobra.Command, options ConfigOptions) error {
	v := viper.New()

	v.SetConfigName(options.FileName)
	// Look for config file in home directory
	v.AddConfigPath(options.FilePath)

	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// TODO: Not sure if we need to support env vars but leaving it here for now
	// Bind flags to environment variables with a common prefix to avoid conflicts
	// Example: --ethRPC will be bound to GUARDIAN_ETHRPC
	v.SetEnvPrefix(options.EnvPrefix)

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
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
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
