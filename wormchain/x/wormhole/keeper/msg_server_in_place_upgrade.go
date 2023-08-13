package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	pfmtypes "github.com/strangelove-ventures/packet-forward-middleware/v4/router/types"
	tokenfactorytypes "github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (k msgServer) InPlaceUpgrade(
	goCtx context.Context,
	msg *types.MsgInPlaceUpgrade,
) (*types.EmptyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.SetTokenfactoryPfmDefaultParams != nil {
		return k.setTokenfactoryPfmDefaultParams(ctx, msg.Signer, msg.SetTokenfactoryPfmDefaultParams)
	}

	return nil, sdkerrors.Wrapf(sdkerrors.ErrNotSupported, "InPlaceUpgrade message type not supported")
}

func (k msgServer) setTokenfactoryPfmDefaultParams(
	ctx sdk.Context,
	signer string,
	msg *types.MsgSetTokenFactoryPfmDefaultParams,
) (*types.EmptyResponse, error) {
	if !k.setTokenfactory || !k.setPfm {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrNotSupported, "either x/tokenfactory or PFM keeper not set")
	}

	// Parse VAA
	v, err := ParseVAA(msg.Vaa)
	if err != nil {
		return nil, err
	}

	// Verify VAA
	action, _, err := k.VerifyGovernanceVAA(ctx, v, vaa.GatewayModule)
	if err != nil {
		return nil, err
	}

	// Ensure the governance action is correct
	if vaa.GovernanceAction(action) != vaa.ActionSetTokenfactoryPfmDefaultParams {
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

	// Set the default params for both tokenfactory and PFM
	k.tokenfactoryKeeper.SetParams(ctx, tokenfactorytypes.DefaultParams())
	k.pfmKeeper.SetParams(ctx, pfmtypes.DefaultParams())

	return &types.EmptyResponse{}, nil
}
