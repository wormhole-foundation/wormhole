package cli_test

import (
	"fmt"
	"testing"

	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/stretchr/testify/require"
	tmcli "github.com/tendermint/tendermint/libs/cli"
	"google.golang.org/grpc/status"

	"github.com/wormhole-foundation/wormchain/testutil/network"
	"github.com/wormhole-foundation/wormchain/testutil/nullify"
	"github.com/wormhole-foundation/wormchain/x/wormhole/client/cli"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func networkWithConsensusGuardianSetIndexObjects(t *testing.T) (*network.Network, types.ConsensusGuardianSetIndex) {
	t.Helper()
	cfg := network.DefaultConfig()
	state := types.GenesisState{}
	require.NoError(t, cfg.Codec.UnmarshalJSON(cfg.GenesisState[types.ModuleName], &state))

	consensusGuardianSetIndex := &types.ConsensusGuardianSetIndex{
		Index: 0,
	}
	nullify.Fill(&consensusGuardianSetIndex)
	state.ConsensusGuardianSetIndex = consensusGuardianSetIndex

	guardianSetList := []types.GuardianSet{{
		Index:          0,
		Keys:           [][]byte{},
		ExpirationTime: 0,
	}}
	nullify.Fill(&guardianSetList)
	state.GuardianSetList = guardianSetList

	buf, err := cfg.Codec.MarshalJSON(&state)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf
	return network.New(t, cfg), *state.ConsensusGuardianSetIndex
}

func TestShowConsensusGuardianSetIndex(t *testing.T) {
	net, obj := networkWithConsensusGuardianSetIndexObjects(t)

	ctx := net.Validators[0].ClientCtx
	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}
	for _, tc := range []struct {
		desc string
		args []string
		err  error
		obj  types.ConsensusGuardianSetIndex
	}{
		{
			desc: "get",
			args: common,
			obj:  obj,
		},
	} {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			var args []string
			args = append(args, tc.args...)
			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdShowConsensusGuardianSetIndex(), args)
			if tc.err != nil {
				stat, ok := status.FromError(tc.err)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), tc.err)
			} else {
				require.NoError(t, err)
				var resp types.QueryGetConsensusGuardianSetIndexResponse
				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.ConsensusGuardianSetIndex)
				require.Equal(t,
					nullify.Fill(&tc.obj),
					nullify.Fill(&resp.ConsensusGuardianSetIndex),
				)
			}
		})
	}
}
