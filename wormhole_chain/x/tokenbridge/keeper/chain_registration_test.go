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

func createNChainRegistration(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.ChainRegistration {
	items := make([]types.ChainRegistration, n)
	for i := range items {
		items[i].ChainID = uint32(i)

		keeper.SetChainRegistration(ctx, items[i])
	}
	return items
}

func TestChainRegistrationGet(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	items := createNChainRegistration(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetChainRegistration(ctx,
			item.ChainID,
		)
		require.True(t, found)
		require.Equal(t, item, rst)
	}
}
func TestChainRegistrationRemove(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	items := createNChainRegistration(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveChainRegistration(ctx,
			item.ChainID,
		)
		_, found := keeper.GetChainRegistration(ctx,
			item.ChainID,
		)
		require.False(t, found)
	}
}

func TestChainRegistrationGetAll(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	items := createNChainRegistration(keeper, ctx, 10)
	require.ElementsMatch(t, items, keeper.GetAllChainRegistration(ctx))
}
