package governor

import (
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
		If a chain is deprecated, we want to make sure its still governed
		in the case that it is used. This will effectively stall all
		transfers for 24 hours on a deprecated chain.
	*/
	min_daily_limit := uint64(0)

	/* This IS NOT a hard limit, we can adjust it up as we see fit,
	   but setting something sane such that if we accidentally go
	   too high that the unit tests will make sure it's
	   intentional */
	max_daily_limit := uint64(100_000_001)

	// Do not remove this assertion
	assert.NotEqual(t, max_daily_limit, uint64(0))

	/* Assuming that a governed chains should always be more than zero and less than 50,000,001 */
	for _, chainConfigEntry := range chainConfigEntries {
		t.Run(chainConfigEntry.EmitterChainID.String(), func(t *testing.T) {
			assert.GreaterOrEqual(t, chainConfigEntry.DailyLimit, min_daily_limit)
			assert.Less(t, chainConfigEntry.DailyLimit, max_daily_limit)
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

		// If the daily limit is 0 then both the big TX and daily limit should be 0.
		if e.DailyLimit == 0 {
			assert.Equal(t, e.BigTransactionSize, e.DailyLimit)
			continue
		}

		// it's always ideal to have bigTransactionSize be less than dailyLimit
		assert.Less(t, e.BigTransactionSize, e.DailyLimit)

		// in fact, it's even better for bigTransactionSize not to exceed 1/3rd the limit (convention has it at 1/10th to start)
		assert.Less(t, e.BigTransactionSize, e.DailyLimit/3)
	}
}
