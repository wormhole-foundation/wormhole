package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// SetGuardianValidator set a specific guardianValidator in the store from its index
func (k Keeper) SetGuardianValidator(ctx sdk.Context, guardianValidator types.GuardianValidator) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuardianValidatorKeyPrefix))
	b := k.cdc.MustMarshal(&guardianValidator)
	store.Set(types.GuardianValidatorKey(
		guardianValidator.GuardianKey,
	), b)
}

// GetGuardianValidator returns a guardianValidator from its index
func (k Keeper) GetGuardianValidator(
	ctx sdk.Context,
	guardianKey []byte,

) (val types.GuardianValidator, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuardianValidatorKeyPrefix))

	b := store.Get(types.GuardianValidatorKey(
		guardianKey,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveGuardianValidator removes a guardianValidator from the store
func (k Keeper) RemoveGuardianValidator(
	ctx sdk.Context,
	guardianKey []byte,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuardianValidatorKeyPrefix))
	store.Delete(types.GuardianValidatorKey(
		guardianKey,
	))
}

// GetAllGuardianValidator returns all guardianValidator
func (k Keeper) GetAllGuardianValidator(ctx sdk.Context) (list []types.GuardianValidator) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuardianValidatorKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.GuardianValidator
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
