package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ConsensusGuardianSetIndex(c context.Context, req *types.QueryGetConsensusGuardianSetIndexRequest) (*types.QueryGetConsensusGuardianSetIndexResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetConsensusGuardianSetIndex(ctx)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetConsensusGuardianSetIndexResponse{ConsensusGuardianSetIndex: val}, nil
}
