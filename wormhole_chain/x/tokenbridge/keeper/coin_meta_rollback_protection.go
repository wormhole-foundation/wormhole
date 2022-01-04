package keeper

import (
	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetCoinMetaRollbackProtection set a specific coinMetaRollbackProtection in the store from its index
func (k Keeper) SetCoinMetaRollbackProtection(ctx sdk.Context, coinMetaRollbackProtection types.CoinMetaRollbackProtection) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CoinMetaRollbackProtectionKeyPrefix))
	b := k.cdc.MustMarshal(&coinMetaRollbackProtection)
	store.Set(types.CoinMetaRollbackProtectionKey(
		coinMetaRollbackProtection.Index,
	), b)
}

// GetCoinMetaRollbackProtection returns a coinMetaRollbackProtection from its index
func (k Keeper) GetCoinMetaRollbackProtection(
	ctx sdk.Context,
	index string,

) (val types.CoinMetaRollbackProtection, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CoinMetaRollbackProtectionKeyPrefix))

	b := store.Get(types.CoinMetaRollbackProtectionKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveCoinMetaRollbackProtection removes a coinMetaRollbackProtection from the store
func (k Keeper) RemoveCoinMetaRollbackProtection(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CoinMetaRollbackProtectionKeyPrefix))
	store.Delete(types.CoinMetaRollbackProtectionKey(
		index,
	))
}

// GetAllCoinMetaRollbackProtection returns all coinMetaRollbackProtection
func (k Keeper) GetAllCoinMetaRollbackProtection(ctx sdk.Context) (list []types.CoinMetaRollbackProtection) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CoinMetaRollbackProtectionKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.CoinMetaRollbackProtection
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
