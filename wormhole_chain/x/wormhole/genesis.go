package wormhole

import (
	"github.com/certusone/wormhole-chain/x/wormhole/keeper"
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the guardianSet
	for _, elem := range genState.GuardianSetList {
		k.SetGuardianSet(ctx, elem)
	}

	// Set guardianSet count
	k.SetGuardianSetCount(ctx, genState.GuardianSetCount)
	// Set if defined
	if genState.Config != nil {
		k.SetConfig(ctx, *genState.Config)
	}
	// Set all the replayProtection
	for _, elem := range genState.ReplayProtectionList {
		k.SetReplayProtection(ctx, elem)
	}
	// Set all the sequenceCounter
	for _, elem := range genState.SequenceCounterList {
		k.SetSequenceCounter(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()

	genesis.GuardianSetList = k.GetAllGuardianSet(ctx)
	genesis.GuardianSetCount = k.GetGuardianSetCount(ctx)
	// Get all config
	config, found := k.GetConfig(ctx)
	if found {
		genesis.Config = &config
	}
	genesis.ReplayProtectionList = k.GetAllReplayProtection(ctx)
	genesis.SequenceCounterList = k.GetAllSequenceCounter(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
