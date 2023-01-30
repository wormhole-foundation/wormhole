package keeper

import (
	"bytes"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// SetSequenceCounter set a specific sequenceCounter in the store from its index
func (k Keeper) SetValidatorAllowedAddress(ctx sdk.Context, address types.ValidatorAllowedAddress) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorAllowlistKey))
	b := k.cdc.MustMarshal(&address)
	store.Set([]byte(address.AllowedAddress), b)
}

func (k Keeper) GetValidatorAllowedAddress(ctx sdk.Context, address string) types.ValidatorAllowedAddress {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorAllowlistKey))
	b := store.Get([]byte(address))
	var allowedAddr types.ValidatorAllowedAddress
	k.cdc.MustUnmarshal(b, &allowedAddr)
	return allowedAddr
}

func (k Keeper) HasValidatorAllowedAddress(ctx sdk.Context, address string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorAllowlistKey))
	return store.Has([]byte(address))
}

// RemoveSequenceCounter removes a sequenceCounter from the store
func (k Keeper) RemoveValidatorAllowedAddress(ctx sdk.Context, address string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorAllowlistKey))
	store.Delete([]byte(address))
}

func (k Keeper) GetAllAllowedAddresses(ctx sdk.Context) (list []types.ValidatorAllowedAddress) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorAllowlistKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.ValidatorAllowedAddress
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// Checks if a given address is registered as a guardian validator and either:
// * Is in the current guardian set, OR
// * Is in a future guardian set
func (k Keeper) IsAddressValidatorOrFutureValidator(ctx sdk.Context, addr string) bool {
	currentIndex, _ := k.GetConsensusGuardianSetIndex(ctx)
	matchedValidator, found := k.GetGuardianValidatorByValidatorAddress(ctx, addr)
	if !found {
		return false
	}
	// check that the validator is in a current or future guardian set
	guardianSets := k.GetAllGuardianSet(ctx)
	for _, gSet := range guardianSets {
		if gSet.Index >= currentIndex.Index {
			for _, gKey := range gSet.Keys {
				if bytes.Equal(matchedValidator.GuardianKey, gKey) {
					return true
				}
			}
		}
	}
	return false
}

func (k Keeper) GetGuardianValidatorByValidatorAddress(ctx sdk.Context, addr string) (validator types.GuardianValidator, found bool) {
	addrBz, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return
	}
	validators := k.GetAllGuardianValidator(ctx)
	for _, val := range validators {
		if bytes.Equal(val.ValidatorAddr, addrBz) {
			return val, true
		}
	}
	return
}
