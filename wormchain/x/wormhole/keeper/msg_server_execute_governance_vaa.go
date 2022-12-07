package keeper

import (
	"context"
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (k msgServer) ExecuteGovernanceVAA(goCtx context.Context, msg *types.MsgExecuteGovernanceVAA) (*types.MsgExecuteGovernanceVAAResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse VAA
	v, err := ParseVAA(msg.Vaa)
	if err != nil {
		return nil, err
	}

	coreModule := [32]byte{}
	copy(coreModule[:], vaa.CoreModule)
	// Verify VAA
	action, payload, err := k.VerifyGovernanceVAA(ctx, v, coreModule)
	if err != nil {
		return nil, err
	}

	// Execute action
	switch vaa.GovernanceAction(action) {
	case vaa.ActionGuardianSetUpdate:
		if len(payload) < 5 {
			return nil, types.ErrInvalidGovernancePayloadLength
		}
		// Update guardian set
		newIndex := binary.BigEndian.Uint32(payload[:4])
		numGuardians := int(payload[4])

		if len(payload) != 5+20*numGuardians {
			return nil, types.ErrInvalidGovernancePayloadLength
		}

		added := make(map[string]bool)
		var keys [][]byte
		for i := 0; i < numGuardians; i++ {
			k := payload[5+i*20 : 5+i*20+20]
			sk := string(k)
			if _, found := added[sk]; found {
				return nil, types.ErrDuplicateGuardianAddress
			}
			keys = append(keys, k)
			added[sk] = true
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

	return &types.MsgExecuteGovernanceVAAResponse{}, nil
}
