package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func (k msgServer) CreateAllowlistEntry(goCtx context.Context, msg *types.MsgCreateAllowlistEntryRequest) (*types.MsgAllowlistResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	validator_address := msg.Signer
	if !k.IsAddressValidatorOrFutureValidator(ctx, validator_address) {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "must be a current or future validator")
	}

	// is this already in an active allowlist?
	if k.HasValidatorAllowedAddress(ctx, msg.Address) {
		allowed := k.GetValidatorAllowedAddress(ctx, msg.Address)
		if k.IsAddressValidatorOrFutureValidator(ctx, allowed.ValidatorAddress) {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "address is already whitelisted")
		}
	}

	k.SetValidatorAllowedAddress(ctx, types.ValidatorAllowedAddress{
		ValidatorAddress: validator_address,
		AllowedAddress:   msg.Address,
		Name:             msg.Name,
	})

	return &types.MsgAllowlistResponse{}, nil
}

func (k msgServer) DeleteAllowlistEntry(goCtx context.Context, msg *types.MsgDeleteAllowlistEntryRequest) (*types.MsgAllowlistResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	validator_address := msg.Signer
	if !k.IsAddressValidatorOrFutureValidator(ctx, validator_address) {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "must be a current or future validator")
	}

	// is this already in an active allowlist?
	if k.HasValidatorAllowedAddress(ctx, msg.Address) {
		allowed := k.GetValidatorAllowedAddress(ctx, msg.Address)
		if !k.IsAddressValidatorOrFutureValidator(ctx, allowed.ValidatorAddress) {
			// permit deleting entries of past validators
		} else if allowed.ValidatorAddress != validator_address {
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "can only delete allowlist entries you created")
		}
	} else {
		return nil, sdkerrors.ErrKeyNotFound
	}

	k.RemoveValidatorAllowedAddress(ctx, msg.Address)

	return &types.MsgAllowlistResponse{}, nil
}
