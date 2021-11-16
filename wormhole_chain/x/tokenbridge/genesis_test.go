package tokenbridge_test

import (
	"testing"

	keepertest "github.com/certusone/wormhole-chain/testutil/keeper"
	"github.com/certusone/wormhole-chain/x/tokenbridge"
	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Config: &types.Config{},
		ReplayProtectionList: []types.ReplayProtection{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		ChainRegistrationList: []types.ChainRegistration{
			{
				ChainID: 0,
			},
			{
				ChainID: 1,
			},
		},
		CoinMetaRollbackProtectionList: []types.CoinMetaRollbackProtection{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.TokenbridgeKeeper(t)
	tokenbridge.InitGenesis(ctx, *k, genesisState)
	got := tokenbridge.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	require.Equal(t, genesisState.Config, got.Config)
	require.Len(t, got.ReplayProtectionList, len(genesisState.ReplayProtectionList))
	require.Subset(t, genesisState.ReplayProtectionList, got.ReplayProtectionList)
	require.Len(t, got.ChainRegistrationList, len(genesisState.ChainRegistrationList))
	require.Subset(t, genesisState.ChainRegistrationList, got.ChainRegistrationList)
	require.Len(t, got.CoinMetaRollbackProtectionList, len(genesisState.CoinMetaRollbackProtectionList))
	require.Subset(t, genesisState.CoinMetaRollbackProtectionList, got.CoinMetaRollbackProtectionList)
	// this line is used by starport scaffolding # genesis/test/assert
}
