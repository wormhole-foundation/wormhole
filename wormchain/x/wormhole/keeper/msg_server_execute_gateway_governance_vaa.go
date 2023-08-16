package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (k msgServer) ExecuteGatewayGovernanceVaa(
	goCtx context.Context,
	msg *types.MsgExecuteGatewayGovernanceVaa,
) (*types.EmptyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate signer
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "signer")
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(sdk.AttributeKeySender, msg.Signer),
	))

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

	// Execute action
	switch vaa.GovernanceAction(action) {
	case vaa.ActionScheduleUpgrade:
		return k.scheduleUpgrade(ctx, payload)
	case vaa.ActionCancelUpgrade:
		return k.cancelUpgrade(ctx)
	case vaa.ActionSetIbcComposabilityMwContract:
		return k.setIbcComposabilityMwContract(ctx, payload)
	default:
		return nil, types.ErrUnknownGovernanceAction
	}
}

func (k msgServer) scheduleUpgrade(
	ctx sdk.Context,
	payload []byte,
) (*types.EmptyResponse, error) {
	// Deserialize payload to get the name and height for the upgrade plan
	var payloadBody vaa.BodyGatewayScheduleUpgrade
	payloadBody.Deserialize(payload)

	plan := upgradetypes.Plan{
		Name:   payloadBody.Name,
		Height: int64(payloadBody.Height),
	}
	k.upgradeKeeper.ScheduleUpgrade(ctx, plan)

	return &types.EmptyResponse{}, nil
}

func (k msgServer) cancelUpgrade(ctx sdk.Context) (*types.EmptyResponse, error) {
	k.upgradeKeeper.ClearUpgradePlan(ctx)
	return &types.EmptyResponse{}, nil
}

func (k msgServer) setIbcComposabilityMwContract(
	ctx sdk.Context,
	payload []byte,
) (*types.EmptyResponse, error) {
	// validate the contractAddress in the VAA payload match the ones in the message
	var payloadBody vaa.BodyGatewayIbcComposabilityMwContract
	payloadBody.Deserialize(payload)

	// convert bytes to bech32 address
	contractAddr, err := sdk.Bech32ifyAddressBytes(
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		payloadBody.ContractAddr[:],
	)
	if err != nil {
		return nil, types.ErrInvalidIbcComposabilityMwContractAddr
	}

	newContract := types.IbcComposabilityMwContract{
		ContractAddress: contractAddr,
	}

	k.StoreIbcComposabilityMwContract(ctx, newContract)

	return &types.EmptyResponse{}, nil
}
