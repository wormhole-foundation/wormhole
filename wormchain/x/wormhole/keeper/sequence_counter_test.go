package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNSequenceCounter(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.SequenceCounter {
	items := make([]types.SequenceCounter, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetSequenceCounter(ctx, items[i])
	}
	return items
}

func TestSequenceCounterGet(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	items := createNSequenceCounter(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetSequenceCounter(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t, item, rst)
	}
}
func TestSequenceCounterRemove(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	items := createNSequenceCounter(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveSequenceCounter(ctx,
			item.Index,
		)
		_, found := keeper.GetSequenceCounter(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestSequenceCounterGetAll(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	items := createNSequenceCounter(keeper, ctx, 10)
	require.ElementsMatch(t, items, keeper.GetAllSequenceCounter(ctx))
}
