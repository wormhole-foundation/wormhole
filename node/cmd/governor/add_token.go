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
	// Parse and validate chain ID
	chainIDUint, err := strconv.ParseUint(addTokenChainID, 10, 16)
	if err != nil {
		log.Fatalf("Invalid chain ID '%s': must be a number", addTokenChainID)
	}
	chainID := vaa.ChainID(chainIDUint)

	// Validate it's a known Wormhole chain
	_, err = vaa.KnownChainIDFromNumber(chainIDUint)
	if err != nil {
		log.Fatalf("Unknown chain ID %d: %v", chainID, err)
	}

	// Normalize address for CoinGecko query (standard Ethereum format, lowercase)
	queryAddr := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(addTokenAddress), "0x"))
	if len(queryAddr) != 40 {
		log.Fatalf("Invalid token address: %s (must be 40 hex characters)", addTokenAddress)
	}

	// Create CoinGecko client
	client := cg.NewClient(addTokenAPIKey, nil)

	// Use static chain-to-platform mapping (no API call needed)
	client.UseStaticChainMapping()

	// Get platform for this chain
	platformID := client.GetPlatformForChain(chainID)
	if platformID == "" {
		log.Fatalf("Chain %s (%d) is not supported by CoinGecko.\nUse 'guardiand governor chain-mapping' to see supported chains.", chainID, chainID)
	}

	fmt.Printf("Using chain %s (%d) -> platform: %s\n", chainID, chainID, platformID)
	fmt.Printf("Querying CoinGecko for token: 0x%s\n\n", queryAddr)

	// Query CoinGecko for token information
	cgTokenInfo, err := client.GetTokenInfo(chainID, "0x"+queryAddr)
	if err != nil {
		log.Fatalf("Failed to query token info: %v", err)
	}

	// Convert to local tokenInfo struct for formatting
	tokenInfo := &tokenInfo{
		Symbol:      cgTokenInfo.Symbol,
		CoinGeckoID: cgTokenInfo.CoinGeckoID,
		Decimals:    cgTokenInfo.Decimals,
		Price:       cgTokenInfo.Price,
	}

	// Get the user-friendly CoinGecko URL
	tokenURL, err := client.GetTokenURL(chainID, addTokenAddress)
	if err != nil {
		// Non-fatal - just log a warning
		fmt.Printf("Warning: Could not generate CoinGecko URL: %v\n\n", err)
	}

	// Display token information
	fmt.Printf("Token found!\n")
	fmt.Printf("  Symbol:       %s\n", tokenInfo.Symbol)
	fmt.Printf("  CoinGecko ID: %s\n", tokenInfo.CoinGeckoID)
	fmt.Printf("  Decimals:     %d\n", tokenInfo.Decimals)
	fmt.Printf("  Price (USD):  $%.6f\n", tokenInfo.Price)
	if tokenURL != "" {
		fmt.Printf("  URL:          %s\n", tokenURL)
	}
	fmt.Println()

	// NOW normalize to Wormhole format (64 hex chars, zero-padded)
	wormholeAddr := normalizeAddress(queryAddr)
	if wormholeAddr == "" {
		log.Fatalf("Failed to normalize address to Wormhole format")
	}

	// Format the entry with optional URL comment
	entry := formatTokenEntry(chainID, wormholeAddr, tokenInfo, tokenURL)
	fmt.Printf("Generated entry:\n%s\n\n", entry)

	if addTokenDryRun {
		fmt.Println("Dry run mode - file not modified")
		return
	}

	// Add to manual_tokens.go
	err = addTokenToFile(entry)
	if err != nil {
		log.Fatalf("Failed to add token to file: %v", err)
	}

	fmt.Printf("✓ Token added to manual_tokens.go\n")
	fmt.Printf("  Chain: %d\n", chainID)
	fmt.Printf("  Address: %s\n", wormholeAddr)
	fmt.Printf("  Symbol: %s\n", tokenInfo.Symbol)
}

type tokenInfo struct {
	Symbol      string
	CoinGeckoID string
	Decimals    int
	Price       float64
}

// normalizeAddress converts a standard Ethereum address (40 hex chars) to Wormhole format
// Wormhole format: 64 hex characters (32 bytes), zero-padded on the left, lowercase, no 0x prefix
// Input: "a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48" (40 chars, already lowercase, no 0x)
// Output: "000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48" (64 chars)
func normalizeAddress(addr string) string {
	// Input should already be lowercase, no 0x, 40 characters
	// Just in case, strip prefix and trim
	addr = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(addr)), "0x")

	// Validate length
	if len(addr) > 64 {
		return "" // Invalid - too long
	}

	// Pad to 64 characters (32 bytes) with leading zeros
	if len(addr) < 64 {
		addr = strings.Repeat("0", 64-len(addr)) + addr
	}

	return addr
}

// formatTokenEntry formats a token entry in the style of manual_tokens.go
// If tokenURL is provided, it's added as a comment after the entry
func formatTokenEntry(chainID vaa.ChainID, addr string, info *tokenInfo, tokenURL string) string {
	// Format price with appropriate precision
	var priceStr string
	if info.Price >= 1000 {
		priceStr = fmt.Sprintf("%.0f", info.Price)
	} else if info.Price >= 1 {
		priceStr = fmt.Sprintf("%.2f", info.Price)
	} else if info.Price >= 0.01 {
		priceStr = fmt.Sprintf("%.4f", info.Price)
	} else {
		priceStr = fmt.Sprintf("%.6f", info.Price)
	}

	entry := fmt.Sprintf(
		`		{Chain: %d, Addr: "%s", Symbol: "%s", CoinGeckoId: "%s", Decimals: %d, Price: %s},`,
		chainID, addr, info.Symbol, info.CoinGeckoID, info.Decimals, priceStr,
	)

	// Add URL as a comment if available
	if tokenURL != "" {
		entry += fmt.Sprintf(" // %s", tokenURL)
	}

	return entry
}

// addTokenToFile adds the token entry to manual_tokens.go
func addTokenToFile(entry string) error {
	filePath := "node/pkg/governor/manual_tokens.go"

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Find the last entry before the closing brace
	lines := strings.Split(string(content), "\n")

	// Find the line with the closing brace of the slice
	insertIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "}" {
			insertIdx = i
			break
		}
	}

	if insertIdx == -1 {
		return fmt.Errorf("could not find insertion point in file")
	}

	// Insert the new entry before the closing brace
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertIdx]...)
	newLines = append(newLines, entry)
	newLines = append(newLines, lines[insertIdx:]...)

	// Write back to file
	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(filePath, []byte(newContent), 0600)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
