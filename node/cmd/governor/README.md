# Governor CLI

Commands for managing Governor tokens and querying CoinGecko prices.

## `asset-platforms`

List CoinGecko asset platforms.

```bash
guardiand governor asset-platforms
guardiand governor asset-platforms --api-key YOUR_KEY
guardiand governor asset-platforms --output json
```

**Flags:**
- `--api-key`: CoinGecko API key
- `-o, --output`: `table` or `json` (default: `table`)

---

## `chain-mapping`

Show Wormhole ChainID to CoinGecko platform mapping.

```bash
guardiand governor chain-mapping
guardiand governor chain-mapping --output json
```

**Flags:**
- `--api-key`: CoinGecko API key
- `-o, --output`: `table` or `json`

---

## `token-price`

Query token prices by contract address.

```bash
# Using Wormhole chain ID (recommended)
guardiand governor token-price --chain-id 2 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

# Using CoinGecko platform ID
guardiand governor token-price --platform ethereum 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

# Multiple tokens
guardiand governor token-price --chain-id 2 0xaddr1 0xaddr2 0xaddr3
```

**Arguments:**
- `CONTRACT_ADDRESSES...`: One or more contract addresses

**Flags (mutually exclusive):**
- `-c, --chain-id`: Wormhole chain ID (e.g., `2`, `4`, `5`)
- `-p, --platform`: CoinGecko platform ID (e.g., `ethereum`, `binance-smart-chain`)
- `--api-key`: CoinGecko API key
- `-o, --output`: `table` or `json`

**Common Chain IDs:**
```
2  = Ethereum
4  = BSC
5  = Polygon
6  = Avalanche
23 = Arbitrum
24 = Optimism
30 = Base
```

---

## `add-token`

Add a token to `manual_tokens.go` by querying CoinGecko.

```bash
# Add USDC on Ethereum
guardiand governor add-token --chain-id 2 --address 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

# Preview without modifying
guardiand governor add-token --chain-id 2 --address 0x... --dry-run
```

**Flags:**
- `-c, --chain-id`: Wormhole chain ID (required)
- `-a, --address`: Token contract address (required)
- `--api-key`: CoinGecko API key
- `--dry-run`: Preview without modifying file

**Output:**
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
```

**Note:** Addresses are automatically converted from standard format (40 hex chars) to Wormhole format (64 hex chars, zero-padded).

---

## API Keys

Set your API key as an environment variable:

```bash
export COINGECKO_API_KEY=your_key
guardiand governor token-price --api-key "$COINGECKO_API_KEY" ...
```

The CLI uses the free tier endpoint without an API key, or the pro endpoint when one is provided.
