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

func (k Keeper) SequenceCounterAll(c context.Context, req *types.QueryAllSequenceCounterRequest) (*types.QueryAllSequenceCounterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var sequenceCounters []types.SequenceCounter
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	sequenceCounterStore := prefix.NewStore(store, types.KeyPrefix(types.SequenceCounterKeyPrefix))

	pageRes, err := query.Paginate(sequenceCounterStore, req.Pagination, func(key []byte, value []byte) error {
		var sequenceCounter types.SequenceCounter
		if err := k.cdc.Unmarshal(value, &sequenceCounter); err != nil {
			return err
		}

		sequenceCounters = append(sequenceCounters, sequenceCounter)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSequenceCounterResponse{SequenceCounter: sequenceCounters, Pagination: pageRes}, nil
}

func (k Keeper) SequenceCounter(c context.Context, req *types.QueryGetSequenceCounterRequest) (*types.QueryGetSequenceCounterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetSequenceCounter(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetSequenceCounterResponse{SequenceCounter: val}, nil
}
