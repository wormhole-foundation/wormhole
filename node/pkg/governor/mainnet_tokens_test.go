package governor

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"

	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/stretchr/testify/assert"
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

	/* Assume that all chains will have governed tokens */
	for chain, _ := range common.KnownTokenbridgeEmitters {
		t.Run(vaa.ChainID(chain).String(), func(t *testing.T) {
			found := false
			for _, tokenConfigEntry := range tokenConfigEntries {
				if tokenConfigEntry.chain == uint16(chain) {
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
