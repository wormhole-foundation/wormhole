package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// SetConfig set config in the store
func (k Keeper) SetConfig(ctx sdk.Context, config types.Config) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ConfigKey))
	b := k.cdc.MustMarshal(&config)
	store.Set([]byte{0}, b)
}

// GetConfig returns config
func (k Keeper) GetConfig(ctx sdk.Context) (val types.Config, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ConfigKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveConfig removes config from the store
func (k Keeper) RemoveConfig(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ConfigKey))
	store.Delete([]byte{0})
}
