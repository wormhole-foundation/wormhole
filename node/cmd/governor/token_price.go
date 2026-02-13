package governor

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	cg "github.com/certusone/wormhole/node/pkg/governor/coingecko"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	tokenPriceAPIKey   string
	tokenPricePlatform string
	tokenPriceChainID  string
	tokenPriceOutput   string
)

var tokenPriceCmd = &cobra.Command{
	Use:   "token-price [CONTRACT_ADDRESSES...]",
	Short: "Query token prices by contract address",
	Long: `Query CoinGecko API to get token prices by contract address.

You can specify either --platform or --chain-id (but not both):
  --platform: CoinGecko platform ID (e.g., ethereum, binance-smart-chain)
  --chain-id: Wormhole chain ID (e.g., 1, 56, 137) - automatically mapped to platform
	
Examples:
  # Using Wormhole chain ID (recommended)
  guardiand governor token-price --chain-id 1 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48
  guardiand governor token-price --chain-id 56 0x... 0x...  # BSC

  # Using platform ID directly
  guardiand governor token-price --platform ethereum 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

  # With API key
  guardiand governor token-price --chain-id 1 --api-key YOUR_API_KEY 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

  # Output as JSON
  guardiand governor token-price --chain-id 1 --output json 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

Common chain IDs:
  1 (Ethereum), 56 (BSC), 137 (Polygon), 43114 (Avalanche), 42161 (Arbitrum), 10 (Optimism)
`,
	Args: cobra.MinimumNArgs(1),
	Run:  runTokenPrice,
}

func init() {
	tokenPriceCmd.Flags().StringVar(&tokenPriceAPIKey, "api-key", "", "CoinGecko API key (optional)")
	tokenPriceCmd.Flags().StringVarP(&tokenPricePlatform, "platform", "p", "", "CoinGecko platform ID (e.g., ethereum, binance-smart-chain)")
	tokenPriceCmd.Flags().StringVarP(&tokenPriceChainID, "chain-id", "c", "", "Wormhole chain ID (e.g., 1, 56, 137)")
	tokenPriceCmd.Flags().StringVarP(&tokenPriceOutput, "output", "o", "table", "Output format: table, json")
	tokenPriceCmd.MarkFlagsMutuallyExclusive("platform", "chain-id")
}

func runTokenPrice(cmd *cobra.Command, args []string) {
	// Validate that either platform or chain-id is provided (but not both)
	if tokenPricePlatform == "" && tokenPriceChainID == "" {
		log.Fatal("Error: Either --platform or --chain-id must be specified")
	}

	// Determine the platform ID to use
	var platformID string
	if tokenPriceChainID != "" {
		// Parse chain ID
		chainIDUint, err := strconv.ParseUint(tokenPriceChainID, 10, 16)
		if err != nil {
			log.Fatalf("Invalid chain ID '%s': must be a number (e.g., 1, 56, 137)", tokenPriceChainID)
		}

		chainID := vaa.ChainID(chainIDUint)

		// Validate that this is a known chain ID
		chainName, err := vaa.ChainIDFromString(chainID.String())
		if err != nil {
			log.Fatalf("Unknown chain ID %d: not a known Wormhole chain", chainID)
		}
		if chainName != chainID {
			log.Fatalf("Chain ID %d is not recognized", chainID)
		}

		// Create CoinGecko client and build mapping
		coinGecko := cg.NewClient(tokenPriceAPIKey, nil)
		_, err = coinGecko.BuildChainToPlatformMap(vaa.GetAllNetworkIDs())
		if err != nil {
			log.Fatalf("Failed to build chain-to-platform mapping: %v", err)
		}

		// Get platform ID for this chain
		platformID = coinGecko.GetPlatformForChain(chainID)
		if platformID == "" {
			log.Fatalf("Chain %s (%d) is not supported by CoinGecko.\nUse 'guardiand governor chain-mapping' to see supported chains.", chainID, chainID)
		}

		fmt.Printf("Using chain %s (%d) -> platform: %s\n", chainID, chainID, platformID)
	} else {
		// Use platform ID directly
		platformID = tokenPricePlatform
	}

	// Normalize contract addresses to lowercase
	contractAddresses := make([]string, len(args))
	for i, addr := range args {
		contractAddresses[i] = strings.ToLower(strings.TrimPrefix(addr, "0x"))
		if !strings.HasPrefix(contractAddresses[i], "0x") {
			contractAddresses[i] = "0x" + contractAddresses[i]
		}
	}

	// Create CoinGecko client (may already exist from chain mapping)
	coinGecko := cg.NewClient(tokenPriceAPIKey, nil)

	// Fetch prices
	prices, err := coinGecko.SimpleTokenPrice(platformID, contractAddresses)
	if err != nil {
		log.Fatalf("Failed to fetch token prices: %v", err)
	}

	if len(prices) == 0 {
		log.Fatalf("No prices returned. Please check that the platform and contract addresses are correct.")
	}

	// Output results
	switch tokenPriceOutput {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(prices); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
	case "table":
		fmt.Printf("%-45s %-15s\n", "CONTRACT ADDRESS", "PRICE (USD)")
		fmt.Println("------------------------------------------------------------")
		for _, p := range prices {
			price := "N/A"
			if usdPrice, ok := p.Prices["usd"]; ok {
				price = fmt.Sprintf("$%.6f", usdPrice)
			}
			fmt.Printf("%-45s %-15s\n", p.ContractAddress, price)
		}
	default:
		log.Fatalf("Unknown output format: %s (valid options: table, json)", tokenPriceOutput)
	}
}
