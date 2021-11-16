package tokenbridge

import (
	"github.com/certusone/wormhole-chain/x/tokenbridge/keeper"
	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set if defined
	if genState.Config != nil {
		k.SetConfig(ctx, *genState.Config)
	}
	// Set all the replayProtection
	for _, elem := range genState.ReplayProtectionList {
		k.SetReplayProtection(ctx, elem)
	}
	// Set all the chainRegistration
	for _, elem := range genState.ChainRegistrationList {
		k.SetChainRegistration(ctx, elem)
	}
	// Set all the coinMetaRollbackProtection
	for _, elem := range genState.CoinMetaRollbackProtectionList {
		k.SetCoinMetaRollbackProtection(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()

	// Get all config
	config, found := k.GetConfig(ctx)
	if found {
		genesis.Config = &config
	}
	genesis.ReplayProtectionList = k.GetAllReplayProtection(ctx)
	genesis.ChainRegistrationList = k.GetAllChainRegistration(ctx)
	genesis.CoinMetaRollbackProtectionList = k.GetAllCoinMetaRollbackProtection(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
