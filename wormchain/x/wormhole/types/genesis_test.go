package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func TestGenesisState_Validate(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
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
					Index: 14,
				},
				GuardianValidatorList: []types.GuardianValidator{
					{
						GuardianKey:   []byte{0},
						ValidatorAddr: []byte{3},
					},
					{
						GuardianKey:   []byte{1},
						ValidatorAddr: []byte{4},
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated guardianSet",
			genState: &types.GenesisState{
				GuardianSetList: []types.GuardianSet{
					{
						Index: 0,
					},
					{
						Index: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated replayProtection",
			genState: &types.GenesisState{
				ReplayProtectionList: []types.ReplayProtection{
					{
						Index: "0",
					},
					{
						Index: "0",
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated sequenceCounter",
			genState: &types.GenesisState{
				SequenceCounterList: []types.SequenceCounter{
					{
						Index: "0",
					},
					{
						Index: "0",
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated guardianValidator",
			genState: &types.GenesisState{
				GuardianValidatorList: []types.GuardianValidator{
					{
						GuardianKey:   []byte{0},
						ValidatorAddr: []byte{10},
					},
					{
						GuardianKey:   []byte{1},
						ValidatorAddr: []byte{10},
					},
				},
			},
			valid: true,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
