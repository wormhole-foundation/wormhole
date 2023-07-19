package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func (k Keeper) StoreMiddlewareContract(ctx sdk.Context, entry types.WormholeMiddlewareContract) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MiddlewareContractKey))
	b := k.cdc.MustMarshal(&entry)
	store.Set([]byte{0}, b)
}

func (k Keeper) GetMiddlewareContract(ctx sdk.Context) types.WormholeMiddlewareContract {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MiddlewareContractKey))
	entry := store.Get([]byte{0})

	var val types.WormholeMiddlewareContract
	k.cdc.Unmarshal(entry, &val)

	return val
}
