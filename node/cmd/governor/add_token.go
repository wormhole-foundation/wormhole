package governor

import (
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
	addTokenAPIKey  string
	addTokenChainID string
	addTokenAddress string
	addTokenDryRun  bool
)

var addTokenCmd = &cobra.Command{
	Use:   "add-token --chain-id CHAIN_ID --address TOKEN_ADDRESS",
	Short: "Add a token to manual_tokens.go by querying CoinGecko",
	Long: `Query CoinGecko for token information and add it to manual_tokens.go.

This command:
1. Validates the Wormhole chain ID
2. Looks up the CoinGecko platform for the chain
3. Queries CoinGecko for token details (symbol, price, decimals, CoinGecko ID)
4. Formats the entry and adds it to manual_tokens.go

Examples:
  # Add USDC on Ethereum (chain 2)
  guardiand governor add-token --chain-id 2 --address 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

  # Add token on BSC (chain 4) with API key
  guardiand governor add-token --chain-id 4 --address 0x... --api-key YOUR_KEY

  # Dry run (print without modifying file)
  guardiand governor add-token --chain-id 2 --address 0x... --dry-run
`,
	Run: runAddToken,
}

func init() {
	addTokenCmd.Flags().StringVar(&addTokenAPIKey, "api-key", "", "CoinGecko API key (optional)")
	addTokenCmd.Flags().StringVarP(&addTokenChainID, "chain-id", "c", "", "Wormhole chain ID (required)")
	addTokenCmd.Flags().StringVarP(&addTokenAddress, "address", "a", "", "Token contract address (required)")
	addTokenCmd.Flags().BoolVar(&addTokenDryRun, "dry-run", false, "Print the entry without modifying the file")
	if err := addTokenCmd.MarkFlagRequired("chain-id"); err != nil {
		log.Fatalf("Failed to mark chain-id flag as required: %v", err)
	}
	if err := addTokenCmd.MarkFlagRequired("address"); err != nil {
		log.Fatalf("Failed to mark address flag as required: %v", err)
	}
}

func runAddToken(cmd *cobra.Command, args []string) {
	chainIDUint, err := strconv.ParseUint(addTokenChainID, 10, 16)
	if err != nil {
		log.Fatalf("Invalid chain ID '%s': must be a number", addTokenChainID)
	}
	chainID := vaa.ChainID(chainIDUint)

	_, err = vaa.KnownChainIDFromNumber(chainIDUint)
	if err != nil {
		log.Fatalf("Unknown chain ID %d: %v", chainID, err)
	}

	queryAddr := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(addTokenAddress), "0x"))
	if len(queryAddr) != 40 {
		log.Fatalf("Invalid token address: %s (must be 40 hex characters)", addTokenAddress)
	}

	client := cg.NewClient(addTokenAPIKey, nil)
	client.UseStaticChainMapping()

	platformID := client.GetPlatformForChain(chainID)
	if platformID == "" {
		log.Fatalf("Chain %s (%d) is not supported by CoinGecko.\nUse 'guardiand governor chain-mapping' to see supported chains.", chainID, chainID)
	}

	fmt.Printf("Using chain %s (%d) -> platform: %s\n", chainID, chainID, platformID)
	fmt.Printf("Querying CoinGecko for token: 0x%s\n\n", queryAddr)

	cgTokenInfo, err := client.GetTokenInfo(chainID, "0x"+queryAddr)
	if err != nil {
		log.Fatalf("Failed to query token info: %v", err)
	}

	tokenURL, err := client.GetTokenURL(chainID, addTokenAddress)
	if err != nil {
		fmt.Printf("Warning: Could not generate CoinGecko URL: %v\n\n", err)
	}

	fmt.Printf("Token found!\n")
	fmt.Printf("  Symbol:       %s\n", cgTokenInfo.Symbol)
	fmt.Printf("  CoinGecko ID: %s\n", cgTokenInfo.CoinGeckoID)
	fmt.Printf("  Decimals:     %d\n", cgTokenInfo.Decimals)
	fmt.Printf("  Price (USD):  $%.6f\n", cgTokenInfo.Price)
	if tokenURL != "" {
		fmt.Printf("  URL:          %s\n", tokenURL)
	}
	fmt.Println()

	wormholeAddr := padAddress(queryAddr)
	entry := formatTokenEntry(chainID, wormholeAddr, cgTokenInfo.Symbol, cgTokenInfo.CoinGeckoID, cgTokenInfo.Decimals, cgTokenInfo.Price, tokenURL)
	fmt.Printf("Generated entry:\n%s\n\n", entry)

	if addTokenDryRun {
		fmt.Println("Dry run mode - file not modified")
		return
	}

	err = addTokenToFile(entry)
	if err != nil {
		log.Fatalf("Failed to add token to file: %v", err)
	}

	fmt.Printf("✓ Token added to manual_tokens.go\n")
	fmt.Printf("  Chain: %d\n", chainID)
	fmt.Printf("  Address: %s\n", wormholeAddr)
	fmt.Printf("  Symbol: %s\n", cgTokenInfo.Symbol)
}

// padAddress converts an Ethereum address to Wormhole format (64 hex chars, zero-padded).
func padAddress(addr string) string {
	if len(addr) < 64 {
		return strings.Repeat("0", 64-len(addr)) + addr
	}
	return addr
}

func formatTokenEntry(chainID vaa.ChainID, addr, symbol, coinGeckoID string, decimals int, price float64, tokenURL string) string {
	var priceStr string
	switch {
	case price >= 1000:
		priceStr = fmt.Sprintf("%.0f", price)
	case price >= 1:
		priceStr = fmt.Sprintf("%.2f", price)
	case price >= 0.01:
		priceStr = fmt.Sprintf("%.4f", price)
	default:
		priceStr = fmt.Sprintf("%.6f", price)
	}

	entry := fmt.Sprintf(
		`		{Chain: %d, Addr: "%s", Symbol: "%s", CoinGeckoId: "%s", Decimals: %d, Price: %s},`,
		chainID, addr, symbol, coinGeckoID, decimals, priceStr,
	)

	if tokenURL != "" {
		entry += fmt.Sprintf(" // %s", tokenURL)
	}

	return entry
}

func addTokenToFile(entry string) error {
	filePath := "node/pkg/governor/manual_tokens.go"

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	insertIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "}" {
			insertIdx = i
			break
		}
	}

	if insertIdx == -1 {
		return fmt.Errorf("could not find insertion point in file")
	}

	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertIdx]...)
	newLines = append(newLines, entry)
	newLines = append(newLines, lines[insertIdx:]...)

	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(filePath, []byte(newContent), 0600)
}
