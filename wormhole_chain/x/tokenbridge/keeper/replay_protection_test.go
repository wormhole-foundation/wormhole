package keeper_test

import (
	"strconv"
	"testing"

	keepertest "github.com/certusone/wormhole-chain/testutil/keeper"
	"github.com/certusone/wormhole-chain/x/tokenbridge/keeper"
	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNReplayProtection(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.ReplayProtection {
	items := make([]types.ReplayProtection, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetReplayProtection(ctx, items[i])
	}
	return items
}

func TestReplayProtectionGet(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	items := createNReplayProtection(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetReplayProtection(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t, item, rst)
	}
}
func TestReplayProtectionRemove(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	items := createNReplayProtection(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveReplayProtection(ctx,
			item.Index,
		)
		_, found := keeper.GetReplayProtection(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestReplayProtectionGetAll(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	items := createNReplayProtection(keeper, ctx, 10)
	require.ElementsMatch(t, items, keeper.GetAllReplayProtection(ctx))
}
