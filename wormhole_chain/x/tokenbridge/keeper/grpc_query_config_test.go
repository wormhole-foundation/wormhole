package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/certusone/wormhole-chain/testutil/keeper"
	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
)

func TestConfigQuery(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	item := createTestConfig(keeper, ctx)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetConfigRequest
		response *types.QueryGetConfigResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetConfigRequest{},
			response: &types.QueryGetConfigResponse{Config: item},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Config(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.Equal(t, tc.response, response)
			}
		})
	}
}
