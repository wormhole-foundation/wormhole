package keeper

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func (k Keeper) SetInstantiateAllowlist(ctx sdk.Context, allowed types.WasmAllowedContractCodeId) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.WasmInstantiateAllowlistKey))
	b := k.cdc.MustMarshal(&allowed)
	codeIdStr := strconv.FormatUint(allowed.CodeId, 10)
	store.Set([]byte(allowed.ContractAddress+codeIdStr), b)
}

func (k Keeper) HasInstantiateAllowlist(ctx sdk.Context, contract string, codeId uint64) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.WasmInstantiateAllowlistKey))
	codeIdStr := strconv.FormatUint(codeId, 10)
	return store.Has([]byte(contract + codeIdStr))
}
