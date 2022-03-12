package wormhole_test

import (
	"testing"

	keepertest "github.com/certusone/wormhole-chain/testutil/keeper"
	"github.com/certusone/wormhole-chain/x/wormhole"
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		GuardianSetList: []types.GuardianSet{
			{
				Index: 0,
			},
			{
				Index: 1,
			},
		},
		GuardianSetCount: 2,
		Config:           &types.Config{},
		ReplayProtectionList: []types.ReplayProtection{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		SequenceCounterList: []types.SequenceCounter{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		ActiveGuardianSetIndex: &types.ActiveGuardianSetIndex{
			Index: 70,
		},
		GuardianValidatorList: []types.GuardianValidator{
			{
				GuardianKey: "0",
			},
			{
				GuardianKey: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.WormholeKeeper(t)
	wormhole.InitGenesis(ctx, *k, genesisState)
	got := wormhole.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	require.Len(t, got.GuardianSetList, len(genesisState.GuardianSetList))
	require.Subset(t, genesisState.GuardianSetList, got.GuardianSetList)
	require.Equal(t, genesisState.GuardianSetCount, got.GuardianSetCount)
	require.Equal(t, genesisState.Config, got.Config)
	require.Len(t, got.ReplayProtectionList, len(genesisState.ReplayProtectionList))
	require.Subset(t, genesisState.ReplayProtectionList, got.ReplayProtectionList)
	require.Len(t, got.SequenceCounterList, len(genesisState.SequenceCounterList))
	require.Subset(t, genesisState.SequenceCounterList, got.SequenceCounterList)
	require.Equal(t, genesisState.ActiveGuardianSetIndex, got.ActiveGuardianSetIndex)
	require.ElementsMatch(t, genesisState.GuardianValidatorList, got.GuardianValidatorList)
	// this line is used by starport scaffolding # genesis/test/assert
}
