# CoinGecko API Client

This package provides a client for interacting with the CoinGecko API.

## Usage

### Basic Usage

```go
import "github.com/certusone/wormhole/node/pkg/governor/coingecko"

// Create a client (with API key and logger)
client := coingecko.NewClient("your-api-key", logger)

// Or without API key (free tier)
client := coingecko.NewClient("", nil)

// Fetch all asset platforms
platforms, err := client.AssetPlatforms()
if err != nil {
    log.Fatal(err)
}

// Fetch token prices
prices, err := client.SimpleTokenPrice("ethereum",
    []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"}) // USDC
if err != nil {
    log.Fatal(err)
}

fmt.Printf("USDC price: $%.4f\n", prices[0].Prices["usd"])
```

### Chain to Platform Mapping

#### Using Static Mapping (Recommended - No API Calls)

```go
import (
    "github.com/certusone/wormhole/node/pkg/governor/coingecko"
    "github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Option 1: Use static functions (no client needed)
platformID := coingecko.GetPlatform(vaa.ChainIDEthereum)
fmt.Printf("Ethereum platform: %s\n", platformID) // "ethereum"

// Check if supported
if coingecko.IsPlatformSupported(vaa.ChainIDBSC) {
    platform := coingecko.GetPlatform(vaa.ChainIDBSC)
    fmt.Printf("BSC platform: %s\n", platform) // "binance-smart-chain"
}

// Get all mappings
mapping := coingecko.GetChainMapping()
for chainID, platformID := range mapping {
    fmt.Printf("Chain %d -> Platform %s\n", chainID, platformID)
}

// Option 2: Load static mapping into client
client := coingecko.NewClient("", logger)
client.UseStaticChainMapping() // Loads hard-coded map, no API call
platformID = client.GetPlatformForChain(vaa.ChainIDEthereum)
```

#### Using Dynamic Mapping (Queries CoinGecko API)

```go
// Create client
client := coingecko.NewClient("", logger)

// Build the mapping (fetches from CoinGecko API)
allChains := vaa.GetAllNetworkIDs()
missing, err := client.BuildChainToPlatformMap(allChains)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Missing chains: %v\n", missing)

// Get platform ID for a specific chain
platformID := client.GetPlatformForChain(vaa.ChainIDEthereum)
fmt.Printf("Ethereum platform: %s\n", platformID)
```

## Features

- **Client-based API**: Create a reusable client with optional API key
- **Optional logging**: Pass a `*zap.Logger` for debug/error logging
- **Free & Pro tier support**: Automatically uses the correct endpoint based on API key presence
- **Type-safe**: Struct-based response parsing
- **Chain mapping**: Map Wormhole ChainIDs to CoinGecko platform IDs using `chain_identifier`

## Methods

### Price & Platform Queries

#### `AssetPlatforms() ([]AssetPlatform, error)`
Returns a list of all asset platforms supported by CoinGecko.

#### `SimpleTokenPrice(platformID string, contractAddresses []string) ([]TokenPrice, error)`
Queries the price of one or more tokens by contract address. Currency is hard-coded to USD.

### Chain to Platform Mapping

#### Static Functions (No API Calls)

##### `GetPlatform(chainID vaa.ChainID) string`
Returns the CoinGecko platform ID for a Wormhole chain using a hard-coded mapping. Fast, offline, no API call needed.

##### `GetChainMapping() map[vaa.ChainID]string`
Returns the complete hard-coded mapping of all Wormhole chains to CoinGecko platforms.

##### `IsPlatformSupported(chainID vaa.ChainID) bool`
Checks if a chain has a CoinGecko platform mapping.

#### Client Methods

##### `UseStaticChainMapping()`
Loads the hard-coded chain-to-platform mapping into the client. Use this for fast, offline operation.

##### `BuildChainToPlatformMap(chainIDs []vaa.ChainID) ([]vaa.ChainID, error)`
Fetches all asset platforms from CoinGecko API and builds a dynamic mapping. Returns chains that couldn't be mapped. Use this to verify/update the static mapping.

**Note:** This method queries the API and caches the mapping in the client.

##### `GetPlatformForChain(chainID vaa.ChainID) string`
Returns the CoinGecko platform ID for a given Wormhole ChainID. Returns empty string if not found.

**Prerequisite:** Must call either `UseStaticChainMapping()` or `BuildChainToPlatformMap()` first.

##### `GetChainToPlatformMap() map[vaa.ChainID]string`
Returns a copy of the client's cached chain-to-platform mapping.

##### `GetTokenURL(chainID vaa.ChainID, contractAddr string) (string, error)`
Returns a user-friendly CoinGecko URL for a token. Queries the API to get the token's coin ID, then formats the URL.

**Example:**
```go
client := coingecko.NewClient("", logger)
client.UseStaticChainMapping()
url, err := client.GetTokenURL(vaa.ChainIDBase, "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf")
// Returns: "https://www.coingecko.com/en/coins/coinbase-wrapped-btc"
```

**Prerequisite:** Must call either `UseStaticChainMapping()` or `BuildChainToPlatformMap()` first.

#### Utility Functions

##### `FormatTokenURL(coinID string) string`
Formats a CoinGecko URL from a coin ID. No API call, no client needed.

**Example:**
```go
url := coingecko.FormatTokenURL("coinbase-wrapped-btc")
// Returns: "https://www.coingecko.com/en/coins/coinbase-wrapped-btc"
```

## How Chain Mapping Works

CoinGecko's `/asset_platforms` endpoint returns a `chain_identifier` field for each platform that corresponds to the EVM chain ID (e.g., 1 for Ethereum, 56 for BSC, 137 for Polygon).

The `BuildChainToPlatformMap()` method:
1. Fetches all platforms via `AssetPlatforms()`
2. Filters platforms that have a `chain_identifier`
3. Casts the `chain_identifier` to `vaa.ChainID`
4. Maps `vaa.ChainID` → CoinGecko platform ID (e.g., `1 → "ethereum"`)
5. Caches the mapping for fast lookups

**Example Mappings:**
- `vaa.ChainIDEthereum` (1) → `"ethereum"`
- `vaa.ChainIDBSC` (56) → `"binance-smart-chain"`
- `vaa.ChainIDPolygon` (137) → `"polygon-pos"`
- `vaa.ChainIDAvalanche` (43114) → `"avalanche"`

**Note:** Not all Wormhole chains have a CoinGecko platform (e.g., Solana, Aptos). Use `GetPlatformForChain()` and check for empty string.

## API Key

- **Free tier**: Pass empty string `""` to `NewClient()`
- **Pro tier**: Pass your API key to `NewClient()`

The client automatically uses the correct endpoint based on whether an API key is provided.
