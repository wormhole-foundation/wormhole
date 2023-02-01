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

func (k Keeper) AllowlistAll(c context.Context, req *types.QueryAllValidatorAllowlist) (*types.QueryAllValidatorAllowlistResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var allowedAddresses []*types.ValidatorAllowedAddress
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	allowedStore := prefix.NewStore(store, types.KeyPrefix(types.ValidatorAllowlistKey))

	pageRes, err := query.Paginate(allowedStore, req.Pagination, func(key []byte, value []byte) error {
		var allowedAddress types.ValidatorAllowedAddress
		if err := k.cdc.Unmarshal(value, &allowedAddress); err != nil {
			return err
		}

		allowedAddresses = append(allowedAddresses, &allowedAddress)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllValidatorAllowlistResponse{Allowlist: allowedAddresses, Pagination: pageRes}, nil
}

func (k Keeper) Allowlist(c context.Context, req *types.QueryValidatorAllowlist) (*types.QueryValidatorAllowlistResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	allowedAddresses := k.GetAllAllowedAddresses(ctx)
	allowlist := []*types.ValidatorAllowedAddress{}
	for _, allowed := range allowedAddresses {
		if allowed.ValidatorAddress == req.ValidatorAddress {
			allowlist = append(allowlist, &allowed)
		}
	}

	return &types.QueryValidatorAllowlistResponse{Allowlist: allowlist, ValidatorAddress: req.ValidatorAddress}, nil
}
