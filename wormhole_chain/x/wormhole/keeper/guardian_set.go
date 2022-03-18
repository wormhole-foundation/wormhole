package keeper

import (
	"bytes"
	"encoding/binary"

	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetLatestGuardianSetIndex(ctx sdk.Context) uint32 {
	return k.GetGuardianSetCount(ctx) - 1
}

func (k Keeper) UpdateGuardianSet(ctx sdk.Context, newGuardianSet types.GuardianSet) error {
	config, ok := k.GetConfig(ctx)
	if !ok {
		return types.ErrNoConfig
	}

	oldSet, exists := k.GetGuardianSet(ctx, k.GetGuardianSetCount(ctx)-1)
	if !exists {
		return types.ErrGuardianSetNotFound
	}

	if oldSet.Index+1 != newGuardianSet.Index {
		return types.ErrGuardianSetNotSequential
	}

	// Create new set
	_, err := k.AppendGuardianSet(ctx, types.GuardianSet{
		Keys:           newGuardianSet.Keys,
		Index:          newGuardianSet.Index,
		ExpirationTime: 0,
	})
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

	err = k.TrySwitchToNewConsensusGuardianSet(ctx)
	if err != nil {
		return err
	}
	return err
}

func (k Keeper) TrySwitchToNewConsensusGuardianSet(ctx sdk.Context) error {
	latestGuardianSetIndex := k.GetLatestGuardianSetIndex(ctx)
	consensusGuardianSetIndex, found := k.GetActiveGuardianSetIndex(ctx)
	if !found {
		return types.ErrGuardianSetNotFound
	}

	// nothing to do if the latest set is already the consensus set
	if latestGuardianSetIndex == consensusGuardianSetIndex.Index {
		return nil
	}

	latestGuardianSet, found := k.GetGuardianSet(ctx, latestGuardianSetIndex)
	if !found {
		return types.ErrGuardianSetNotFound
	}

	// count how many registrations we have
	registered := 0
	for _, key := range latestGuardianSet.Keys {
		_, found := k.GetGuardianValidator(ctx, key)
		if found {
			registered++
		}
	}

	// see if we have enough validators registered to produce blocks.
	// TODO(csongor): this has to be kept in sync with tendermint consensus
	quorum := CalculateQuorum(len(latestGuardianSet.Keys))
	if registered >= quorum {
		// we have enough, set consensus set to the latest one. Guardian set upgrade complete.
		k.SetActiveGuardianSetIndex(ctx, types.ActiveGuardianSetIndex{
			Index: latestGuardianSetIndex,
		})
	}

	return nil
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

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuardianSetKey))
	appendedValue := k.cdc.MustMarshal(&guardianSet)
	store.Set(GetGuardianSetIDBytes(guardianSet.Index), appendedValue)

	// Update guardianSet count
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

// TODO(csongor): Should we keep this function? It's linear in the size of the
// guardian set and it's executed on each endblocker when assigning voting power
// to validators. We could rewrite that method instead to loop through the
// guardian keys instead, which might be more efficient.
func (k Keeper) IsConsensusGuardian(ctx sdk.Context, addr sdk.ValAddress) (bool, error) {
	consensusGuardianSetIndex, found := k.GetActiveGuardianSetIndex(ctx)
	if !found {
		return false, types.ErrGuardianSetNotFound
	}

	consensusGuardianSet, found := k.GetGuardianSet(ctx, consensusGuardianSetIndex.Index)
	if !found {
		return false, types.ErrGuardianSetNotFound
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

// GetGuardianSetIDFromBytes returns ID in uint64 format from a byte array
func GetGuardianSetIDFromBytes(bz []byte) uint32 {
	return binary.BigEndian.Uint32(bz)
}
