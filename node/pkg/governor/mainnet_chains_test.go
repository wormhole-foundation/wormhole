package governor

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestChainListSize(t *testing.T) {
	chainConfigEntries := ChainList()

	/* Assuming that governed chains will not go down over time,
	   lets set a floor of expected chains to guard against parsing
	   or loading regressions */
	assert.Greater(t, len(chainConfigEntries), 14)
}

func TestChainDailyLimitRange(t *testing.T) {
	chainConfigEntries := ChainList()

	/*
		If a chain is deprecated, we want to make sure it's still governed
		in the case that it is used. This will effectively stall all
		transfers for 24 hours on a deprecated chain.
	*/
	minUSDLimit := uint64(0)

	/* This IS NOT a hard limit, we can adjust it up as we see fit,
	   but setting something sane such that if we accidentally go
	   too high that the unit tests will make sure it's
	   intentional */
	maxUSDLimit := uint64(100_000_001)

	// Do not remove this assertion
	assert.NotEqual(t, maxUSDLimit, uint64(0))

	// Assuming that a governed chains should always be within the bounds defined by the min and max.
	for _, chainConfigEntry := range chainConfigEntries {
		t.Run(chainConfigEntry.EmitterChainID.String(), func(t *testing.T) {
			assert.GreaterOrEqual(t, chainConfigEntry.USDLimit, minUSDLimit)
			assert.Less(t, chainConfigEntry.USDLimit, maxUSDLimit)
		})
	}
}

func TestChainListChainPresent(t *testing.T) {
	chainConfigEntries := ChainList()

	entries := make([]vaa.ChainID, 0, len(chainConfigEntries))
	for _, e := range chainConfigEntries {
		entries = append(entries, e.EmitterChainID)
	}

	emitters := make([]vaa.ChainID, 0, len(sdk.KnownTokenbridgeEmitters))
	for e := range sdk.KnownTokenbridgeEmitters {
		emitters = append(emitters, e)
	}

	assert.ElementsMatch(t, entries, emitters)
}

func TestChainListBigTransfers(t *testing.T) {
	chainConfigEntries := ChainList()

	for _, e := range chainConfigEntries {

		// If the chain config's USD limit is 0, then the big TX limit should be zero.
		if e.USDLimit == 0 {
			assert.Equal(t, e.BigTransactionSize, e.USDLimit)
			continue
		}

		// It's always ideal to have bigTransactionSize be less than the USD Limit.
		assert.Less(t, e.BigTransactionSize, e.USDLimit)

		// In fact, it's even better for bigTransactionSize not to exceed 1/3rd the limit (convention has it at 1/10th to start)
		switch e.EmitterChainID {
		// Base and Arbitrum are intentionally configured to not follow the 1/3rd convention for now.
		// However, this check is still useful for other chains.
		case vaa.ChainIDBase, vaa.ChainIDArbitrum:
			continue
		default:
			assert.Less(t, e.BigTransactionSize, e.USDLimit/3,
				fmt.Sprintf(
					"Chain %s has a big transaction size of %d which is more than 1/3 of the UUSD limit of %d",
					e.EmitterChainID,
					e.BigTransactionSize,
					e.USDLimit,
				))
		}
	}
}
