package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// TestLatestGuardianSetIndex tests the querying of the latest guardian set index
func TestLatestGuardianSetIndex(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)

	// Invalid query with nil request
	_, err := k.LatestGuardianSetIndex(ctx, nil)
	require.Error(t, err)

	// Query the latest guardian set index - should be empty
	res, err := k.LatestGuardianSetIndex(ctx, &types.QueryLatestGuardianSetIndexRequest{})
	require.NoError(t, err)
	require.NotNil(t, res)
	fmt.Println(res)
	require.Equal(t, uint32(0xffffffff), res.LatestGuardianSetIndex)

	// Create guardian set
	guardians, _ := createNGuardianValidator(k, ctx, 10)
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})

	createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{
		Index: 0,
	})

	// Query the latest guardian set index - after population
	res, err = k.LatestGuardianSetIndex(ctx, &types.QueryLatestGuardianSetIndexRequest{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint32(0), res.LatestGuardianSetIndex)
}
