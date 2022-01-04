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

func createNCoinMetaRollbackProtection(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.CoinMetaRollbackProtection {
	items := make([]types.CoinMetaRollbackProtection, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetCoinMetaRollbackProtection(ctx, items[i])
	}
	return items
}

func TestCoinMetaRollbackProtectionGet(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	items := createNCoinMetaRollbackProtection(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetCoinMetaRollbackProtection(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t, item, rst)
	}
}
func TestCoinMetaRollbackProtectionRemove(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	items := createNCoinMetaRollbackProtection(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveCoinMetaRollbackProtection(ctx,
			item.Index,
		)
		_, found := keeper.GetCoinMetaRollbackProtection(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestCoinMetaRollbackProtectionGetAll(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	items := createNCoinMetaRollbackProtection(keeper, ctx, 10)
	require.ElementsMatch(t, items, keeper.GetAllCoinMetaRollbackProtection(ctx))
}
