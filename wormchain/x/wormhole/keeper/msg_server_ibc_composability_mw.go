package keeper

import (
	"bytes"
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (k msgServer) SetIbcComposabilityMwContract(goCtx context.Context, msg *types.MsgSetIbcComposabilityMwContract) (*types.MsgSetIbcComposabilityMwContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse VAA
	v, err := ParseVAA(msg.Vaa)
	if err != nil {
		return nil, err
	}

	// Verify VAA
	action, payload, err := k.VerifyGovernanceVAA(ctx, v, vaa.GatewayModule)
	if err != nil {
		return nil, err
	}

	// Ensure the governance action is correct
	if vaa.GovernanceAction(action) != vaa.ActionSetIbcComposabilityMwContract {
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
		return nil, sdkerrors.Wrap(err, "ibc composability mw contract")
	}

	// validate the contractAddress in the VAA payload match the ones in the message
	var payloadBody vaa.BodyWormchainIbcComposabilityMwContract
	payloadBody.Deserialize(payload)
	if !bytes.Equal(payloadBody.ContractAddr[:], addrBytes) {
		return nil, types.ErrInvalidIbcComposabilityMwContractAddr
	}

	newContract := types.IbcComposabilityMwContract{
		ContractAddress: msg.Address,
	}

	k.StoreIbcComposabilityMwContract(ctx, newContract)

	return &types.MsgSetIbcComposabilityMwContractResponse{}, nil
}
