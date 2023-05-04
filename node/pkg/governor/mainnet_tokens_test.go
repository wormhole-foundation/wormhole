package governor

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestTokenListSize(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assuming that governed tokens will need to be updated every time
	   we regenerate it */
	assert.Equal(t, 802, len(tokenConfigEntries))
}

func TestTokenListAddressSize(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that token addresses must always be 32 bytes (64 chars) */
	for _, tokenConfigEntry := range tokenConfigEntries {
		testLabel := vaa.ChainID(tokenConfigEntry.chain).String() + ":" + tokenConfigEntry.symbol
		t.Run(testLabel, func(t *testing.T) {
			assert.Equal(t, len(tokenConfigEntry.addr), 64)
		})
	}
}

func TestTokenListChainTokensPresent(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all chains within a token bridge will have governed tokens */
	for e := range sdk.KnownTokenbridgeEmitters {
		t.Run(e.String(), func(t *testing.T) {
			found := false
			for _, tokenConfigEntry := range tokenConfigEntries {
				if tokenConfigEntry.chain == uint16(e) {
					found = true
					break
				}
			}

			if e != vaa.ChainIDXpla && e != vaa.ChainIDAptos && e != vaa.ChainIDArbitrum {
				assert.Equal(t, found, true)
			}
		})
	}
}

func TestTokenListTokenAddressDuplicates(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all governed token entry addresses won't include duplicates */
	addrs := make(map[string]string)
	for _, e := range tokenConfigEntries {
		// In a few cases, the same address exists on multiple chains, so we need to compare both the chain and the address.
		// Also using that as the map payload so if we do have a duplicate, we can print out something meaningful.
		key := fmt.Sprintf("%v:%v", e.chain, e.addr)
		assert.Equal(t, "", addrs[key])
		addrs[key] = key + ":" + e.symbol
	}
}

func TestTokenListEmptySymbols(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all governed token entry symbol strings will be greater than zero */
	for _, tokenConfigEntry := range tokenConfigEntries {
		// Some Solana tokens don't have the symbol set. For now, we'll still enforce this for other chains.
		if len(tokenConfigEntry.symbol) == 0 && vaa.ChainID(tokenConfigEntry.chain) != vaa.ChainIDSolana {
			assert.Equal(t, "", fmt.Sprintf("token %v:%v does not have the symbol set", tokenConfigEntry.chain, tokenConfigEntry.addr))
		}
	}
}

func TestTokenListEmptyCoinGeckoId(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all governed token entry coingecko id strings will be greater than zero */
	for _, tokenConfigEntry := range tokenConfigEntries {
		assert.Greater(t, len(tokenConfigEntry.coinGeckoId), 0)
	}
}
