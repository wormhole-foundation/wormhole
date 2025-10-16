package governor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestTokenListSize(t *testing.T) {
	tokenConfigEntries := TokenList()

	// We should have a sensible number of tokens
	// These numbers shouldn't have to change frequently
	assert.Greater(t, len(tokenConfigEntries), 1000)
	// We throttle CoinGecko queries so we can query up to 12,000 tokens
	// in a 15 minute window. This test is an early warning for updating
	// the CoinGecko query mechanism.
	assert.Less(t, len(tokenConfigEntries), 10000)
}

func TestTokenListAddressSize(t *testing.T) {
	tokenConfigEntries := TokenList()

	/* Assume that token addresses must always be 32 bytes (64 chars) */
	for _, tokenConfigEntry := range tokenConfigEntries {
		testLabel := vaa.ChainID(tokenConfigEntry.Chain).String() + ":" + tokenConfigEntry.Symbol
		t.Run(testLabel, func(t *testing.T) {
			assert.Equal(t, len(tokenConfigEntry.Addr), 64)
		})
	}
}

// Flag a situation where a Governed chain does not have any governed assets. Often times when adding a mainnet chain,
// a list of tokens will be added so that they can be governed. (These tokens are sourced by CoinGecko or manually
// populated.) While this is not a hard requirement, it may represent that a developer has forgotten to take the step
// of configuring tokens when deploying the chain. This test helps to remind them.
func TestGovernedChainHasGovernedAssets(t *testing.T) {
	// Add a chain ID to this set if it genuinely has no native assets that should be governed.
	ignoredChains := map[vaa.ChainID]bool{
		// Wormchain is an abstraction over IBC-connected chains so no assets are "native" to it
		vaa.ChainIDWormchain: true,
		// TODO: Remove this once we have governed tokens for Mezo.
		vaa.ChainIDMezo: true,
		// TODO: Remove this once we have governed tokens for XRPLEVM.
		vaa.ChainIDXRPLEVM: true,
		// TODO: Remove this once we have governed tokens for Linea.
		vaa.ChainIDLinea: true,
	}
	if len(ignoredChains) > 0 {
		ignoredOutput := []string{}
		for id := range ignoredChains {
			ignoredOutput = append(ignoredOutput, id.String())
		}

		t.Logf("This test ignored the following chains: %s\n", strings.Join(ignoredOutput, "\n"))
	}

	tokenConfigEntries := TokenList()

	for _, chainConfigEntry := range ChainList() {
		e := chainConfigEntry.EmitterChainID
		if _, ignored := ignoredChains[e]; ignored {
			continue
		}
		t.Run(e.String(), func(t *testing.T) {
			found := false
			for _, tokenConfigEntry := range tokenConfigEntries {
				if tokenConfigEntry.Chain == uint16(e) {
					found = true
					break
				}
			}
			assert.True(t, found, "Chain is governed but has no governed native assets configured")
		})
	}

	// Make sure we're not ignoring any chains with governed tokens.
	for _, tokenEntry := range TokenList() {
		t.Run(vaa.ChainID(tokenEntry.Chain).String(), func(t *testing.T) {
			if _, exists := ignoredChains[vaa.ChainID(tokenEntry.Chain)]; exists {
				assert.Fail(t, "Chain is in ignoredChains but it has governed tokens")
			}
		})
	}
}

func TestTokenListTokenAddressDuplicates(t *testing.T) {
	tokenConfigEntries := TokenList()

	/* Assume that all governed token entry addresses won't include duplicates */
	addrs := make(map[string]string)
	for _, e := range tokenConfigEntries {
		// In a few cases, the same address exists on multiple chains, so we need to compare both the chain and the address.
		// Also using that as the map payload so if we do have a duplicate, we can print out something meaningful.
		key := fmt.Sprintf("%v:%v", e.Chain, e.Addr)
		assert.Equal(t, "", addrs[key])
		addrs[key] = key + ":" + e.Symbol
	}
}

func TestTokenListEmptySymbols(t *testing.T) {
	tokenConfigEntries := TokenList()

	/* Assume that all governed token entry symbol strings will be greater than zero */
	for _, tokenConfigEntry := range tokenConfigEntries {
		// Some Solana tokens don't have the symbol set. For now, we'll still enforce this for other chains.
		if len(tokenConfigEntry.Symbol) == 0 && vaa.ChainID(tokenConfigEntry.Chain) != vaa.ChainIDSolana {
			assert.Equal(t, "", fmt.Sprintf("token %v:%v does not have the symbol set", tokenConfigEntry.Chain, tokenConfigEntry.Addr))
		}
	}
}

func TestTokenListEmptyCoinGeckoId(t *testing.T) {
	tokenConfigEntries := TokenList()

	/* Assume that all governed token entry coingecko id strings will be greater than zero */
	for _, tokenConfigEntry := range tokenConfigEntries {
		assert.Greater(t, len(tokenConfigEntry.CoinGeckoId), 0)
	}
}

// TestManualTokensNotInGeneratedList ensures that tokens in the manual list are not duplicated in the generated list.
// The manual list should only be used to fill gaps in automated token addition. If a token appears in both lists,
// it should be removed from the manual list since it's already covered by the generated list.
func TestManualTokensNotInGeneratedList(t *testing.T) {
	manualTokens := manualTokenList()
	generatedTokens := generatedMainnetTokenList()

	// Build a map of generated tokens for O(1) lookup efficiency
	// Key format: "chain:address" to uniquely identify each token
	generatedTokenMap := make(map[string]TokenConfigEntry)
	for _, token := range generatedTokens {
		key := fmt.Sprintf("%v:%v", token.Chain, token.Addr)
		generatedTokenMap[key] = token
	}

	// Check each manual token against the generated list
	var duplicates []string
	for _, manualToken := range manualTokens {
		key := fmt.Sprintf("%v:%v", manualToken.Chain, manualToken.Addr)
		if generatedToken, exists := generatedTokenMap[key]; exists {
			duplicates = append(duplicates,
				fmt.Sprintf("Chain %v (%s), Address %s, Symbol %s (manual) / %s (generated)",
					manualToken.Chain,
					vaa.ChainID(manualToken.Chain).String(),
					manualToken.Addr,
					manualToken.Symbol,
					generatedToken.Symbol))
		}
	}

	if len(duplicates) > 0 {
		assert.Fail(t, fmt.Sprintf("Found %d token(s) in both manual and generated lists. "+
			"These should be removed from manual_tokens.go:\n%s",
			len(duplicates),
			strings.Join(duplicates, "\n")))
	}
}
