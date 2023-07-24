package keeper

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func (k Keeper) SetWasmInstantiateAllowlist(ctx sdk.Context, entry types.WasmInstantiateAllowedContractCodeId) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.WasmInstantiateAllowlistKey))
	b := k.cdc.MustMarshal(&entry)
	codeIdStr := strconv.FormatUint(entry.CodeId, 10)
	store.Set([]byte(entry.ContractAddress+codeIdStr), b)
}

func (k Keeper) HasWasmInstantiateAllowlist(ctx sdk.Context, contract string, codeId uint64) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.WasmInstantiateAllowlistKey))
	codeIdStr := strconv.FormatUint(codeId, 10)
	return store.Has([]byte(contract + codeIdStr))
}

func (k Keeper) GetAllWasmInstiateAllowedAddresses(ctx sdk.Context) (list []types.WasmInstantiateAllowedContractCodeId) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.WasmInstantiateAllowlistKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.WasmInstantiateAllowedContractCodeId
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) KeeperDeleteWasmInstantiateAllowlist(ctx sdk.Context, entry types.WasmInstantiateAllowedContractCodeId) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.WasmInstantiateAllowlistKey))
	codeIdStr := strconv.FormatUint(entry.CodeId, 10)
	store.Delete([]byte(entry.ContractAddress + codeIdStr))
}
