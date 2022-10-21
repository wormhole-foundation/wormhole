package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GuardianSetAll(c context.Context, req *types.QueryAllGuardianSetRequest) (*types.QueryAllGuardianSetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var guardianSets []types.GuardianSet
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	guardianSetStore := prefix.NewStore(store, types.KeyPrefix(types.GuardianSetKey))

	pageRes, err := query.Paginate(guardianSetStore, req.Pagination, func(key []byte, value []byte) error {
		var guardianSet types.GuardianSet
		if err := k.cdc.Unmarshal(value, &guardianSet); err != nil {
			return err
		}

		guardianSets = append(guardianSets, guardianSet)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGuardianSetResponse{GuardianSet: guardianSets, Pagination: pageRes}, nil
}

func (k Keeper) GuardianSet(c context.Context, req *types.QueryGetGuardianSetRequest) (*types.QueryGetGuardianSetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	guardianSet, found := k.GetGuardianSet(ctx, req.Index)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetGuardianSetResponse{GuardianSet: guardianSet}, nil
}
