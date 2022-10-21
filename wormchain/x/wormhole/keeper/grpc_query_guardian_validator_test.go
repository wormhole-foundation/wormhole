package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/testutil/nullify"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestGuardianValidatorQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs, _ := createNGuardianValidator(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetGuardianValidatorRequest
		response *types.QueryGetGuardianValidatorResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetGuardianValidatorRequest{
				GuardianKey: msgs[0].GuardianKey,
			},
			response: &types.QueryGetGuardianValidatorResponse{GuardianValidator: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetGuardianValidatorRequest{
				GuardianKey: msgs[1].GuardianKey,
			},
			response: &types.QueryGetGuardianValidatorResponse{GuardianValidator: msgs[1]},
		},
		{
			desc:     "KeyNotFound",
			request:  &types.QueryGetGuardianValidatorRequest{GuardianKey: []byte{0, 3, 4}},
			response: &types.QueryGetGuardianValidatorResponse{},
			err:      status.Error(codes.InvalidArgument, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.GuardianValidator(wctx, tc.request)
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

func TestGuardianValidatorQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs, _ := createNGuardianValidator(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllGuardianValidatorRequest {
		return &types.QueryAllGuardianValidatorRequest{
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
			resp, err := keeper.GuardianValidatorAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.GuardianValidator), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.GuardianValidator),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.GuardianValidatorAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.GuardianValidator), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.GuardianValidator),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.GuardianValidatorAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.GuardianValidator),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.GuardianValidatorAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
