package keeper

import (
	"context"
	"math"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ChainRegistrationAll(c context.Context, req *types.QueryAllChainRegistrationRequest) (*types.QueryAllChainRegistrationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var chainRegistrations []types.ChainRegistration
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	chainRegistrationStore := prefix.NewStore(store, types.KeyPrefix(types.ChainRegistrationKeyPrefix))

	pageRes, err := query.Paginate(chainRegistrationStore, req.Pagination, func(key []byte, value []byte) error {
		var chainRegistration types.ChainRegistration
		if err := k.cdc.Unmarshal(value, &chainRegistration); err != nil {
			return err
		}

		chainRegistrations = append(chainRegistrations, chainRegistration)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllChainRegistrationResponse{ChainRegistration: chainRegistrations, Pagination: pageRes}, nil
}

func (k Keeper) ChainRegistration(c context.Context, req *types.QueryGetChainRegistrationRequest) (*types.QueryGetChainRegistrationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	if req.ChainID > math.MaxUint16 {
		return nil, status.Error(codes.InvalidArgument, "chainID must be uint16")
	}

	val, found := k.GetChainRegistration(
		ctx,
		req.ChainID,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetChainRegistrationResponse{ChainRegistration: val}, nil
}
