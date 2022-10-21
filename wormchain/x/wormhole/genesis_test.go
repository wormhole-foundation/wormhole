package wormhole_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
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
		Config: &types.Config{},
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
		ConsensusGuardianSetIndex: &types.ConsensusGuardianSetIndex{
			Index: 70,
		},
		GuardianValidatorList: []types.GuardianValidator{
			{
				GuardianKey: []byte{0},
			},
			{
				GuardianKey: []byte{1},
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	require.NoError(t, genesisState.Validate())

	k, ctx := keepertest.WormholeKeeper(t)
	wormhole.InitGenesis(ctx, *k, genesisState)
	got := wormhole.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	require.Len(t, got.GuardianSetList, len(genesisState.GuardianSetList))
	require.Subset(t, genesisState.GuardianSetList, got.GuardianSetList)
	require.Equal(t, genesisState.Config, got.Config)
	require.Len(t, got.ReplayProtectionList, len(genesisState.ReplayProtectionList))
	require.Subset(t, genesisState.ReplayProtectionList, got.ReplayProtectionList)
	require.Len(t, got.SequenceCounterList, len(genesisState.SequenceCounterList))
	require.Subset(t, genesisState.SequenceCounterList, got.SequenceCounterList)
	require.Equal(t, genesisState.ConsensusGuardianSetIndex, got.ConsensusGuardianSetIndex)
	require.ElementsMatch(t, genesisState.GuardianValidatorList, got.GuardianValidatorList)
	// this line is used by starport scaffolding # genesis/test/assert
}
