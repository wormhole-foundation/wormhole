package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

const (
	// Base URLs for CoinGecko API
	freeAPIBaseURL = "https://api.coingecko.com/api/v3"
	proAPIBaseURL  = "https://pro-api.coingecko.com/api/v3"
)

// Client provides methods for interacting with the CoinGecko API.
type Client struct {
	apiKey string
	logger *zap.Logger
	client *http.Client

	// chainToPlatform stores the mapping of vaa.ChainID to CoinGecko platform ID
	chainToPlatform map[vaa.ChainID]string

	// platformCache stores the set of AssetPlatforms
	platformCache    map[AssetPlatform]struct{}
	platformCacheMux sync.RWMutex
}

// NewClient creates a new CoinGecko API client.
// The apiKey parameter is optional; pass empty string to use the free tier.
// The logger parameter is optional; pass nil if logging is not needed.
func NewClient(apiKey string, logger *zap.Logger) *Client {
	return &Client{
		apiKey:          apiKey,
		logger:          logger,
		client:          &http.Client{},
		chainToPlatform: make(map[vaa.ChainID]string),
		platformCache:   make(map[AssetPlatform]struct{}),
	}
}

// AssetPlatform represents a single asset platform from CoinGecko's API.
// Example response:
// [
//
//	{
//	  "id": "polygon-pos",
//	  "chain_identifier": 137,
//	  "name": "Polygon POS",
//	  "shortname": "MATIC",
//	  "native_coin_id": "matic-network",
//	  "image": {
//	    "thumb": "https://coin-images.coingecko.com/asset_platforms/images/15/thumb/polygon_pos.png?1706606645",
//	    "small": "https://coin-images.coingecko.com/asset_platforms/images/15/small/polygon_pos.png?1706606645",
//	    "large": "https://coin-images.coingecko.com/asset_platforms/images/15/large/polygon_pos.png?1706606645"
//	  }
//	}
//
// ]
type AssetPlatform struct {
	// e.g. "polygon-pos"
	ID string `json:"id"`
	// e.g. "Polygon POS"
	Name string `json:"name"`
	// e.g. "MATIC"
	Shortname string `json:"shortname"`
}

func (a *AssetPlatform) String() string {
	return fmt.Sprintf("AssetPlatform{ID: %s, Name: %s, Shortname: %s}",
		a.ID, a.Name, a.Shortname)

}

// TokenPrice represents the price information for a token.
type TokenPrice struct {
	ContractAddress string             // The contract address that was queried
	Prices          map[string]float64 // Map of currency to price (e.g., "usd" -> 0.9998)
}

// AssetPlatforms returns a list of all asset platforms supported by CoinGecko.
func (c *Client) BuildPlatformCache() error {

	// Only build the cache if it hasn't been built yet.
	if len(c.platformCache) > 0 {
		return nil
	}

	url := freeAPIBaseURL + "/asset_platforms"

	// Create HTTP request
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header if provided
	if c.apiKey != "" {
		req.Header.Add("x-cg-demo-api-key", c.apiKey)
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to execute AssetPlatforms request", zap.Error(err))
		}
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := common.SafeRead(resp.Body)
		if c.logger != nil {
			c.logger.Error("CoinGecko API returned error",
				zap.Int("status_code", resp.StatusCode),
				zap.String("response", string(body)))
		}
		return fmt.Errorf("CoinGecko API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var platforms []AssetPlatform
	if err := json.NewDecoder(resp.Body).Decode(&platforms); err != nil {
		if c.logger != nil {
			c.logger.Error("failed to decode AssetPlatforms response", zap.Error(err))
		}
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if c.logger != nil {
		c.logger.Debug("successfully fetched asset platforms", zap.Int("count", len(platforms)))
	}

	// Store the platforms in the set
	c.platformCacheMux.Lock()
	defer c.platformCacheMux.Unlock()
	for _, platform := range platforms {
		c.platformCache[platform] = struct{}{}
	}

	return nil
}

// SimpleTokenPrice queries the price of one or more tokens by contract address.
// This is a simpler/lighter endpoint than the full contract info endpoint.
//
// Parameters:
//   - platformID: The asset platform ID (e.g., "ethereum", "binance-smart-chain")
//   - contractAddresses: List of contract addresses to query (e.g., "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
//
// (Currency is hard-coded to "usd")
//
// Returns a slice of TokenPrice, one for each contract address queried.
//
// Example:
//
//	client := coingecko.NewClient("your-api-key", logger)
//	prices, err := client.SimpleTokenPrice("ethereum",
//	    []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("USDC price: $%.4f\n", prices[0].Prices["usd"])
func (c *Client) SimpleTokenPrice(platformID string, contractAddresses []string) ([]TokenPrice, error) {
	// Validate inputs
	if platformID == "" {
		return nil, fmt.Errorf("platformID is required")
	}
	if len(contractAddresses) == 0 {
		return nil, fmt.Errorf("at least one contract address is required")
	}

	// Build URL
	var baseURL string
	if c.apiKey == "" {
		baseURL = freeAPIBaseURL + "/simple/token_price/" + platformID
	} else {
		baseURL = proAPIBaseURL + "/simple/token_price/" + platformID
	}

	// Build query parameters
	params := url.Values{}
	params.Add("contract_addresses", strings.Join(contractAddresses, ","))
	params.Add("vs_currencies", "usd")
	if c.apiKey != "" {
		params.Add("x_cg_pro_api_key", c.apiKey)
	}

	// Construct full URL
	fullURL := baseURL + "?" + params.Encode()

	// Create HTTP request
	req, err := http.NewRequestWithContext(context.Background(), "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to execute SimpleTokenPrice request",
				zap.String("platform", platformID),
				zap.Int("num_contracts", len(contractAddresses)),
				zap.Error(err))
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := common.SafeRead(resp.Body)
		if c.logger != nil {
			c.logger.Error("CoinGecko API returned error",
				zap.String("platform", platformID),
				zap.Int("status_code", resp.StatusCode),
				zap.String("response", string(body)))
		}
		return nil, fmt.Errorf("CoinGecko API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	// Response format: { "0xcontractaddr": { "usd": 1.23, "eur": 1.10 }, ... }
	var rawResult map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		if c.logger != nil {
			c.logger.Error("failed to decode SimpleTokenPrice response", zap.Error(err))
		}
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

	if c.logger != nil {
		c.logger.Debug("successfully fetched token prices",
			zap.String("platform", platformID),
			zap.Int("num_prices", len(result)))
	}

	return result, nil
}

// BuildChainToPlatformMap returns a mapping from vaa.ChainID to CoinGecko platform ID.
//
// Returns a slice of vaa.ChainID that did not match a platform.
func (c *Client) BuildChainToPlatformMap(
	// The chainIDs that will be matched against the asset platforms.
	chainIDs []vaa.ChainID,
) (missingPlatforms []vaa.ChainID, err error) {

	if len(chainIDs) == 0 {
		return missingPlatforms, fmt.Errorf("empty chainIDs argument")
	}

	// Build the mapping by querying CoinGecko API
	err = c.BuildPlatformCache()
	if err != nil {
		return missingPlatforms, fmt.Errorf("failed to fetch asset platforms: %w", err)
	}

	platforms := c.GetPlatforms()
	for _, chainID := range chainIDs {
		found := false
		chainName := chainID.String()

		// Return early for platforms whose chain ID string does not map
		// predictably to a CoinGecko platform ID.
		// https://www.coingecko.com/en/chains

		switch chainID {
		case vaa.ChainIDCosmoshub:
			c.chainToPlatform[chainID] = "cosmos"
			found = true
			continue
		case vaa.ChainIDKlaytn:
			// Klaytn is now called "Kaia" on CoinGecko
			c.chainToPlatform[chainID] = "klay-token"
			found = true
			continue
		case vaa.ChainIDXLayer:
			c.chainToPlatform[chainID] = "x-layer"
			found = true
			continue
		case vaa.ChainIDSeiEVM:
			// SeiEVM uses the new Sei v2 platform
			c.chainToPlatform[chainID] = "sei-v2"
			found = true
			continue
		case vaa.ChainIDWorldchain:
			c.chainToPlatform[chainID] = "world-chain"
			found = true
			continue
		case vaa.ChainIDXRPLEVM:
			c.chainToPlatform[chainID] = "xrpl-evm"
			found = true
			continue
		case vaa.ChainIDWormchain:
			// Wormchain doesn't have a CoinGecko platform (non-EVM Cosmos chain)
			// Leave found=false so it appears in missingPlatforms
			continue
		case vaa.ChainIDFogo:
			// Fogo not yet listed on CoinGecko (as of Dec 2025)
			// Leave found=false so it appears in missingPlatforms
			continue
		case vaa.ChainIDTON:
			c.chainToPlatform[chainID] = "the-open-network"
			found = true
			continue
		}

		for _, platform := range platforms {
			// Skip empty platform IDs
			if platform.ID == "" {
				continue
			}
			// Try multiple matching strategies:
			// 1. Exact match (case-insensitive) with platform ID, name, or shortname
			if strings.EqualFold(platform.ID, chainName) ||
				strings.EqualFold(platform.Name, chainName) ||
				strings.EqualFold(platform.Shortname, chainName) {
				c.chainToPlatform[chainID] = platform.ID
				found = true
				break
			}

			// 2. Check if chain name is a prefix of platform ID
			//    e.g., "polygon" matches "polygon-pos"
			if strings.HasPrefix(strings.ToLower(platform.ID), strings.ToLower(chainName)) {
				c.chainToPlatform[chainID] = platform.ID
				found = true
				break
			}

			// Other substring matches tend to be incorrect, e.g. searching
			// "ink" matches "etherlink", etc.
		}

		if !found {
			missingPlatforms = append(missingPlatforms, chainID)
		}
	}

	// if c.logger != nil {
	// 	c.logger.Info("built chain to platform mapping",
	// 		zap.Int("total_platforms", len(platforms)),
	// 		zap.Int("mapped_chains", mappedCount))
	// }

	return
}

// GetPlatformForChain returns the CoinGecko platform ID for a given vaa.ChainID.
// Returns an empty string if the chain is not found in the cache.
//
// Note: You must call BuildChainToPlatformMap() first to populate the cache.
func (c *Client) GetPlatformForChain(chainID vaa.ChainID) string {
	c.platformCacheMux.RLock()
	defer c.platformCacheMux.RUnlock()

	return c.chainToPlatform[chainID]
}

// GetChainToPlatformMap returns a copy of the entire chain-to-platform mapping.
// This is useful for inspection and debugging.
//
// Note: You must call BuildChainToPlatformMap() first to populate the cache.
func (c *Client) GetChainToPlatformMap() map[vaa.ChainID]string {
	c.platformCacheMux.RLock()
	defer c.platformCacheMux.RUnlock()

	// Return a copy to prevent external modifications
	result := make(map[vaa.ChainID]string, len(c.chainToPlatform))
	maps.Copy(result, c.chainToPlatform)

	return result
}

// GetPlatforms returns a sorted slice of all asset platforms supported by CoinGecko.
func (c *Client) GetPlatforms() []AssetPlatform {
	c.platformCacheMux.RLock()
	defer c.platformCacheMux.RUnlock()

	result := make([]AssetPlatform, 0, len(c.platformCache))
	for platform := range c.platformCache {
		result = append(result, platform)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}

// GetAPIKey returns the API key used by the client (empty string if using free tier)
func (c *Client) GetAPIKey() string {
	return c.apiKey
}

// GetHTTPClient returns the underlying HTTP client
func (c *Client) GetHTTPClient() *http.Client {
	return c.client
}

// UseStaticChainMapping loads the hard-coded chain-to-platform mapping into the client.
// This avoids the need to query the CoinGecko API for platform information.
//
// This is useful for:
// - Offline usage or testing
// - Avoiding API rate limits
// - Faster initialization
//
// The static mapping is maintained in chain_mapping.go and should be periodically
// updated by running `guardiand governor chain-mapping`.
//
// Example:
//
//	client := coingecko.NewClient("", logger)
//	client.UseStaticChainMapping()
//	platform := client.GetPlatformForChain(vaa.ChainIDEthereum)
//	// Returns: "ethereum" (no API call needed)
func (c *Client) UseStaticChainMapping() {
	c.platformCacheMux.Lock()
	defer c.platformCacheMux.Unlock()

	// Copy the static mapping into the client's cache
	c.chainToPlatform = GetChainMapping()

	if c.logger != nil {
		c.logger.Info("loaded static chain-to-platform mapping",
			zap.Int("num_chains", len(c.chainToPlatform)))
	}
}

// GetTokenURL returns a user-friendly CoinGecko URL for a token given its chain and contract address.
// The URL format is: https://www.coingecko.com/en/coins/{coin-id}
//
// This function:
// 1. Looks up the CoinGecko platform for the chain
// 2. Queries the CoinGecko API for the token's coin ID
// 3. Returns the formatted URL
//
// Parameters:
//   - chainID: Wormhole chain ID
//   - contractAddr: Token contract address (with or without 0x prefix)
//
// Returns:
//   - URL string (e.g., "https://www.coingecko.com/en/coins/coinbase-wrapped-btc")
//   - Error if chain not supported or token not found
//
// Example:
//
//	client := coingecko.NewClient("", logger)
//	client.UseStaticChainMapping()
//	url, err := client.GetTokenURL(vaa.ChainIDBase, "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(url) // https://www.coingecko.com/en/coins/coinbase-wrapped-btc
func (c *Client) GetTokenURL(chainID vaa.ChainID, contractAddr string) (string, error) {
	// Get platform for this chain
	platformID := c.GetPlatformForChain(chainID)
	if platformID == "" {
		return "", fmt.Errorf("chain %s (%d) is not supported by CoinGecko", chainID, chainID)
	}

	// Normalize contract address (ensure it has 0x prefix and is lowercase)
	contractAddr = strings.ToLower(strings.TrimSpace(contractAddr))
	if !strings.HasPrefix(contractAddr, "0x") {
		contractAddr = "0x" + contractAddr
	}

	// Build the API query URL
	var query string
	if c.apiKey == "" {
		query = fmt.Sprintf("%s/coins/%s/contract/%s", freeAPIBaseURL, platformID, contractAddr)
	} else {
		query = fmt.Sprintf("%s/coins/%s/contract/%s?x_cg_pro_api_key=%s",
			proAPIBaseURL, platformID, contractAddr, c.apiKey)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(context.Background(), "GET", query, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header if using demo/free API
	if c.apiKey != "" && strings.Contains(query, "api.coingecko.com") {
		req.Header.Add("x-cg-demo-api-key", c.apiKey)
	}

	// Make HTTP request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := common.SafeRead(resp.Body)
		return "", fmt.Errorf("CoinGecko API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to get the coin ID
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ID == "" {
		return "", fmt.Errorf("no coin ID returned for token")
	}

	// Build the user-friendly URL
	url := fmt.Sprintf("https://www.coingecko.com/en/coins/%s", result.ID)

	if c.logger != nil {
		c.logger.Debug("generated CoinGecko URL",
			zap.Stringer("chain", chainID),
			zap.String("contract", contractAddr),
			zap.String("coin_id", result.ID),
			zap.String("url", url))
	}

	return url, nil
}

// TokenInfo represents detailed information about a token from CoinGecko.
type TokenInfo struct {
	Symbol      string  // Token symbol (e.g., "USDC")
	CoinGeckoID string  // CoinGecko coin ID (e.g., "usd-coin")
	Decimals    int     // Number of decimals
	Price       float64 // Current price in USD
}

// GetTokenInfo queries CoinGecko for detailed token information by contract address.
// Returns symbol, CoinGecko ID, decimals, and current USD price.
//
// This is useful for getting comprehensive token data in a single API call.
//
// Example:
//
//	client := coingecko.NewClient("", logger)
//	client.UseStaticChainMapping()
//	info, err := client.GetTokenInfo(vaa.ChainIDEthereum, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("%s: $%.2f\n", info.Symbol, info.Price)
func (c *Client) GetTokenInfo(chainID vaa.ChainID, contractAddr string) (*TokenInfo, error) {
	// Get platform for this chain
	platformID := c.GetPlatformForChain(chainID)
	if platformID == "" {
		return nil, fmt.Errorf("chain %s (%d) is not supported by CoinGecko", chainID, chainID)
	}

	// Normalize contract address (ensure it has 0x prefix and is lowercase)
	contractAddr = strings.ToLower(strings.TrimSpace(contractAddr))
	if !strings.HasPrefix(contractAddr, "0x") {
		contractAddr = "0x" + contractAddr
	}

	// Build the API query URL
	var query string
	if c.apiKey == "" {
		query = fmt.Sprintf("%s/coins/%s/contract/%s", freeAPIBaseURL, platformID, contractAddr)
	} else {
		query = fmt.Sprintf("%s/coins/%s/contract/%s?x_cg_pro_api_key=%s",
			proAPIBaseURL, platformID, contractAddr, c.apiKey)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(context.Background(), "GET", query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header if using demo/free API
	if c.apiKey != "" && strings.Contains(query, "api.coingecko.com") {
		req.Header.Add("x-cg-demo-api-key", c.apiKey)
	}

	// Make HTTP request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := common.SafeRead(resp.Body)
		return nil, fmt.Errorf("CoinGecko API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response - need to import the governor package's struct
	var result struct {
		ID              string `json:"id"`
		Symbol          string `json:"symbol"`
		Name            string `json:"name"`
		DetailPlatforms map[string]struct {
			DecimalPlace    int    `json:"decimal_place"`
			ContractAddress string `json:"contract_address"`
		} `json:"detail_platforms"`
		MarketData struct {
			CurrentPrice map[string]float64 `json:"current_price"`
		} `json:"market_data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract information
	info := &TokenInfo{
		Symbol:      strings.ToUpper(result.Symbol),
		CoinGeckoID: result.ID,
	}

	// Get price
	if usdPrice, ok := result.MarketData.CurrentPrice["usd"]; ok {
		info.Price = usdPrice
	} else {
		return nil, fmt.Errorf("no USD price available for this token")
	}

	// Get decimals from detail_platforms
	if details, ok := result.DetailPlatforms[platformID]; ok {
		info.Decimals = details.DecimalPlace
	} else {
		// Fallback: try any platform
		for _, details := range result.DetailPlatforms {
			if details.DecimalPlace > 0 {
				info.Decimals = details.DecimalPlace
				break
			}
		}
		if info.Decimals == 0 {
			return nil, fmt.Errorf("could not determine token decimals")
		}
	}

	if c.logger != nil {
		c.logger.Debug("fetched token info",
			zap.Stringer("chain", chainID),
			zap.String("contract", contractAddr),
			zap.String("symbol", info.Symbol),
			zap.String("coin_id", info.CoinGeckoID),
			zap.Int("decimals", info.Decimals),
			zap.Float64("price", info.Price))
	}

	return info, nil
}
