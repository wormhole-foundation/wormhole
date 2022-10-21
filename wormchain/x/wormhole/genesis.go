package wormhole

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the guardianSet
	for _, elem := range genState.GuardianSetList {
		if _, err := k.AppendGuardianSet(ctx, elem); err != nil {
			panic(err)
		}
	}

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
	// Set if defined
	if genState.ConsensusGuardianSetIndex != nil {
		k.SetConsensusGuardianSetIndex(ctx, *genState.ConsensusGuardianSetIndex)
	}
	// Set all the guardianValidator
	for _, elem := range genState.GuardianValidatorList {
		k.SetGuardianValidator(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()

	genesis.GuardianSetList = k.GetAllGuardianSet(ctx)

	// Get all config
	config, found := k.GetConfig(ctx)
	if found {
		genesis.Config = &config
	}
	genesis.ReplayProtectionList = k.GetAllReplayProtection(ctx)
	genesis.SequenceCounterList = k.GetAllSequenceCounter(ctx)
	// Get all consensusGuardianSetIndex
	consensusGuardianSetIndex, found := k.GetConsensusGuardianSetIndex(ctx)
	if found {
		genesis.ConsensusGuardianSetIndex = &consensusGuardianSetIndex
	}
	genesis.GuardianValidatorList = k.GetAllGuardianValidator(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
