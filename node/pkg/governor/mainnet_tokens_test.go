package governor

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"

	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/stretchr/testify/assert"
)

func TestTokenListSize(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assuming that governed tokens will need to be updated every time
	   we regenerate it */
	assert.Equal(t, len(tokenConfigEntries), 123)
}

func TestTokenListFloor(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that we will never have a floor price of zero,
	   otherwise this would disable the value of the notional
	   value limit for the token */
	for _, tokenConfigEntry := range tokenConfigEntries {
		testLabel := vaa.ChainID(tokenConfigEntry.chain).String() + ":" + tokenConfigEntry.symbol
		t.Run(testLabel, func(t *testing.T) {
			assert.Greater(t, tokenConfigEntry.price, float64(0))
		})
	}

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
	for e := range common.KnownTokenbridgeEmitters {
		t.Run(vaa.ChainID(e).String(), func(t *testing.T) {
			found := false
			for _, tokenConfigEntry := range tokenConfigEntries {
				if tokenConfigEntry.chain == uint16(e) {
					found = true
					break
				}
			}

			assert.Equal(t, found, true)
		})
	}
}

func TestTokenListTokenAddressDuplicates(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all governed token entry addresses won't include duplicates */
	addrs := make(map[string]bool)
	for _, e := range tokenConfigEntries {
		assert.False(t, addrs[e.addr])
		addrs[e.addr] = true
	}
}

func TestTokenListDecimalRange(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all governed token entries will have decimals of 6 or 8 */
	for _, tokenConfigEntry := range tokenConfigEntries {
		d := tokenConfigEntry.decimals
		assert.Condition(t, func() bool { return d == 6 || d == 8 })
	}
}

func TestTokenListEmptySymbols(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all governed token entry symbol strings will be greater than zero */
	for _, tokenConfigEntry := range tokenConfigEntries {
		assert.Greater(t, len(tokenConfigEntry.symbol), 0)
	}
}

func TestTokenListEmptyCoinGeckoId(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all governed token entry coingecko id strings will be greater than zero */
	for _, tokenConfigEntry := range tokenConfigEntries {
		assert.Greater(t, len(tokenConfigEntry.coinGeckoId), 0)
	}
}
