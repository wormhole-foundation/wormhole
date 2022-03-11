package keeper

import (
	"context"

	"github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ActiveGuardianSetIndex(c context.Context, req *types.QueryGetActiveGuardianSetIndexRequest) (*types.QueryGetActiveGuardianSetIndexResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetActiveGuardianSetIndex(ctx)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetActiveGuardianSetIndexResponse{ActiveGuardianSetIndex: val}, nil
}
