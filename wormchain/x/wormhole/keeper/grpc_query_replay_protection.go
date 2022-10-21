package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ReplayProtectionAll(c context.Context, req *types.QueryAllReplayProtectionRequest) (*types.QueryAllReplayProtectionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var replayProtections []types.ReplayProtection
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	replayProtectionStore := prefix.NewStore(store, types.KeyPrefix(types.ReplayProtectionKeyPrefix))

	pageRes, err := query.Paginate(replayProtectionStore, req.Pagination, func(key []byte, value []byte) error {
		var replayProtection types.ReplayProtection
		if err := k.cdc.Unmarshal(value, &replayProtection); err != nil {
			return err
		}

		replayProtections = append(replayProtections, replayProtection)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllReplayProtectionResponse{ReplayProtection: replayProtections, Pagination: pageRes}, nil
}

func (k Keeper) ReplayProtection(c context.Context, req *types.QueryGetReplayProtectionRequest) (*types.QueryGetReplayProtectionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetReplayProtection(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetReplayProtectionResponse{ReplayProtection: val}, nil
}
