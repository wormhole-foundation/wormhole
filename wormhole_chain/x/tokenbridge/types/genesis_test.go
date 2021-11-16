package types_test

import (
	"testing"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	"github.com/stretchr/testify/require"
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
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
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
			desc: "duplicated chainRegistration",
			genState: &types.GenesisState{
				ChainRegistrationList: []types.ChainRegistration{
					{
						ChainID: 0,
					},
					{
						ChainID: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated coinMetaRollbackProtection",
			genState: &types.GenesisState{
				CoinMetaRollbackProtectionList: []types.CoinMetaRollbackProtection{
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
