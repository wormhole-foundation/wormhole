package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/certusone/wormhole-chain/testutil/keeper"
	"github.com/certusone/wormhole-chain/testutil/nullify"
	"github.com/certusone/wormhole-chain/x/wormhole/keeper"
	"github.com/certusone/wormhole-chain/x/wormhole/types"
)

func createTestConsensusGuardianSetIndex(keeper *keeper.Keeper, ctx sdk.Context) types.ConsensusGuardianSetIndex {
	item := types.ConsensusGuardianSetIndex{}
	keeper.SetConsensusGuardianSetIndex(ctx, item)
	return item
}

func TestConsensusGuardianSetIndexGet(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	item := createTestConsensusGuardianSetIndex(keeper, ctx)
	rst, found := keeper.GetConsensusGuardianSetIndex(ctx)
	require.True(t, found)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func TestConsensusGuardianSetIndexRemove(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	createTestConsensusGuardianSetIndex(keeper, ctx)
	keeper.RemoveConsensusGuardianSetIndex(ctx)
	_, found := keeper.GetConsensusGuardianSetIndex(ctx)
	require.False(t, found)
}
