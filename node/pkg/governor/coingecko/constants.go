// This file contains the code related to querying CoinGecko for token prices.
// The CoinGecko API is documented here: https://docs.coingecko.com/
// An example of the query to be generated: https://api.coingecko.com/api/v3/simple/price?ids=gemma-extending-tech,bitcoin,weth&vs_currencies=usd
package coingecko

import (
	"time"
)

const (
	// CoinGeckoRequestInterval acts as a rate limiter for batches of individual token queries.
	CoinGeckoRequestInterval = 15 * time.Second

	// TokensPerCoinGeckoQuery specifies how many tokens will be in each CoinGecko query. The token list will be broken up into chunks of this size.
	TokensPerCoinGeckoQuery = 200
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
