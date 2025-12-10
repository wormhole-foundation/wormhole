package governor

import (
	"fmt"
	"log"
	"sort"

	"github.com/certusone/wormhole/node/pkg/governor/coingecko"
	"github.com/spf13/cobra"
)

var (
	assetPlatformsAPIKey string
)

var assetPlatformsCmd = &cobra.Command{
	Use:   "asset-platforms",
	Short: "List all CoinGecko asset platforms",
	Long: `Query CoinGecko API to list all supported asset platforms.
	
Examples:
  # List all platforms (free tier)
  guardiand governor asset-platforms

  # List all platforms with API key
  guardiand governor asset-platforms --api-key YOUR_API_KEY
`,
	Run: runAssetPlatforms,
}

func init() {
	assetPlatformsCmd.Flags().StringVar(&assetPlatformsAPIKey, "api-key", "", "CoinGecko API key (optional)")
}

func runAssetPlatforms(cmd *cobra.Command, args []string) {
	// Create CoinGecko client
	client := coingecko.NewClient(assetPlatformsAPIKey, nil)

	// Fetch platforms
	err := client.BuildPlatformCache()
	if err != nil {
		log.Fatalf("Failed to fetch asset platforms: %v", err)
	}
	platforms := client.GetPlatforms()
	sort.Slice(platforms, func(i, j int) bool {
		return platforms[i].ID < platforms[j].ID
	})
	for _, platform := range platforms {
		fmt.Printf("%s\n", platform)
	}
}
