package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) IbcComposabilityMwContract(c context.Context, req *types.QueryIbcComposabilityMwContractRequest) (*types.QueryIbcComposabilityMwContractResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	ibcComposabilityMwContract := k.GetIbcComposabilityMwContract(ctx)

	return &types.QueryIbcComposabilityMwContractResponse{ContractAddress: ibcComposabilityMwContract.ContractAddress}, nil
}
