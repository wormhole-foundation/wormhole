package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/testutil/nullify"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func TestConsensusGuardianSetIndexQuery(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	item := createTestConsensusGuardianSetIndex(keeper, ctx)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetConsensusGuardianSetIndexRequest
		response *types.QueryGetConsensusGuardianSetIndexResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetConsensusGuardianSetIndexRequest{},
			response: &types.QueryGetConsensusGuardianSetIndexResponse{ConsensusGuardianSetIndex: item},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.ConsensusGuardianSetIndex(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}
