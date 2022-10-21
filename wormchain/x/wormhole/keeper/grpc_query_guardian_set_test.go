package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func TestGuardianSetQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNGuardianSet(t, keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetGuardianSetRequest
		response *types.QueryGetGuardianSetResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetGuardianSetRequest{Index: msgs[0].Index},
			response: &types.QueryGetGuardianSetResponse{GuardianSet: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetGuardianSetRequest{Index: msgs[1].Index},
			response: &types.QueryGetGuardianSetResponse{GuardianSet: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetGuardianSetRequest{Index: uint32(len(msgs))},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.GuardianSet(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.response, response)
			}
		})
	}
}

func TestGuardianSetQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNGuardianSet(t, keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllGuardianSetRequest {
		return &types.QueryAllGuardianSetRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.GuardianSetAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.GuardianSet), step)
			require.Subset(t, msgs, resp.GuardianSet)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.GuardianSetAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.GuardianSet), step)
			require.Subset(t, msgs, resp.GuardianSet)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.GuardianSetAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.GuardianSetAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
