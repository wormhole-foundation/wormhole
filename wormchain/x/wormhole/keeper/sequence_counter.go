package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// SetSequenceCounter set a specific sequenceCounter in the store from its index
func (k Keeper) SetSequenceCounter(ctx sdk.Context, sequenceCounter types.SequenceCounter) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SequenceCounterKeyPrefix))
	b := k.cdc.MustMarshal(&sequenceCounter)
	store.Set(types.SequenceCounterKey(
		sequenceCounter.Index,
	), b)
}

// GetSequenceCounter returns a sequenceCounter from its index
func (k Keeper) GetSequenceCounter(
	ctx sdk.Context,
	index string,

) (val types.SequenceCounter, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SequenceCounterKeyPrefix))

	b := store.Get(types.SequenceCounterKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSequenceCounter removes a sequenceCounter from the store
func (k Keeper) RemoveSequenceCounter(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SequenceCounterKeyPrefix))
	store.Delete(types.SequenceCounterKey(
		index,
	))
}

// GetAllSequenceCounter returns all sequenceCounter
func (k Keeper) GetAllSequenceCounter(ctx sdk.Context) (list []types.SequenceCounter) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SequenceCounterKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.SequenceCounter
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
