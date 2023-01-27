package keeper_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/testutil/nullify"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// Create N guardians and return both their public and private keys
func createNGuardianValidator(keeper *keeper.Keeper, ctx sdk.Context, n int) ([]types.GuardianValidator, []*ecdsa.PrivateKey) {
	items := make([]types.GuardianValidator, n)
	privKeys := []*ecdsa.PrivateKey{}
	for i := range items {
		privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		if err != nil {
			panic(err)
		}
		guardianAddr := crypto.PubkeyToAddress(privKey.PublicKey)
		privKeyValidator, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		if err != nil {
			panic(err)
		}

		validatorAddr := crypto.PubkeyToAddress(privKeyValidator.PublicKey)
		items[i].GuardianKey = guardianAddr[:]
		items[i].ValidatorAddr = validatorAddr[:]
		privKeys = append(privKeys, privKey)

		keeper.SetGuardianValidator(ctx, items[i])
	}
	return items, privKeys
}

func createNewGuardianSet(keeper *keeper.Keeper, ctx sdk.Context, guardians []types.GuardianValidator) *types.GuardianSet {
	next_index := keeper.GetGuardianSetCount(ctx)

	guardianSet := &types.GuardianSet{
		Index:          next_index,
		Keys:           [][]byte{},
		ExpirationTime: 0,
	}
	for _, guardian := range guardians {
		guardianSet.Keys = append(guardianSet.Keys, guardian.GuardianKey)
	}

	keeper.AppendGuardianSet(ctx, *guardianSet)
	return guardianSet
}

func TestGuardianValidatorGet(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	items, _ := createNGuardianValidator(keeper, ctx, 10)
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
	items, _ := createNGuardianValidator(keeper, ctx, 10)
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
	items, _ := createNGuardianValidator(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllGuardianValidator(ctx)),
	)
}
