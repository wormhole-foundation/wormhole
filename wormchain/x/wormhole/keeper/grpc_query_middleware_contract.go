package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) WormholeMiddlewareContract(c context.Context, req *types.QueryWormholeMiddlewareContractRequest) (*types.QueryWormholeMiddlewareContractResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	wormholeMiddlewareContract := k.GetMiddlewareContract(ctx)

	return &types.QueryWormholeMiddlewareContractResponse{ContractAddress: wormholeMiddlewareContract.ContractAddress}, nil
}
