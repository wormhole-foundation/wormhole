package keeper

import (
	"context"
	"encoding/binary"

	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormhole-chain/x/tokenbridge/types"
	whtypes "github.com/wormhole-foundation/wormhole-chain/x/wormhole/types"
)

type GovernanceAction uint8

// TokenBridgeModule is the identifier of the TokenBridge module (which is used for governance messages)
// TODO(csongor): where's the best place to put this? CoreModule is in the node code, why is TokenBridgeModule not?
var TokenBridgeModule = [32]byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x42, 0x72, 0x69, 0x64, 0x67, 0x65}

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
	action, payload, err := k.wormholeKeeper.VerifyGovernanceVAA(ctx, v, TokenBridgeModule)
	if err != nil {
		return nil, err
	}

	wormholeConfig, ok := k.wormholeKeeper.GetConfig(ctx)
	if !ok {
		return nil, whtypes.ErrNoConfig
	}

	// Execute action
	switch GovernanceAction(action) {
	case ActionRegisterChain:
		if len(payload) != 34 {
			return nil, types.ErrInvalidGovernancePayloadLength
		}
		// Add chain registration
		chainId := binary.BigEndian.Uint16(payload[:2])
		bridgeEmitter := payload[2:34]

		if chainId == uint16(wormholeConfig.ChainId) {
			return nil, types.ErrRegisterWormholeChain
		}

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

	return &types.MsgExecuteGovernanceVAAResponse{}, nil
}
