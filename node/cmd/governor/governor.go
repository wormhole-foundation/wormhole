package governor

import "github.com/spf13/cobra"

var GovernorCmd = &cobra.Command{
	Use:   "governor",
	Short: "Governor utilities and CoinGecko queries",
}

func init() {
	GovernorCmd.AddCommand(assetPlatformsCmd)
	GovernorCmd.AddCommand(tokenPriceCmd)
	GovernorCmd.AddCommand(chainMappingCmd)
	GovernorCmd.AddCommand(addTokenCmd)
}
