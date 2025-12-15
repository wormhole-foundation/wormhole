package governor

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	ipfslog "github.com/ipfs/go-log/v2"
	cg "github.com/certusone/wormhole/node/pkg/governor/coingecko"
	"github.com/mr-tron/base58"
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
	chainID, err := vaa.StringToKnownChainID(addTokenChainID)
	if err != nil {
		log.Fatalf("Invalid chain ID '%s': must be a known chain ID", addTokenChainID)
	}

	queryAddr := strings.TrimSpace(addTokenAddress)


	// Setup logging
	lvl, logErr := ipfslog.LevelFromString("WARN")
	if logErr != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	logger := ipfslog.Logger("governor-add-token").Desugar()
	ipfslog.SetAllLoggers(lvl)

	client := cg.NewClient(addTokenAPIKey, logger)
	client.UseStaticChainMapping()

	fmt.Printf("Querying CoinGecko for token: %s\n\n", queryAddr)

	cgTokenInfo, err := client.GetTokenInfo(chainID, queryAddr)
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

	wormholeAddr := padEVMAddress(queryAddr)
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

// padEVMAddress converts an EVM address to Wormhole format (64 hex chars, zero-padded, no 0x prefix).
// Returns the original address if the argument is not 0x-prefixed.
func padEVMAddress(addr string) string {
	if !strings.HasPrefix(addr, "0x") {
		return addr
	}

	addrNoPrefix := addr[2:]

	if len(addrNoPrefix) < 64 {
		return strings.Repeat("0", 64-len(addrNoPrefix)) + addrNoPrefix
	}
	return addr
}

// normalizeAddress converts an address from any chain into the Wormhole format (64 hex chars, zero-padded).
// Supports EVM addresses (0x-prefixed), Solana addresses (base58-encoded), and Move addresses (hex-encoded).
func normalizeAddress(addrRaw string) string {

	addr := addrRaw
	if strings.HasPrefix(addr, "0x") {
		addr = strings.TrimPrefix(addr, "0x")
	}

	if len(addr) == 64 {
		// A Move address is hex-encoded and already 64 chars long.
		return addr
	}

	// Two possible cases:
	// 1. The address is hex-encoded EVM address
	// 2. The address is a base58 encoded Solana address

	// EVM address: check if it's a valid hex-encoded address
	var addrBuf []byte
	count, err := hex.Decode(addrBuf[:], []byte(addr))
	if err == nil && count == 20 {
		// EVM address
		return padEVMAddress(addr)
	}

	// Solana address: decode base58 encoded address
	solAddrBytes, err := base58.Decode(addr)
	if err != nil {
		log.Fatalf("Failed to decode address '%s': %v", addr, err)
	}
	return hex.EncodeToString(solAddrBytes)

}

func formatTokenEntry(chainID vaa.ChainID, addrRaw, symbol, coinGeckoID string, decimals int, price float64, tokenURL string) string {
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
		chainID, addrRaw, symbol, coinGeckoID, decimals, priceStr,
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
