package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) LatestGuardianSetIndex(goCtx context.Context, req *types.QueryLatestGuardianSetIndexRequest) (*types.QueryLatestGuardianSetIndexResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryLatestGuardianSetIndexResponse{
		LatestGuardianSetIndex: k.GetLatestGuardianSetIndex(ctx),
	}, nil
}
