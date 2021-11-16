package keeper_test

import (
	"strconv"
	"testing"

	keepertest "github.com/certusone/wormhole-chain/testutil/keeper"
	"github.com/certusone/wormhole-chain/testutil/nullify"
	"github.com/certusone/wormhole-chain/x/wormhole/keeper"
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNGuardianValidator(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.GuardianValidator {
	items := make([]types.GuardianValidator, n)
	for i := range items {
		items[i].GuardianKey = []byte(strconv.Itoa(i))

		keeper.SetGuardianValidator(ctx, items[i])
	}
	return items
}

func TestGuardianValidatorGet(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	items := createNGuardianValidator(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetGuardianValidator(ctx,
			item.GuardianKey,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestGuardianValidatorRemove(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	items := createNGuardianValidator(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveGuardianValidator(ctx,
			item.GuardianKey,
		)
		_, found := keeper.GetGuardianValidator(ctx,
			item.GuardianKey,
		)
		require.False(t, found)
	}
}

func TestGuardianValidatorGetAll(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	items := createNGuardianValidator(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllGuardianValidator(ctx)),
	)
}
