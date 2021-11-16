package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/certusone/wormhole-chain/testutil/keeper"
	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestCoinMetaRollbackProtectionQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNCoinMetaRollbackProtection(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetCoinMetaRollbackProtectionRequest
		response *types.QueryGetCoinMetaRollbackProtectionResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetCoinMetaRollbackProtectionRequest{
				Index: msgs[0].Index,
			},
			response: &types.QueryGetCoinMetaRollbackProtectionResponse{CoinMetaRollbackProtection: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetCoinMetaRollbackProtectionRequest{
				Index: msgs[1].Index,
			},
			response: &types.QueryGetCoinMetaRollbackProtectionResponse{CoinMetaRollbackProtection: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetCoinMetaRollbackProtectionRequest{
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
			response, err := keeper.CoinMetaRollbackProtection(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.Equal(t, tc.response, response)
			}
		})
	}
}

func TestCoinMetaRollbackProtectionQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.TokenbridgeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNCoinMetaRollbackProtection(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllCoinMetaRollbackProtectionRequest {
		return &types.QueryAllCoinMetaRollbackProtectionRequest{
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
			resp, err := keeper.CoinMetaRollbackProtectionAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.CoinMetaRollbackProtection), step)
			require.Subset(t, msgs, resp.CoinMetaRollbackProtection)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.CoinMetaRollbackProtectionAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.CoinMetaRollbackProtection), step)
			require.Subset(t, msgs, resp.CoinMetaRollbackProtection)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.CoinMetaRollbackProtectionAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.CoinMetaRollbackProtectionAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
