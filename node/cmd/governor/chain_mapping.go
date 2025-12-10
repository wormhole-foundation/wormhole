package governor

import (
	"fmt"
	"log"
	"slices"

	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/governor/coingecko"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	chainMappingAPIKey string
	chainMappingOutput string
	onlyGovernedChains bool
)

var chainMappingCmd = &cobra.Command{
	Use:   "chain-mapping",
	Short: "Show mapping of Wormhole ChainIDs to CoinGecko platforms",
	Long: `Query CoinGecko to build and display the mapping between Wormhole ChainIDs
and CoinGecko platform identifiers.

This uses the chain_identifier field from CoinGecko's asset platforms to match
EVM chain IDs to their corresponding platform names.

Examples:
  # Show chain to platform mapping (free tier)
  guardiand governor chain-mapping

  # With API key
  guardiand governor chain-mapping --api-key YOUR_API_KEY

  # JSON output
  guardiand governor chain-mapping --output json
`,
	Run: runChainMapping,
}

func init() {
	chainMappingCmd.Flags().StringVar(&chainMappingAPIKey, "api-key", "", "CoinGecko API key (optional)")
	chainMappingCmd.Flags().StringVarP(&chainMappingOutput, "output", "o", "table", "Output format: table, json")
	chainMappingCmd.Flags().BoolVar(&onlyGovernedChains, "only-governed", false, "Only show chains governed by the Guardian")
}

func runChainMapping(cmd *cobra.Command, args []string) {
	// Create CoinGecko client
	client := coingecko.NewClient(chainMappingAPIKey, nil)

	// Build the mapping

	log.Printf("Building chain to platform mapping...")

	var chainIDs []vaa.ChainID
	if onlyGovernedChains {
		governedChains := governor.ChainList()
		governedChainIDs := make([]vaa.ChainID, 0, len(governedChains))
		for _, chain := range governedChains {
			governedChainIDs = append(governedChainIDs, chain.EmitterChainID)
		}
		chainIDs = governedChainIDs
	} else {
		chainIDs = vaa.GetAllNetworkIDs()
	}

	log.Printf("Fetching asset platforms...")
	missingPlatforms, err := client.BuildChainToPlatformMap(chainIDs)
	if err != nil {
		log.Fatalf("Failed to build chain mapping: %v", err.Error())
	}

	log.Printf("Mapping complete!")
	mapping := client.GetChainToPlatformMap()

	// Output results
	fmt.Printf("%-10s %-25s %-30s\n", "WORMHOLE_CHAIN_ID", "CHAIN_NAME", "COINGECKO_PLATFORM")
	fmt.Println("--------------------------------------------------------------------------------")
	for chainID := range slices.Values(chainIDs) {
		if platformName, ok := mapping[chainID]; ok {
			fmt.Printf("%-10d %-25s %-30s\n", chainID, chainID.String(), platformName)
		}

	}
	fmt.Printf("\nTotal mapped chains: (%d/%d)\n", len(mapping), len(chainIDs))
	if len(missingPlatforms) > 0 {
		log.Printf("Warning: The following chains were not mapped: %v", missingPlatforms)
	}

}
