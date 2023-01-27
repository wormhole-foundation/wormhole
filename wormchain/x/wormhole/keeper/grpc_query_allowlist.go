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
	sequenceCounterStore := prefix.NewStore(store, types.KeyPrefix(types.ValidatorAllowlistKey))

	pageRes, err := query.Paginate(sequenceCounterStore, req.Pagination, func(key []byte, value []byte) error {
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

	return &types.QueryAllValidatorAllowlistResponse{AllowedAddress: allowedAddresses, Pagination: pageRes}, nil
}

func (k Keeper) Allowlist(c context.Context, req *types.QueryValidatorAllowlist) (*types.QueryValidatorAllowlistResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var allowedAddresses []*types.ValidatorAllowedAddress
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	sequenceCounterStore := prefix.NewStore(store, types.KeyPrefix(types.ValidatorAllowlistKey))

	pageRes, err := query.Paginate(sequenceCounterStore, req.Pagination, func(key []byte, value []byte) error {
		var allowedAddress types.ValidatorAllowedAddress
		if err := k.cdc.Unmarshal(value, &allowedAddress); err != nil {
			return err
		}
		// this will cause a less then expected amount to be returned in the pagination,
		// but the alternative is to rework how pagination works, which is complex.
		// this lists are not expected to be long anyways and won't need pagination.
		if allowedAddress.ValidatorAddress == req.ValidatorAddress {
			allowedAddresses = append(allowedAddresses, &allowedAddress)
		}
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryValidatorAllowlistResponse{AllowedAddress: allowedAddresses, Pagination: pageRes, ValidatorAddress: req.ValidatorAddress}, nil
}
