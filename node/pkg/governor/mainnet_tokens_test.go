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
	assert.Equal(t, 809, len(tokenConfigEntries))
}

// Helper method to provide per chain token counts, to detect regressions in token tracking
func getTokenCount(chainId vaa.ChainID) int {
	var count = 0
	for _, tce := range tokenList() {
		if vaa.ChainID(tce.chain) == chainId {
			count = count + 1
		}
	}
	return count
}

func TestTokenListSizePerChain(t *testing.T) {

	tests := []struct {
		chainId vaa.ChainID
		num     int
	}{
		/**
		 * SECURITY: If you are seeing a drop in tracked tokens for a given chain,
		 * make sure to understand why to prevent having a regression in the tokens
		 * we are tracking to be governed.
		 */
		{vaa.ChainIDSolana, 122},
		{vaa.ChainIDEthereum, 244},
		{vaa.ChainIDTerra, 8},
		{vaa.ChainIDBSC, 204},
		{vaa.ChainIDPolygon, 84},
		{vaa.ChainIDAvalanche, 35},
		{vaa.ChainIDAurora, 11},
		{vaa.ChainIDFantom, 28},
		{vaa.ChainIDKarura, 4},
		{vaa.ChainIDAcala, 3},
		{vaa.ChainIDKlaytn, 6},
		{vaa.ChainIDCelo, 5},
		{vaa.ChainIDNear, 3},
		{vaa.ChainIDMoonbeam, 13},
		{vaa.ChainIDTerra2, 2},
		{vaa.ChainIDInjective, 1},
		{vaa.ChainIDAptos, 6},
		{vaa.ChainIDArbitrum, 18},
		{vaa.ChainIDOptimism, 5},
		{vaa.ChainIDPythNet, 0},
		{vaa.ChainIDXpla, 1},
		{vaa.ChainIDBtc, 0},
		{vaa.ChainIDBase, 0},
		{vaa.ChainIDSei, 0},
		{vaa.ChainIDWormchain, 0},
		{vaa.ChainIDSepolia, 0},
		{vaa.ChainIDOasis, 2},
	}
	for _, tc := range tests {
		t.Run(tc.chainId.String(), func(t *testing.T) {
			assert.Equal(t, tc.num, getTokenCount(tc.chainId))
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
