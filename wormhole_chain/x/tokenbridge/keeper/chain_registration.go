package keeper

import (
	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetChainRegistration set a specific chainRegistration in the store from its index
func (k Keeper) SetChainRegistration(ctx sdk.Context, chainRegistration types.ChainRegistration) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ChainRegistrationKeyPrefix))
	b := k.cdc.MustMarshal(&chainRegistration)
	store.Set(types.ChainRegistrationKey(
		chainRegistration.ChainID,
	), b)
}

// GetChainRegistration returns a chainRegistration from its index
func (k Keeper) GetChainRegistration(
	ctx sdk.Context,
	chainID uint32,

) (val types.ChainRegistration, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ChainRegistrationKeyPrefix))

	b := store.Get(types.ChainRegistrationKey(chainID))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveChainRegistration removes a chainRegistration from the store
func (k Keeper) RemoveChainRegistration(
	ctx sdk.Context,
	chainID uint32,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ChainRegistrationKeyPrefix))
	store.Delete(types.ChainRegistrationKey(
		chainID,
	))
}

// GetAllChainRegistration returns all chainRegistration
func (k Keeper) GetAllChainRegistration(ctx sdk.Context) (list []types.ChainRegistration) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ChainRegistrationKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.ChainRegistration
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
