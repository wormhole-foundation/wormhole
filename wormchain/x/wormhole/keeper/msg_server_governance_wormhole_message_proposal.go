package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func (k msgServer) GovernanceWormholeMessageProposal(goCtx context.Context, proposal *types.MsgGovernanceWormholeMessageProposal) (res *types.EmptyResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate authority of the proposal
	if k.authority != proposal.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, proposal.Authority)
	}

	config, ok := k.GetConfig(ctx)
	if !ok {
		return res, types.ErrNoConfig
	}

	// Post a wormhole governance message
	message := &bytes.Buffer{}
	message.Write(proposal.Module)
	MustWrite(message, binary.BigEndian, uint8(proposal.Action))
	MustWrite(message, binary.BigEndian, uint16(proposal.TargetChain))
	message.Write(proposal.Payload)

	emitterAddress, err := types.EmitterAddressFromBytes32(config.GovernanceEmitter)
	if err != nil {
		return res, fmt.Errorf("failed to post message: %w", err)
	}

	err = k.PostMessage(ctx, emitterAddress, 0, message.Bytes())
	if err != nil {
		return res, fmt.Errorf("failed to post message: %w", err)
	}

	return res, nil
}

// MustWrite calls binary.Write and panics on errors
func MustWrite(w io.Writer, order binary.ByteOrder, data interface{}) {
	if err := binary.Write(w, order, data); err != nil {
		panic(fmt.Errorf("failed to write binary data: %v", data).Error())
	}
}
