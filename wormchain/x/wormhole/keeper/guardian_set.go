package keeper

import (
	"bytes"
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func (k Keeper) GetLatestGuardianSetIndex(ctx sdk.Context) uint32 {
	return k.GetGuardianSetCount(ctx) - 1
}

func (k Keeper) UpdateGuardianSet(ctx sdk.Context, newGuardianSet types.GuardianSet) error {
	config, ok := k.GetConfig(ctx)
	if !ok {
		return types.ErrNoConfig
	}

	oldSet, exists := k.GetGuardianSet(ctx, k.GetLatestGuardianSetIndex(ctx))
	if !exists {
		return types.ErrGuardianSetNotFound
	}

	if oldSet.Index+1 != newGuardianSet.Index {
		return types.ErrGuardianSetNotSequential
	}

	if newGuardianSet.ExpirationTime != 0 {
		return types.ErrNewGuardianSetHasExpiry
	}

	// Create new set
	_, err := k.AppendGuardianSet(ctx, newGuardianSet)
	if err != nil {
		return err
	}

	// Expire old set
	oldSet.ExpirationTime = uint64(ctx.BlockTime().Unix()) + config.GuardianSetExpiration
	k.setGuardianSet(ctx, oldSet)

	// Emit event
	err = ctx.EventManager().EmitTypedEvent(&types.EventGuardianSetUpdate{
		OldIndex: oldSet.Index,
		NewIndex: oldSet.Index + 1,
	})
	if err != nil {
		return err
	}

	return k.TrySwitchToNewConsensusGuardianSet(ctx)
}

func (k Keeper) TrySwitchToNewConsensusGuardianSet(ctx sdk.Context) error {
	latestGuardianSetIndex := k.GetLatestGuardianSetIndex(ctx)
	consensusGuardianSetIndex, found := k.GetConsensusGuardianSetIndex(ctx)
	if !found {
		return types.ErrConsensusSetUndefined
	}

	// nothing to do if the latest set is already the consensus set
	if latestGuardianSetIndex == consensusGuardianSetIndex.Index {
		return nil
	}

	latestGuardianSet, found := k.GetGuardianSet(ctx, latestGuardianSetIndex)
	if !found {
		return types.ErrGuardianSetNotFound
	}

	// make sure each guardian has a registered validator
	for _, key := range latestGuardianSet.Keys {
		_, found := k.GetGuardianValidator(ctx, key)
		// if one of them doesn't, we don't attempt to switch
		if !found {
			return nil
		}
	}

	oldConsensusGuardianSetIndex := consensusGuardianSetIndex.Index
	newConsensusGuardianSetIndex := latestGuardianSetIndex

	// everyone's registered, set consensus set to the latest one. Guardian set upgrade complete.
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{
		Index: newConsensusGuardianSetIndex,
	})

	err := ctx.EventManager().EmitTypedEvent(&types.EventConsensusSetUpdate{
		OldIndex: oldConsensusGuardianSetIndex,
		NewIndex: newConsensusGuardianSetIndex,
	})

	return err
}

// GetGuardianSetCount get the total number of guardianSet
func (k Keeper) GetGuardianSetCount(ctx sdk.Context) uint32 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.GuardianSetCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint32(bz)
}

// setGuardianSetCount set the total number of guardianSet
func (k Keeper) setGuardianSetCount(ctx sdk.Context, count uint32) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.GuardianSetCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint32(bz, count)
	store.Set(byteKey, bz)
}

// AppendGuardianSet appends a guardianSet in the store with a new id and update the count
func (k Keeper) AppendGuardianSet(
	ctx sdk.Context,
	guardianSet types.GuardianSet,
) (uint32, error) {
	// Create the guardianSet
	count := k.GetGuardianSetCount(ctx)

	if guardianSet.Index != count {
		return 0, types.ErrGuardianSetNotSequential
	}

	k.setGuardianSet(ctx, guardianSet)
	k.setGuardianSetCount(ctx, count+1)

	return count, nil
}

// SetGuardianSet set a specific guardianSet in the store
func (k Keeper) setGuardianSet(ctx sdk.Context, guardianSet types.GuardianSet) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuardianSetKey))
	b := k.cdc.MustMarshal(&guardianSet)
	store.Set(GetGuardianSetIDBytes(guardianSet.Index), b)
}

// GetGuardianSet returns a guardianSet from its id
func (k Keeper) GetGuardianSet(ctx sdk.Context, id uint32) (val types.GuardianSet, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuardianSetKey))
	b := store.Get(GetGuardianSetIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// Returns true when the given validator address is registered as a guardian and
// that guardian is a member of the consensus guardian set.
//
// Note that this function is linear in the size of the consensus guardian set,
// and it's eecuted on each endblocker when assigning voting power to validators.
func (k Keeper) IsConsensusGuardian(ctx sdk.Context, addr sdk.ValAddress) (bool, error) {
	// If there are no guardian sets, return true
	// This is useful for testing, but the code path is never encountered when
	// the chain is bootstrapped with a non-empty guardian set at gensis.
	guardianSetCount := k.GetGuardianSetCount(ctx)
	if guardianSetCount == 0 {
		return true, nil
	}

	consensusGuardianSetIndex, found := k.GetConsensusGuardianSetIndex(ctx)
	if !found {
		return false, types.ErrConsensusSetUndefined
	}

	consensusGuardianSet, found := k.GetGuardianSet(ctx, consensusGuardianSetIndex.Index)

	if !found {
		return false, types.ErrGuardianSetNotFound
	}

	// If the consensus guardian set is empty, return true.
	// This is useful for testing, but the code path is never encountered when
	// the chain is bootstrapped with a non-empty guardian set at gensis.
	if len(consensusGuardianSet.Keys) == 0 {
		return true, nil
	}

	isConsensusGuardian := false
	for _, key := range consensusGuardianSet.Keys {
		validator, found := k.GetGuardianValidator(ctx, key)
		if !found {
			continue
		}
		if bytes.Equal(validator.ValidatorAddr, addr.Bytes()) {
			isConsensusGuardian = true
			break
		}
	}

	return isConsensusGuardian, nil
}

// GetAllGuardianSet returns all guardianSet
func (k Keeper) GetAllGuardianSet(ctx sdk.Context) (list []types.GuardianSet) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuardianSetKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.GuardianSet
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetGuardianSetIDBytes returns the byte representation of the ID
func GetGuardianSetIDBytes(id uint32) []byte {
	bz := make([]byte, 4)
	binary.BigEndian.PutUint32(bz, id)
	return bz
}

// GetGuardianSetIDFromBytes returns ID in uint32 format from a byte array
func GetGuardianSetIDFromBytes(bz []byte) uint32 {
	return binary.BigEndian.Uint32(bz)
}
