package wormhole

import (
	"github.com/certusone/wormhole-chain/x/wormhole/keeper"
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// NewWormholeGovernanceProposalHandler creates a governance handler to manage new proposal types.
// It enables GuardianSetProposal to update the guardian set and GenericWormholeMessageProposal to emit a generic wormhole
// message from the governance emitter.
func NewWormholeGovernanceProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.GuardianSetUpdateProposal:
			return handleGuardianSetUpdateProposal(ctx, k, c)

		case *types.GovernanceWormholeMessageProposal:
			return handleGovernanceWormholeMessageProposal(ctx, k, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized wormhole proposal content type: %T", c)
		}
	}
}

func handleGuardianSetUpdateProposal(ctx sdk.Context, k keeper.Keeper, proposal *types.GuardianSetUpdateProposal) error {

	return nil
}

func handleGovernanceWormholeMessageProposal(ctx sdk.Context, k keeper.Keeper, proposal *types.GovernanceWormholeMessageProposal) error {

	return nil
}
