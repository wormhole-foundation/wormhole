package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// SetReplayProtection set a specific replayProtection in the store from its index
func (k Keeper) SetReplayProtection(ctx sdk.Context, replayProtection types.ReplayProtection) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReplayProtectionKeyPrefix))
	b := k.cdc.MustMarshal(&replayProtection)
	store.Set(types.ReplayProtectionKey(
		replayProtection.Index,
	), b)
}

// GetReplayProtection returns a replayProtection from its index
func (k Keeper) GetReplayProtection(
	ctx sdk.Context,
	index string,

) (val types.ReplayProtection, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReplayProtectionKeyPrefix))

	b := store.Get(types.ReplayProtectionKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveReplayProtection removes a replayProtection from the store
func (k Keeper) RemoveReplayProtection(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReplayProtectionKeyPrefix))
	store.Delete(types.ReplayProtectionKey(
		index,
	))
}

// GetAllReplayProtection returns all replayProtection
func (k Keeper) GetAllReplayProtection(ctx sdk.Context) (list []types.ReplayProtection) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReplayProtectionKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.ReplayProtection
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
