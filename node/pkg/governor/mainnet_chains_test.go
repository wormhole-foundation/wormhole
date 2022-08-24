package governor

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestChainListSize(t *testing.T) {
	chainConfigEntries := chainList()

	/* Assuming that governed chains will not go down over time,
	   lets set a floor of expected chains to guard against parsing
	   or loading regressions */
	assert.Greater(t, len(chainConfigEntries), 14)
}

func TestChainDailyLimitRange(t *testing.T) {
	chainConfigEntries := chainList()

	/* Assuming that a governed chains should always be more than zero and less than 50,000,001 */
	for _, chainConfigEntry := range chainConfigEntries {
		t.Run(chainConfigEntry.emitterChainID.String(), func(t *testing.T) {
			assert.Greater(t, chainConfigEntry.dailyLimit, uint64(0))
			assert.Less(t, chainConfigEntry.dailyLimit, uint64(50000001))
		})
	}
}

func TestChainListChainPresent(t *testing.T) {
	chainConfigEntries := chainList()

	chains := []uint16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 18}

	/* Assume that all chains will have governed tokens */
	for _, chain := range chains {
		t.Run(vaa.ChainID(chain).String(), func(t *testing.T) {
			found := false
			for _, chainConfigEntry := range chainConfigEntries {
				if chainConfigEntry.emitterChainID == vaa.ChainID(chain) {
					found = true
					break
				}
			}

			assert.Equal(t, found, true)
		})
	}
}
