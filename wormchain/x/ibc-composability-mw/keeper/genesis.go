package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/types"
)

// InitGenesis
func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
	store := ctx.KVStore(k.storeKey)
	for key, value := range state.TransposedDataInFlight {
		store.Set([]byte(key), value)
	}
}

// ExportGenesis
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	store := ctx.KVStore(k.storeKey)

	transposedDataInFlight := make(map[string][]byte)

	itr := store.Iterator(nil, nil)
	for ; itr.Valid(); itr.Next() {
		transposedDataInFlight[string(itr.Key())] = itr.Value()
	}
	return &types.GenesisState{TransposedDataInFlight: transposedDataInFlight}
}
