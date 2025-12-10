package coingecko

import (
	"fmt"
	"maps"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// chainToPlatformMap is a hard-coded mapping of Wormhole ChainIDs to CoinGecko platform IDs.
// This map was generated from the CoinGecko /asset_platforms endpoint and manual mappings
// for chains that don't match predictably.
//
// Source: Generated via `guardiand governor chain-mapping`
var chainToPlatformMap = map[vaa.ChainID]string{
	vaa.ChainIDSolana:     "solana",
	vaa.ChainIDEthereum:   "ethereum",
	vaa.ChainIDTerra:      "terra",
	vaa.ChainIDBSC:        "binance-smart-chain",
	vaa.ChainIDPolygon:    "polygon-pos",
	vaa.ChainIDAvalanche:  "avalanche",
	vaa.ChainIDAlgorand:   "algorand",
	vaa.ChainIDFantom:     "fantom",
	vaa.ChainIDKlaytn:     "klay-token", // Klaytn (now called Kaia on CoinGecko)
	vaa.ChainIDCelo:       "celo",
	vaa.ChainIDNear:       "near-protocol",
	vaa.ChainIDMoonbeam:   "moonbeam",
	vaa.ChainIDInjective:  "injective",
	vaa.ChainIDOsmosis:    "osmosis",
	vaa.ChainIDSui:        "sui",
	vaa.ChainIDAptos:      "aptos",
	23:                    "arbitrum-nova", // No vaa.ChainIDArbitrumNova constant available
	vaa.ChainIDOptimism:   "optimistic-ethereum",
	vaa.ChainIDBase:       "base",
	vaa.ChainIDFileCoin:   "filecoin",
	vaa.ChainIDSei:        "sei-network",
	vaa.ChainIDRootstock:  "rootstock",
	vaa.ChainIDScroll:     "scroll",
	vaa.ChainIDMantle:     "mantle",
	vaa.ChainIDXLayer:     "x-layer",
	vaa.ChainIDLinea:      "linea",
	vaa.ChainIDBerachain:  "berachain",
	vaa.ChainIDSeiEVM:     "sei-v2",
	vaa.ChainIDEclipse:    "eclipse",
	vaa.ChainIDBOB:        "bob-network",
	vaa.ChainIDUnichain:   "unichain",
	vaa.ChainIDWorldchain: "world-chain",
	vaa.ChainIDInk:        "ink",
	vaa.ChainIDHyperEVM:   "hyperevm",
	vaa.ChainIDMonad:      "monad",
	vaa.ChainIDMovement:   "movement",
	vaa.ChainIDMezo:       "mezo",
	vaa.ChainIDSonic:      "sonic",
	vaa.ChainIDCodex:      "codex",
	vaa.ChainIDPlume:      "plume-network",
	vaa.ChainIDXRPLEVM:    "xrpl-evm",
	vaa.ChainIDPlasma:     "plasma",
	vaa.ChainIDStacks:     "stacks",
	vaa.ChainIDStellar:    "stellar",
	vaa.ChainIDTON:        "the-open-network",
	vaa.ChainIDMegaETH:    "megaeth",
	vaa.ChainIDCosmoshub:  "cosmos",
	vaa.ChainIDEvmos:      "evmos",
	vaa.ChainIDKujira:     "kujira",
	vaa.ChainIDNeutron:    "neutron",
	vaa.ChainIDCelestia:   "celestia",
	vaa.ChainIDStargaze:   "stargaze",
	vaa.ChainIDDymension:  "dymension",
	vaa.ChainIDProvenance: "provenance",
	vaa.ChainIDNoble:      "noble",
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
	maps.Copy(result, chainToPlatformMap)
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
