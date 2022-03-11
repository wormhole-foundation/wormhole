package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/certusone/wormhole-chain/testutil/keeper"
	"github.com/certusone/wormhole-chain/testutil/nullify"
	"github.com/certusone/wormhole-chain/x/wormhole/types"
)

func TestActiveGuardianSetIndexQuery(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	item := createTestActiveGuardianSetIndex(keeper, ctx)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetActiveGuardianSetIndexRequest
		response *types.QueryGetActiveGuardianSetIndexResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetActiveGuardianSetIndexRequest{},
			response: &types.QueryGetActiveGuardianSetIndexResponse{ActiveGuardianSetIndex: item},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.ActiveGuardianSetIndex(wctx, tc.request)
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
