package keeper

import (
	"bytes"
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (k msgServer) AddWasmInstantiateAllowlist(goCtx context.Context, msg *types.MsgAddWasmInstantiateAllowlist) (*types.MsgWasmInstantiateAllowlistResponse, error) {
	return k.ExecuteWasmInstantiateAllowlistAction(goCtx, msg.Vaa, msg.Signer, msg.CodeId, msg.Address, vaa.ActionAddWasmInstantiateAllowlist)
}

func (k msgServer) DeleteWasmInstantiateAllowlist(goCtx context.Context, msg *types.MsgDeleteWasmInstantiateAllowlist) (*types.MsgWasmInstantiateAllowlistResponse, error) {
	return k.ExecuteWasmInstantiateAllowlistAction(goCtx, msg.Vaa, msg.Signer, msg.CodeId, msg.Address, vaa.ActionDeleteWasmInstantiateAllowlist)
}

func (k msgServer) ExecuteWasmInstantiateAllowlistAction(goCtx context.Context, vaaBytes []byte, signer string, codeId uint64, contractAddress string, expectedAction vaa.GovernanceAction) (*types.MsgWasmInstantiateAllowlistResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse VAA
	v, err := ParseVAA(vaaBytes)
	if err != nil {
		return nil, err
	}

	// Verify VAA
	action, payload, err := k.VerifyGovernanceVAA(ctx, v, vaa.WasmdModule)
	if err != nil {
		return nil, err
	}

	// Ensure the governance action is correct
	if vaa.GovernanceAction(action) != expectedAction {
		return nil, types.ErrUnknownGovernanceAction
	}

	// Validate signer
	_, err = sdk.AccAddressFromBech32(signer)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "signer")
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(sdk.AttributeKeySender, signer),
	))

	// verify the cosmos address is correct
	addrBytes, err := sdk.AccAddressFromBech32(contractAddress)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "contract")
	}

	// validate the <contractAddress, codeId> in the VAA payload match the ones in the message
	var payloadBody vaa.BodyWormchainWasmAllowlistInstantiate
	payloadBody.Deserialize(payload)
	if !bytes.Equal(payloadBody.ContractAddr[:], addrBytes) {
		return nil, types.ErrInvalidAllowlistContractAddr
	}

	if payloadBody.CodeId != codeId {
		return nil, types.ErrInvalidAllowlistCodeId
	}

	// add or delete the <contractAddress, codeId> pair
	allowlistEntry := types.WasmInstantiateAllowedContractCodeId{
		ContractAddress: contractAddress,
		CodeId:          codeId,
	}
	if expectedAction == vaa.ActionAddWasmInstantiateAllowlist {
		k.SetWasmInstantiateAllowlist(ctx, allowlistEntry)
	} else if expectedAction == vaa.ActionDeleteWasmInstantiateAllowlist {
		k.KeeperDeleteWasmInstantiateAllowlist(ctx, allowlistEntry)
	} else {
		return nil, types.ErrUnknownGovernanceAction
	}

	return &types.MsgWasmInstantiateAllowlistResponse{}, nil
}
