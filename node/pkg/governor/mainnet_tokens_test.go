package governor

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestTokenListSize(t *testing.T) {
	tokenConfigEntries := tokenList()

	// We should have a sensible number of tokens
	// These numbers shouldn't have to change frequently
	assert.Greater(t, len(tokenConfigEntries), 1000)
	// We throttle CoinGecko queries so we can query up to 12,000 tokens
	// in a 15 minute window. This test is an early warning for updating
	// the CoinGecko query mechanism.
	assert.Less(t, len(tokenConfigEntries), 10000)
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

			if e != vaa.ChainIDXpla && e != vaa.ChainIDAptos && e != vaa.ChainIDArbitrum && e != vaa.ChainIDWormchain {
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

// If not true, then there exists some chain that Wormhole governs yet has no governed tokens. That means assets
// native to the chain are not actually governed and can be transferred across all other supported chains without
// increasing their Governor limit. We don't want this.
func TestAllGovernedChainsHaveGovernedAssets(t *testing.T) {

	// Build a list of chains that are not governed we will filter these out. Compare with `mainnet_chains.go`
	// When a new ChainID is added, it must either be added to this list or else have entries populated in
	// one of the list of tokens accessed by `tokenList()`.
	ignoredChains := map[vaa.ChainID]bool{
		// Pyth is special
		vaa.ChainIDPythNet: true,
		// BTC is special
		vaa.ChainIDBtc: true,
		// Wormchain is a Wormhole abstraction over IBC type networks
		vaa.ChainIDWormchain: true,
		// From the perspective of the Governor, all of these are "Wormchain" because they use IBC.
		vaa.ChainIDCosmoshub:  true,
		vaa.ChainIDEvmos:      true,
		vaa.ChainIDKujira:     true,
		vaa.ChainIDNeutron:    true,
		vaa.ChainIDCelestia:   true,
		vaa.ChainIDStargaze:   true,
		vaa.ChainIDSeda:       true,
		vaa.ChainIDDymension:  true,
		vaa.ChainIDProvenance: true,
		// ID reserved but not yet Governed
		vaa.ChainIDBerachain: true,
		vaa.ChainIDSnaxchain: true,
		// Testnets
		vaa.ChainIDHolesky:         true,
		vaa.ChainIDSepolia:         true,
		vaa.ChainIDArbitrumSepolia: true,
		vaa.ChainIDBaseSepolia:     true,
		vaa.ChainIDOptimismSepolia: true,
		vaa.ChainIDPolygonSepolia:  true,
		// Otherwise archived/inactive from the Governor's perspective
		vaa.ChainIDRootstock: true,
		vaa.ChainIDLinea:     true,
		vaa.ChainIDGnosis:    true,
		vaa.ChainIDOsmosis:   true,
	}

	chainsWithNoGovernedAssets := []vaa.ChainID{}

	// Scan all governed tokens and build a set of all chain IDs found in the "Origin" field of the tokens.
	originChains := make(map[uint16]bool)
	for _, token := range tokenList() {
		_, ok := originChains[token.chain]
		if !ok {
			originChains[token.chain] = true
		}
	}

	// For all governed chains, make sure that they showed up when we scanned all the Origins of the governed tokens.
	// If not, add them to the list of chains that should be governed yet have no assets configured for them.
	for _, id := range vaa.GetAllNetworkIDs() {
		if _, ok := originChains[uint16(id)]; !ok {
			if _, artificial := ignoredChains[id]; !artificial {
				chainsWithNoGovernedAssets = append(chainsWithNoGovernedAssets, id)
			}
		}
	}

	if len(chainsWithNoGovernedAssets) > 0 {
		output := []string{}
		for _, id := range chainsWithNoGovernedAssets {
			output = append(output, id.String())

		}
		sort.Strings(output)
		t.Logf("Governed chains without governed assets: %s\n", strings.Join(output, "\n"))
	}

	if len(ignoredChains) > 0 {
		ignoredOutput := []string{}
		for id, _ := range ignoredChains {
			ignoredOutput = append(ignoredOutput, id.String())
		}

		t.Logf("This test ignored the following chains because they are not governed: %s\n", strings.Join(ignoredOutput, "\n"))
	}
	assert.Zero(t, len(chainsWithNoGovernedAssets))
}
