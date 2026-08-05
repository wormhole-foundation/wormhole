package evm

import (
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	dgAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/delegated_guardians"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// The on-chain WormholeDelegatedGuardians contract accepts configs the node rejects.
// buildDelegatedGuardianConfig must skip those chains (fail closed) instead of
// surfacing an error that would kill the EVM watcher via the polling routine.
func TestBuildDelegatedGuardianConfigSkipsInvalidChains(t *testing.T) {
	logger := zap.NewNop()

	mkKeys := func(n int, offset byte) []common.Address {
		keys := make([]common.Address, n)
		for i := 0; i < n; i++ {
			keys[i] = common.Address{offset, byte(i + 1)}
		}
		return keys
	}

	validKeys := mkKeys(4, 0x01)
	validThreshold := uint8(vaa.CalculateQuorum(4))

	configs := []dgAbi.WormholeDelegatedGuardiansDelegatedGuardianSet{
		// valid chain — must be included
		{ChainId: 2, Threshold: validThreshold, Keys: validKeys, Timestamp: 100},
		// threshold above key count — node rejects, must be skipped not fatal
		{ChainId: 5, Threshold: 7, Keys: mkKeys(3, 0x02), Timestamp: 100},
		// threshold below the quorum floor — node rejects, must be skipped not fatal
		{ChainId: 7, Threshold: 1, Keys: mkKeys(6, 0x03), Timestamp: 100},
		// duplicate keys — node rejects, must be skipped not fatal
		{ChainId: 9, Threshold: 2, Keys: []common.Address{{0x09}, {0x09}}, Timestamp: 100},
		// no keys — already skipped by the pre-existing rule
		{ChainId: 11, Threshold: 0, Keys: nil, Timestamp: 100},
	}

	dgConfig := buildDelegatedGuardianConfig(configs, logger)

	assert.Len(t, dgConfig.Chains, 1, "only the valid chain config must be included")
	_, ok := dgConfig.Chains[vaa.ChainID(2)]
	assert.True(t, ok, "valid chain must survive")
	for _, id := range []uint16{5, 7, 9, 11} {
		_, ok := dgConfig.Chains[vaa.ChainID(id)]
		assert.False(t, ok, "chain %d with invalid config must be skipped", id)
	}
}
