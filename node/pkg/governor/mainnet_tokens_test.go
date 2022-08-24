package governor

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTokenListSize(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assuming that governed tokens will not go down over time,
	   lets set a floor to avoid parsing or loading regressions */
	assert.Greater(t, len(tokenConfigEntries), 122)
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

	chains := []uint16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 18}

	/* Assume that all chains will have governed tokens */
	for _, chain := range chains {
		t.Run(vaa.ChainID(chain).String(), func(t *testing.T) {
			found := false
			for _, tokenConfigEntry := range tokenConfigEntries {
				if tokenConfigEntry.chain == chain {
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
	for x, tokenConfigEntry1 := range tokenConfigEntries {
		for y, tokenConfigEntry2 := range tokenConfigEntries {
			if x == y {
				// don't flag duplicates at the same index
				continue
			}

			assert.NotEqual(t, tokenConfigEntry1.addr, tokenConfigEntry2.addr)
		}
	}
}

func TestTokenListDecimalRange(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all governed token entries will have decimals of 6 or 8 */
	for _, tokenConfigEntry := range tokenConfigEntries {
		assert.Less(t, tokenConfigEntry.decimals, int64(9))
		assert.NotEqual(t, tokenConfigEntry.decimals, int64(7))
		assert.Greater(t, tokenConfigEntry.decimals, int64(5))
	}
}

func TestTokenListEmptySymbols(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all governed token entry strings will be greater than zero */
	for _, tokenConfigEntry := range tokenConfigEntries {
		assert.Greater(t, len(tokenConfigEntry.symbol), 0)
	}
}

func TestTokenListEmptyCoinGeckoId(t *testing.T) {
	tokenConfigEntries := tokenList()

	/* Assume that all governed token entry strings will be greater than zero */
	for _, tokenConfigEntry := range tokenConfigEntries {
		assert.Greater(t, len(tokenConfigEntry.coinGeckoId), 0)
	}
}
