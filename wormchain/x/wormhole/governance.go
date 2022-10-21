package wormhole

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
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
	err := k.UpdateGuardianSet(ctx, types.GuardianSet{
		Index:          proposal.NewGuardianSet.Index,
		Keys:           proposal.NewGuardianSet.Keys,
		ExpirationTime: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to update guardian set: %w", err)
	}

	config, ok := k.GetConfig(ctx)
	if !ok {
		return types.ErrNoConfig
	}

	// Post a wormhole guardian set update governance message
	message := &bytes.Buffer{}

	// Header
	message.Write(vaa.CoreModule)
	MustWrite(message, binary.BigEndian, uint8(2))
	MustWrite(message, binary.BigEndian, uint16(0))

	// Body
	MustWrite(message, binary.BigEndian, proposal.NewGuardianSet.Index)
	MustWrite(message, binary.BigEndian, uint8(len(proposal.NewGuardianSet.Keys)))
	for _, key := range proposal.NewGuardianSet.Keys {
		message.Write(key)
	}

	emitterAddress, err := types.EmitterAddressFromBytes32(config.GovernanceEmitter)
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	err = k.PostMessage(ctx, emitterAddress, 0, message.Bytes())
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	return nil
}

func handleGovernanceWormholeMessageProposal(ctx sdk.Context, k keeper.Keeper, proposal *types.GovernanceWormholeMessageProposal) error {
	config, ok := k.GetConfig(ctx)
	if !ok {
		return types.ErrNoConfig
	}

	// Post a wormhole governance message
	message := &bytes.Buffer{}
	message.Write(proposal.Module)
	MustWrite(message, binary.BigEndian, uint8(proposal.Action))
	MustWrite(message, binary.BigEndian, uint16(proposal.TargetChain))
	message.Write(proposal.Payload)

	emitterAddress, err := types.EmitterAddressFromBytes32(config.GovernanceEmitter)
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	err = k.PostMessage(ctx, emitterAddress, 0, message.Bytes())
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	return nil
}

// MustWrite calls binary.Write and panics on errors
func MustWrite(w io.Writer, order binary.ByteOrder, data interface{}) {
	if err := binary.Write(w, order, data); err != nil {
		panic(fmt.Errorf("failed to write binary data: %v", data).Error())
	}
}
