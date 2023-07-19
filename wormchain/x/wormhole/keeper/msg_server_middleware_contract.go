package keeper

import (
	"bytes"
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (k msgServer) SetWormholeMiddlewareContract(goCtx context.Context, msg *types.MsgSetWormholeMiddlewareContract) (*types.MsgSetWormholeMiddlewareContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse VAA
	v, err := ParseVAA(msg.Vaa)
	if err != nil {
		return nil, err
	}

	// Verify VAA
	action, payload, err := k.VerifyGovernanceVAA(ctx, v, vaa.WasmdModule)
	if err != nil {
		return nil, err
	}

	// Ensure the governance action is correct
	if vaa.GovernanceAction(action) != vaa.ActionSetWormholeMiddlewareContract {
		return nil, types.ErrUnknownGovernanceAction
	}

	// Validate signer
	_, err = sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "signer")
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(sdk.AttributeKeySender, msg.Signer),
	))

	// verify the cosmos address is correct
	addrBytes, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "middleware contract")
	}

	// validate the contractAddress in the VAA payload match the ones in the message
	var payloadBody vaa.BodyWormchainMiddlewareContract
	payloadBody.Deserialize(payload)
	if !bytes.Equal(payloadBody.ContractAddr[:], addrBytes) {
		return nil, types.ErrInvalidMiddlewareContractAddr
	}

	newContract := types.WormholeMiddlewareContract{
		ContractAddress: msg.Address,
	}
	
	k.StoreMiddlewareContract(ctx, newContract)

	return &types.MsgSetWormholeMiddlewareContractResponse{}, nil
}
