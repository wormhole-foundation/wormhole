package keeper

import (
	"context"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) CoinMetaRollbackProtectionAll(c context.Context, req *types.QueryAllCoinMetaRollbackProtectionRequest) (*types.QueryAllCoinMetaRollbackProtectionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var coinMetaRollbackProtections []types.CoinMetaRollbackProtection
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	coinMetaRollbackProtectionStore := prefix.NewStore(store, types.KeyPrefix(types.CoinMetaRollbackProtectionKeyPrefix))

	pageRes, err := query.Paginate(coinMetaRollbackProtectionStore, req.Pagination, func(key []byte, value []byte) error {
		var coinMetaRollbackProtection types.CoinMetaRollbackProtection
		if err := k.cdc.Unmarshal(value, &coinMetaRollbackProtection); err != nil {
			return err
		}

		coinMetaRollbackProtections = append(coinMetaRollbackProtections, coinMetaRollbackProtection)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllCoinMetaRollbackProtectionResponse{CoinMetaRollbackProtection: coinMetaRollbackProtections, Pagination: pageRes}, nil
}

func (k Keeper) CoinMetaRollbackProtection(c context.Context, req *types.QueryGetCoinMetaRollbackProtectionRequest) (*types.QueryGetCoinMetaRollbackProtectionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetCoinMetaRollbackProtection(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetCoinMetaRollbackProtectionResponse{CoinMetaRollbackProtection: val}, nil
}
