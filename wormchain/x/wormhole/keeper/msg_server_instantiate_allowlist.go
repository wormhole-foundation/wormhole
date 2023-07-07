package keeper

import (
	"bytes"
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (k msgServer) AddInstantiateAllowlist(goCtx context.Context, msg *types.MsgAddInstiatiateAllowlist) (*types.MsgAddInstiatiateAllowlistResponse, error) {
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
	if vaa.GovernanceAction(action) != vaa.ActionAllowlistInstantiateContract {
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
		return nil, sdkerrors.Wrap(err, "contract")
	}

	// validate the <contractAddress, codeId> in the VAA payload match the ones in the message
	var payloadBody *vaa.BodyWormchainAllowlistInstantiateContract
	payloadBody.Deserialize(payload)
	if !bytes.Equal(payloadBody.ContractAddr[:], addrBytes) {
		return nil, types.ErrInvalidAllowlistContractAddr
	}

	if payloadBody.CodeId != msg.CodeId {
		return nil, types.ErrInvalidAllowlistCodeId
	}

	// add the <contractAddress, codeId> pair to storage
	k.SetInstantiateAllowlist(ctx, types.WasmAllowedContractCodeId{
		ContractAddress: msg.Address,
		CodeId:          msg.CodeId,
	})

	return &types.MsgAddInstiatiateAllowlistResponse{}, nil
}
