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

func (k Keeper) GuardianValidatorAll(c context.Context, req *types.QueryAllGuardianValidatorRequest) (*types.QueryAllGuardianValidatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var guardianValidators []types.GuardianValidator
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	guardianValidatorStore := prefix.NewStore(store, types.KeyPrefix(types.GuardianValidatorKeyPrefix))

	pageRes, err := query.Paginate(guardianValidatorStore, req.Pagination, func(key []byte, value []byte) error {
		var guardianValidator types.GuardianValidator
		if err := k.cdc.Unmarshal(value, &guardianValidator); err != nil {
			return err
		}

		guardianValidators = append(guardianValidators, guardianValidator)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGuardianValidatorResponse{GuardianValidator: guardianValidators, Pagination: pageRes}, nil
}

func (k Keeper) GuardianValidator(c context.Context, req *types.QueryGetGuardianValidatorRequest) (*types.QueryGetGuardianValidatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetGuardianValidator(
		ctx,
		req.GuardianKey,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetGuardianValidatorResponse{GuardianValidator: val}, nil
}
