package coingecko

import (
	"fmt"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// chainToPlatformMap is a hard-coded mapping of Wormhole ChainIDs to CoinGecko platform IDs.
// This map was generated from the CoinGecko /asset_platforms endpoint and manual mappings
// for chains that don't match predictably.
//
// Last updated: December 2024
// Source: Generated via `guardiand governor chain-mapping`
var chainToPlatformMap = map[vaa.ChainID]string{
	1:    "solana",
	2:    "ethereum",
	3:    "terra",
	4:    "binance-smart-chain",
	5:    "polygon-pos",
	6:    "avalanche",
	8:    "algorand",
	10:   "fantom",
	13:   "klay-token", // Klaytn (now called Kaia on CoinGecko)
	14:   "celo",
	15:   "near-protocol",
	16:   "moonbeam",
	19:   "injective",
	20:   "osmosis",
	21:   "sui",
	22:   "aptos",
	23:   "arbitrum-nova",
	24:   "optimistic-ethereum",
	30:   "base",
	31:   "filecoin",
	32:   "sei-network",
	33:   "rootstock",
	34:   "scroll",
	35:   "mantle",
	37:   "x-layer",
	38:   "linea",
	39:   "berachain",
	40:   "sei-v2", // SeiEVM
	41:   "eclipse",
	42:   "bob-network",
	44:   "unichain",
	45:   "world-chain",
	46:   "ink",
	47:   "hyperevm",
	48:   "monad",
	49:   "movement",
	50:   "mezo",
	52:   "sonic",
	54:   "codex",
	55:   "plume-network",
	57:   "xrpl-evm",
	58:   "plasma",
	60:   "stacks",
	61:   "stellar",
	62:   "the-open-network", // TON
	64:   "megaeth",
	4000: "cosmos", // Cosmoshub
	4001: "evmos",
	4002: "kujira",
	4003: "neutron",
	4004: "celestia",
	4005: "stargaze",
	4007: "dymension",
	4008: "provenance",
	4009: "noble",
}

// GetChainMapping returns the complete hard-coded mapping of Wormhole ChainIDs
// to CoinGecko platform IDs.
//
// This is a static mapping that doesn't require querying the CoinGecko API.
// Use this when you need the full mapping or want to avoid API calls.
//
// Returns a copy of the map to prevent external modifications.
func GetChainMapping() map[vaa.ChainID]string {
	// Return a copy to prevent external modifications
	result := make(map[vaa.ChainID]string, len(chainToPlatformMap))
	for k, v := range chainToPlatformMap {
		result[k] = v
	}
	return result
}

// GetPlatform returns the CoinGecko platform ID for a given Wormhole ChainID
// using the hard-coded mapping.
//
// Returns an empty string if the chain is not found in the mapping.
//
// This is a fast, static lookup that doesn't require querying the CoinGecko API.
// Use this when you know the chain ID and just need the platform ID.
//
// Example:
//
//	platformID := coingecko.GetPlatform(vaa.ChainIDEthereum)
//	// Returns: "ethereum"
//
//	platformID := coingecko.GetPlatform(vaa.ChainIDBSC)
//	// Returns: "binance-smart-chain"
func GetPlatform(chainID vaa.ChainID) string {
	return chainToPlatformMap[chainID]
}

// IsPlatformSupported checks if a given Wormhole ChainID has a CoinGecko platform mapping.
//
// This is useful for checking if a chain is supported before attempting to query prices.
//
// Example:
//
//	if coingecko.IsPlatformSupported(vaa.ChainIDEthereum) {
//	    platform := coingecko.GetPlatform(vaa.ChainIDEthereum)
//	    // Query prices...
//	}
func IsPlatformSupported(chainID vaa.ChainID) bool {
	_, exists := chainToPlatformMap[chainID]
	return exists
}

// FormatTokenURL creates a user-friendly CoinGecko URL for a token given its coin ID.
// This is a simple formatting function that doesn't make any API calls.
//
// The coin ID is the unique identifier CoinGecko uses for each token
// (e.g., "coinbase-wrapped-btc", "usd-coin", "ethereum").
//
// Example:
//
//	url := coingecko.FormatTokenURL("coinbase-wrapped-btc")
//	// Returns: "https://www.coingecko.com/en/coins/coinbase-wrapped-btc"
//
//	url := coingecko.FormatTokenURL("usd-coin")
//	// Returns: "https://www.coingecko.com/en/coins/usd-coin"
func FormatTokenURL(coinID string) string {
	return fmt.Sprintf("https://www.coingecko.com/en/coins/%s", coinID)
}
