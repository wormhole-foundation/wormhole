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
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestSequenceCounterQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNSequenceCounter(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetSequenceCounterRequest
		response *types.QueryGetSequenceCounterResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetSequenceCounterRequest{
				Index: msgs[0].Index,
			},
			response: &types.QueryGetSequenceCounterResponse{SequenceCounter: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetSequenceCounterRequest{
				Index: msgs[1].Index,
			},
			response: &types.QueryGetSequenceCounterResponse{SequenceCounter: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetSequenceCounterRequest{
				Index: strconv.Itoa(100000),
			},
			err: status.Error(codes.InvalidArgument, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.SequenceCounter(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.Equal(t, tc.response, response)
			}
		})
	}
}

func TestSequenceCounterQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNSequenceCounter(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllSequenceCounterRequest {
		return &types.QueryAllSequenceCounterRequest{
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
			resp, err := keeper.SequenceCounterAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.SequenceCounter), step)
			require.Subset(t, msgs, resp.SequenceCounter)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.SequenceCounterAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.SequenceCounter), step)
			require.Subset(t, msgs, resp.SequenceCounter)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.SequenceCounterAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.SequenceCounterAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
