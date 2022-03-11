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

func createTestActiveGuardianSetIndex(keeper *keeper.Keeper, ctx sdk.Context) types.ActiveGuardianSetIndex {
	item := types.ActiveGuardianSetIndex{}
	keeper.SetActiveGuardianSetIndex(ctx, item)
	return item
}

func TestActiveGuardianSetIndexGet(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	item := createTestActiveGuardianSetIndex(keeper, ctx)
	rst, found := keeper.GetActiveGuardianSetIndex(ctx)
	require.True(t, found)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func TestActiveGuardianSetIndexRemove(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	createTestActiveGuardianSetIndex(keeper, ctx)
	keeper.RemoveActiveGuardianSetIndex(ctx)
	_, found := keeper.GetActiveGuardianSetIndex(ctx)
	require.False(t, found)
}
