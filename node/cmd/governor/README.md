# Governor CLI Utilities

Command-line utilities for interacting with Governor functionality and CoinGecko API.

## Commands

### `asset-platforms`

List all CoinGecko asset platforms.

```bash
# Free tier
guardiand governor asset-platforms

# With API key
guardiand governor asset-platforms --api-key YOUR_API_KEY

# JSON output
guardiand governor asset-platforms --output json
```

**Flags:**
- `--api-key`: CoinGecko API key (optional, uses free tier if not provided)
- `--output, -o`: Output format: `table` (default) or `json`

**Example Output:**
```
ID                             NAME                 CHAIN_ID   SHORTNAME
------------------------------------------------------------------------------------
ethereum                       Ethereum             1          
binance-smart-chain            BNB Smart Chain      56         BSC
polygon-pos                    Polygon POS          137        MATIC
avalanche                      Avalanche            43114      AVAX
...
Total platforms: 150
```

---

### `chain-mapping`

Show the mapping between Wormhole ChainIDs and CoinGecko platform identifiers.

```bash
# Show chain to platform mapping
guardiand governor chain-mapping

# With API key
guardiand governor chain-mapping --api-key YOUR_API_KEY

# JSON output
guardiand governor chain-mapping --output json
```

**Flags:**
- `--api-key`: CoinGecko API key (optional, uses free tier if not provided)
- `--output, -o`: Output format: `table` (default) or `json`

**Example Output:**
```
CHAIN_ID   CHAIN_NAME                COINGECKO_PLATFORM            
--------------------------------------------------------------------------------
1          Ethereum                  ethereum                      
56         BSC                       binance-smart-chain           
137        Polygon                   polygon-pos                   
43114      Avalanche                 avalanche                     
42161      Arbitrum                  arbitrum-one                  
10         Optimism                  optimistic-ethereum           
...
Total mapped chains: 45

Note: Only chains with EVM chain IDs are mapped.
Non-EVM chains (e.g., Solana, Aptos) may not appear in CoinGecko's asset platforms.
```

**How It Works:**

CoinGecko's asset platforms include a `chain_identifier` field that contains the EVM chain ID. This command:
1. Fetches all asset platforms from CoinGecko
2. Matches the `chain_identifier` to Wormhole's `vaa.ChainID`
3. Creates a mapping of ChainID → Platform ID

This mapping is useful for:
- Understanding which chains are supported by CoinGecko
- Looking up the correct platform ID for token price queries
- Integrating with the Governor's dynamic token discovery

---

### `token-price`

Query token prices by contract address using either a Wormhole chain ID or CoinGecko platform ID.

**Two Ways to Specify the Chain:**

1. **Using Wormhole Chain ID** (recommended) - Automatically mapped to CoinGecko platform
2. **Using CoinGecko Platform ID** - Direct platform specification

```bash
# Method 1: Using Wormhole Chain ID (RECOMMENDED)
# Get USDC price on Ethereum (chain ID 1)
guardiand governor token-price --chain-id 1 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

# BSC (chain ID 56)
guardiand governor token-price --chain-id 56 0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d

# Polygon (chain ID 137)
guardiand governor token-price --chain-id 137 0x2791bca1f2de4661ed88a30c99a7a9449aa84174

# Multiple tokens on Avalanche (chain ID 43114)
guardiand governor token-price --chain-id 43114 \
  0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e \
  0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7

# Method 2: Using Platform ID directly
guardiand governor token-price --platform ethereum 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

# With API key
guardiand governor token-price \
  --chain-id 1 \
  --api-key YOUR_API_KEY \
  0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

# JSON output
guardiand governor token-price \
  --chain-id 1 \
  --output json \
  0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48
```

**Arguments:**
- `CONTRACT_ADDRESSES...`: One or more contract addresses (required)

**Flags (mutually exclusive):**
- `--chain-id, -c`: Wormhole chain ID (e.g., `1`, `56`, `137`) - **Recommended**
- `--platform, -p`: CoinGecko platform ID (e.g., `ethereum`, `binance-smart-chain`)
- `--api-key`: CoinGecko API key (optional, uses free tier if not provided)
- `--output, -o`: Output format: `table` (default) or `json`

**Note:** You must specify **either** `--chain-id` OR `--platform`, but not both.

**Common Wormhole Chain IDs:**
- `1` - Ethereum
- `56` - BSC (Binance Smart Chain)
- `137` - Polygon
- `43114` - Avalanche
- `42161` - Arbitrum
- `10` - Optimism
- `250` - Fantom
- `8453` - Base
- `59144` - Linea

**Common CoinGecko Platform IDs:**
- `ethereum`
- `binance-smart-chain`
- `polygon-pos`
- `avalanche`
- `arbitrum-one`
- `optimistic-ethereum`
- `fantom`
- `base`
- `linea`

**Example Output:**
```
Using chain Ethereum (1) -> platform: ethereum
CONTRACT ADDRESS                              PRICE (USD)    
------------------------------------------------------------
0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48  $0.999800      
0xdac17f958d2ee523a2206206994597c13d831ec7  $1.000100      
```

**Why Use `--chain-id`?**

Using `--chain-id` is recommended because:
1. **Automatic Mapping**: No need to remember CoinGecko platform names
2. **Validation**: Ensures the chain ID is a known Wormhole chain
3. **Consistency**: Uses the same chain IDs as the Governor
4. **Error Detection**: Warns if a chain is not supported by CoinGecko

When you use `--chain-id`, the command:
1. Validates the chain ID is a known Wormhole chain
2. Fetches CoinGecko's asset platforms
3. Maps the chain ID to the correct platform ID
4. Displays the mapping for transparency

---

### `add-token`

Add a token to `manual_tokens.go` by querying CoinGecko for its information.

This command automates the process of:
1. Looking up token details on CoinGecko
2. Formatting the entry correctly  
3. Adding it to `node/pkg/governor/manual_tokens.go`

```bash
# Add USDC on Ethereum (chain 2)
guardiand governor add-token --chain-id 2 --address 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

# Add token on BSC (chain 4)
guardiand governor add-token --chain-id 4 --address 0x55d398326f99059fF775485246999027B3197955

# Preview without modifying file (dry run)
guardiand governor add-token --chain-id 2 --address 0x... --dry-run

# With API key
guardiand governor add-token \
  --chain-id 2 \
  --address 0x... \
  --api-key YOUR_API_KEY
```

**Arguments:**
- `--chain-id, -c`: Wormhole chain ID (required)
- `--address, -a`: Token contract address (required)

**Flags:**
- `--api-key`: CoinGecko API key (optional)
- `--dry-run`: Print the entry without modifying the file

**Example Output:**
```
Using chain ethereum (2) -> platform: ethereum
Querying CoinGecko for token: 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

Token found!
  Symbol:       USDC
  CoinGecko ID: usd-coin
  Decimals:     6
  Price (USD):  $1.000000

Generated entry:
		{Chain: 2, Addr: "000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Symbol: "USDC", CoinGeckoId: "usd-coin", Decimals: 6, Price: 1.00},

✓ Token added to manual_tokens.go
  Chain: 2
  Address: 000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48
  Symbol: USDC
```

**Important Notes:**
- Provide the standard Ethereum address format (40 hex characters, with or without `0x` prefix)
- CoinGecko is queried using the standard format
- After successful query, the address is automatically converted to Wormhole format (64 hex characters, zero-padded)
- The token must exist on CoinGecko for the specified chain's platform
- The file `manual_tokens.go` must exist and be writable
- Use `--dry-run` to preview the entry before adding it

**Address Format Conversion:**
- Input (standard): `0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48` (40 hex chars)
- CoinGecko query: `0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48` (lowercase)
- Wormhole format: `000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48` (64 hex chars, zero-padded)

## Environment Variables

You can set your CoinGecko API key as an environment variable:

```bash
export COINGECKO_API_KEY=your_api_key_here
guardiand governor token-price --platform ethereum --api-key "$COINGECKO_API_KEY" 0x...
guardiand governor add-token --chain-id 2 --address 0x... --api-key "$COINGECKO_API_KEY"
```

## API Tiers

- **Free Tier**: No API key required, rate-limited
- **Pro Tier**: Requires API key, higher rate limits

The CLI automatically uses the correct endpoint based on whether an API key is provided.

## Integration

These commands use the `node/pkg/governor/coingecko` package, which provides a reusable client for CoinGecko API interactions.

## Error Handling

Common errors:

1. **Invalid platform ID**: Check the platform ID matches CoinGecko's naming (use `asset-platforms` to list all)
2. **Invalid contract address**: Ensure the address is correct and exists on the specified platform
3. **Rate limit exceeded**: Use an API key or wait before retrying
4. **No prices returned**: The contract address may not be indexed by CoinGecko

## Examples

### Get prices for common stablecoins on Ethereum

```bash
guardiand governor token-price --platform ethereum \
  0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48 \  # USDC
  0xdac17f958d2ee523a2206206994597c13d831ec7 \  # USDT
  0x6b175474e89094c44da98b954eedeac495271d0f    # DAI
```

### Export all platforms to JSON for processing

```bash
guardiand governor asset-platforms --output json > platforms.json
jq '.[] | select(.chain_identifier == 1)' platforms.json  # Filter Ethereum mainnet
```

### Check if a token has a price on CoinGecko

```bash
guardiand governor token-price --platform ethereum 0x... || echo "Token not found"
```
