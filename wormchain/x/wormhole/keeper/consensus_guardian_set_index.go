package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// SetConsensusGuardianSetIndex set consensusGuardianSetIndex in the store
func (k Keeper) SetConsensusGuardianSetIndex(ctx sdk.Context, consensusGuardianSetIndex types.ConsensusGuardianSetIndex) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ConsensusGuardianSetIndexKey))
	b := k.cdc.MustMarshal(&consensusGuardianSetIndex)
	store.Set([]byte{0}, b)
}

// GetConsensusGuardianSetIndex returns consensusGuardianSetIndex
func (k Keeper) GetConsensusGuardianSetIndex(ctx sdk.Context) (val types.ConsensusGuardianSetIndex, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ConsensusGuardianSetIndexKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveConsensusGuardianSetIndex removes consensusGuardianSetIndex from the store
func (k Keeper) RemoveConsensusGuardianSetIndex(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ConsensusGuardianSetIndexKey))
	store.Delete([]byte{0})
}
