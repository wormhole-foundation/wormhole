package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// GuardianSetUpdateProposal updates the guardian set
func (k msgServer) GuardianSetUpdateProposal(goCtx context.Context, proposal *types.MsgGuardianSetUpdateProposal) (res *types.EmptyResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate authority of the proposal
	if k.authority != proposal.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, proposal.Authority)
	}

	err = k.UpdateGuardianSet(ctx, types.GuardianSet{
		Index:          proposal.NewGuardianSet.Index,
		Keys:           proposal.NewGuardianSet.Keys,
		ExpirationTime: 0,
	})

	if err != nil {
		return res, fmt.Errorf("failed to update guardian set: %w", err)
	}

	config, ok := k.GetConfig(ctx)
	if !ok {
		return res, types.ErrNoConfig
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
		return res, fmt.Errorf("failed to post message: %w", err)
	}

	err = k.PostMessage(ctx, emitterAddress, 0, message.Bytes())
	if err != nil {
		return res, fmt.Errorf("failed to post message: %w", err)
	}

	return res, nil
}
