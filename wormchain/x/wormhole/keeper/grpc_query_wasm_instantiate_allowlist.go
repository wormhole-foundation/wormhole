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

func (k Keeper) WasmInstantiateAllowlistAll(c context.Context, req *types.QueryAllWasmInstantiateAllowlist) (*types.QueryAllWasmInstantiateAllowlistResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	wasmInstantiateAllowlistStore := prefix.NewStore(store, types.KeyPrefix(types.WasmInstantiateAllowlistKey))

	var allowlist []types.WasmInstantiateAllowedContractCodeId

	pageRes, err := query.Paginate(wasmInstantiateAllowlistStore, req.Pagination, func(key []byte, value []byte) error {
		var allowedContractCodeId types.WasmInstantiateAllowedContractCodeId
		if err := k.cdc.Unmarshal(value, &allowedContractCodeId); err != nil {
			return err
		}

		allowlist = append(allowlist, allowedContractCodeId)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllWasmInstantiateAllowlistResponse{Allowlist: allowlist, Pagination: pageRes}, nil
}
