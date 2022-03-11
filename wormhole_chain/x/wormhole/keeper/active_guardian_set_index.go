package keeper

import (
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetActiveGuardianSetIndex set activeGuardianSetIndex in the store
func (k Keeper) SetActiveGuardianSetIndex(ctx sdk.Context, activeGuardianSetIndex types.ActiveGuardianSetIndex) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ActiveGuardianSetIndexKey))
	b := k.cdc.MustMarshal(&activeGuardianSetIndex)
	store.Set([]byte{0}, b)
}

// GetActiveGuardianSetIndex returns activeGuardianSetIndex
func (k Keeper) GetActiveGuardianSetIndex(ctx sdk.Context) (val types.ActiveGuardianSetIndex, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ActiveGuardianSetIndexKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveActiveGuardianSetIndex removes activeGuardianSetIndex from the store
func (k Keeper) RemoveActiveGuardianSetIndex(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ActiveGuardianSetIndexKey))
	store.Delete([]byte{0})
}
