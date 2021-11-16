package keeper

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/certusone/wormhole/node/pkg/vaa"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type GovernanceAction uint8

var (
	ActionGuardianSetUpdate GovernanceAction = 2
)

func (k msgServer) ExecuteGovernanceVAA(goCtx context.Context, msg *types.MsgExecuteGovernanceVAA) (*types.MsgExecuteGovernanceVAAResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse VAA
	v, err := ParseVAA(msg.Vaa)
	if err != nil {
		return nil, err
	}

	// Verify VAA
	err = k.VerifyVAA(ctx, v)
	if err != nil {
		return nil, err
	}

	config, ok := k.GetConfig(ctx)
	if !ok {
		return nil, types.ErrNoConfig
	}

	_, known := k.GetReplayProtection(ctx, v.HexDigest())
	if known {
		return nil, types.ErrVAAAlreadyExecuted
	}

	// Check governance emitter
	if !bytes.Equal(v.EmitterAddress[:], config.GovernanceEmitter) {
		return nil, types.ErrInvalidGovernanceEmitter
	}
	if v.EmitterChain != vaa.ChainID(config.GovernanceChain) {
		return nil, types.ErrInvalidGovernanceEmitter
	}

	if len(v.Payload) < 35 {
		return nil, types.ErrGovernanceHeaderTooShort
	}

	// Check governance header
	if !bytes.Equal(v.Payload[:32], vaa.CoreModule) {
		return nil, types.ErrUnknownGovernanceModule
	}

	// Decode header
	action := GovernanceAction(v.Payload[32])
	chain := binary.BigEndian.Uint16(v.Payload[33:35])
	payload := v.Payload[35:]

	if chain != 0 && chain != uint16(config.ChainId) {
		return nil, types.ErrInvalidGovernanceTargetChain
	}

	// Execute action
	switch action {
	case ActionGuardianSetUpdate:
		if len(payload) < 3 {
			return nil, types.ErrInvalidGovernancePayloadLength
		}
		// Update guardian set
		newIndex := binary.BigEndian.Uint32(payload[:4])
		numGuardians := int(payload[4])

		if len(payload) != 5+20*numGuardians {
			return nil, types.ErrInvalidGovernancePayloadLength
		}
		var keys [][]byte
		for i := 0; i < numGuardians; i++ {
			keys = append(keys, payload[5+i*20:5+i*20+20])
		}

		err := k.UpdateGuardianSet(ctx, types.GuardianSet{
			Keys:  keys,
			Index: newIndex,
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, types.ErrUnknownGovernanceAction

	}

	// Prevent replay
	k.SetReplayProtection(ctx, types.ReplayProtection{Index: v.HexDigest()})

	return &types.MsgExecuteGovernanceVAAResponse{}, nil
}
