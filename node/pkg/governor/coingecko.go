// This file contains the code related to querying CoinGecko for token prices.
// The CoinGecko API is documented here: https://docs.coingecko.com/
// An example of the query to be generated: https://api.coingecko.com/api/v3/simple/price?ids=gemma-extending-tech,bitcoin,weth&vs_currencies=usd
package governor

import (
	"net/url"
	"strings"
	"time"
)

const (
	// coinGeckoUpdateInterval specifies how often we update prices for tokens.
	coinGeckoUpdateInterval = 15 * time.Minute

	// coinGeckoRequestInterval acts as a rate limiter for batches of individual token queries.
	coinGeckoRequestInterval = 15 * time.Second

	// tokensPerCoinGeckoQuery specifies how many tokens will be in each CoinGecko query. The token list will be broken up into chunks of this size.
	tokensPerCoinGeckoQuery = 200
)

// CoinGecko API response structures for /coins/{platform}/contract/{address} endpoint

// CoinGeckoContractResponse represents the response from CoinGecko's contract address lookup endpoint
type CoinGeckoContractResponse struct {
	ID              string                    `json:"id"`
	Symbol          string                    `json:"symbol"`
	Name            string                    `json:"name"`
	DetailPlatforms map[string]PlatformDetail `json:"detail_platforms"`
	MarketData      MarketData                `json:"market_data"`
}

// PlatformDetail contains platform-specific token information
type PlatformDetail struct {
	DecimalPlace    int    `json:"decimal_place"`
	ContractAddress string `json:"contract_address"`
}

// MarketData contains current market information for a token
type MarketData struct {
	CurrentPrice map[string]float64 `json:"current_price"`
}

// createCoinGeckoQueries creates the set of CoinGecko queries, breaking the set of IDs into the appropriate size chunks.
func createCoinGeckoQueries(idList []string, tokensPerQuery int, coinGeckoApiKey string) []string {
	// Calculate the exact number of chunks needed.
	// This uses ceiling division: ceil(n/d) = (n + d - 1) / d
	// Examples: ceil(250/200) = 2, ceil(200/200) = 1, ceil(199/200) = 1
	numQueries := (len(idList) + tokensPerQuery - 1) / tokensPerQuery
	queries := make([]string, 0, numQueries)

	// Process tokens in chunks of tokensPerQuery size
	for chunkStart := 0; chunkStart < len(idList); chunkStart += tokensPerQuery {
		chunkEnd := min(chunkStart+tokensPerQuery, len(idList))
		queries = append(queries, createCoinGeckoQuery(idList[chunkStart:chunkEnd], coinGeckoApiKey))
	}

	return queries
}

// createCoinGeckoQuery creates a CoinGecko query for the specified set of IDs.
func createCoinGeckoQuery(ids []string, coinGeckoApiKey string) string {
	params := url.Values{}
	params.Add("ids", strings.Join(ids, ","))
	params.Add("vs_currencies", "usd")

	// If modifying this code, ensure that the test 'TestCoinGeckoPriceChecks' passes when adding a pro API key to it.
	// Since the code requires an API key (which we don't want to publish to git), this
	// part of the test is normally skipped but mods to sensitive places should still be checked
	query := ""
	if coinGeckoApiKey == "" {
		query = "https://api.coingecko.com/api/v3/simple/price?" + params.Encode()
	} else { // Pro version API key path
		params.Add("x_cg_pro_api_key", coinGeckoApiKey)
		query = "https://pro-api.coingecko.com/api/v3/simple/price?" + params.Encode()
	}

	return query
}
