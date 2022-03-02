package keeper

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/certusone/wormhole-chain/x/wormhole/keeper"
	"github.com/certusone/wormhole/node/pkg/vaa"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	whtypes "github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type GovernanceAction uint8

var (
	ActionRegisterChain GovernanceAction = 1
)

func (k msgServer) ExecuteGovernanceVAA(goCtx context.Context, msg *types.MsgExecuteGovernanceVAA) (*types.MsgExecuteGovernanceVAAResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse VAA
	v, err := keeper.ParseVAA(msg.Vaa)
	if err != nil {
		return nil, err
	}

	// Verify VAA
	err = k.wormholeKeeper.VerifyVAA(ctx, v)
	if err != nil {
		return nil, err
	}

	wormholeConfig, ok := k.wormholeKeeper.GetConfig(ctx)
	if !ok {
		return nil, whtypes.ErrNoConfig
	}

	_, known := k.GetReplayProtection(ctx, v.HexDigest())
	if known {
		return nil, types.ErrVAAAlreadyExecuted
	}

	// Check governance emitter
	if !bytes.Equal(v.EmitterAddress[:], wormholeConfig.GovernanceEmitter) {
		return nil, types.ErrInvalidGovernanceEmitter
	}
	if v.EmitterChain != vaa.ChainID(wormholeConfig.GovernanceChain) {
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

	if chain != 0 && chain != uint16(wormholeConfig.ChainId) {
		return nil, types.ErrInvalidGovernanceTargetChain
	}

	// Execute action
	switch action {
	case ActionRegisterChain:
		if len(payload) != 34 {
			return nil, types.ErrInvalidGovernancePayloadLength
		}
		// Add chain registration
		chainId := binary.BigEndian.Uint16(payload[:2])
		bridgeEmitter := payload[2:32]

		if _, found := k.GetChainRegistration(ctx, uint32(chainId)); found {
			return nil, types.ErrChainAlreadyRegistered
		}

		k.SetChainRegistration(ctx, types.ChainRegistration{
			ChainID:        uint32(chainId),
			EmitterAddress: bridgeEmitter,
		})

		// Emit event
		err = ctx.EventManager().EmitTypedEvent(&types.EventChainRegistered{
			ChainID:        uint32(chainId),
			EmitterAddress: bridgeEmitter,
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
