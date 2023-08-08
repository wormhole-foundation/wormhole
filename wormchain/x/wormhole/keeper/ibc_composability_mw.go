package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func (k Keeper) StoreIbcComposabilityMwContract(ctx sdk.Context, entry types.IbcComposabilityMwContract) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IbcComposabilityMwContractKey))
	b := k.cdc.MustMarshal(&entry)
	store.Set([]byte{0}, b)
}

func (k Keeper) GetIbcComposabilityMwContract(ctx sdk.Context) types.IbcComposabilityMwContract {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IbcComposabilityMwContractKey))
	entry := store.Get([]byte{0})

	var val types.IbcComposabilityMwContract
	k.cdc.Unmarshal(entry, &val)

	return val
}
