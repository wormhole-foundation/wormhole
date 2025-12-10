package governor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
)

// AssetPlatform represents a single asset platform from CoinGecko's API.
type AssetPlatform struct {
	ID              string `json:"id"`
	ChainIdentifier *int   `json:"chain_identifier"`
	Name            string `json:"name"`
	Shortname       string `json:"shortname"`
}

// TokenPrice represents the price information for a token.
type TokenPrice struct {
	ContractAddress string             // The contract address that was queried
	Prices          map[string]float64 // Map of currency to price (e.g., "usd" -> 0.9998)
}

// AssetPlatforms returns a list of all asset platforms supported by CoinGecko.
// The apiKey parameter is optional; pass empty string to use the free tier.
func AssetPlatforms(apiKey string) ([]AssetPlatform, error) {
	url := "https://api.coingecko.com/api/v3/asset_platforms"

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header if provided
	if apiKey != "" {
		req.Header.Add("x-cg-demo-api-key", apiKey)
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CoinGecko API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var platforms []AssetPlatform
	if err := json.NewDecoder(resp.Body).Decode(&platforms); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return platforms, nil
}

// SimpleTokenPrice queries the price of one or more tokens by contract address.
// This is a simpler/lighter endpoint than the full contract info endpoint.
//
// Parameters:
//   - platformID: The asset platform ID (e.g., "ethereum", "binance-smart-chain")
//   - contractAddresses: List of contract addresses to query (e.g., "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
//   - apiKey: CoinGecko API key (optional - pass empty string for free tier)
//
// (Currency is hard-coded to "usd")
//
// Returns a slice of TokenPrice, one for each contract address queried.
//
// Example:
//
//	prices, err := SimpleTokenPrice("ethereum",
//	    []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"},
//	    []string{"usd"},
//	    "")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("USDC price: $%.4f\n", prices[0].Prices["usd"])
func SimpleTokenPrice(platformID string, contractAddresses []string, apiKey string) ([]TokenPrice, error) {
	// Validate inputs
	if platformID == "" {
		return nil, fmt.Errorf("platformID is required")
	}
	if len(contractAddresses) == 0 {
		return nil, fmt.Errorf("at least one contract address is required")
	}

	// Build URL
	var baseURL string
	if apiKey == "" {
		baseURL = "https://api.coingecko.com/api/v3/simple/token_price/" + platformID
	} else {
		baseURL = "https://pro-api.coingecko.com/api/v3/simple/token_price/" + platformID
	}

	// Build query parameters
	params := url.Values{}
	params.Add("contract_addresses", strings.Join(contractAddresses, ","))
	params.Add("vs_currencies", "usd")
	if apiKey != "" {
		params.Add("x_cg_pro_api_key", apiKey)
	}

	// Construct full URL
	fullURL := baseURL + "?" + params.Encode()

	// Create HTTP request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := common.SafeRead(resp.Body)
		return nil, fmt.Errorf("CoinGecko API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	// Response format: { "0xcontractaddr": { "usd": 1.23, "eur": 1.10 }, ... }
	var rawResult map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to TokenPrice slice
	result := make([]TokenPrice, 0, len(rawResult))
	for contractAddr, prices := range rawResult {
		result = append(result, TokenPrice{
			ContractAddress: contractAddr,
			Prices:          prices,
		})
	}

	return result, nil
}
