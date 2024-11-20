package keeper

import (
	"context"
	"encoding/binary"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
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
	case vaa.ActionSlashingParamsUpdate:
		if len(payload) != 40 {
			return nil, types.ErrInvalidGovernancePayloadLength
		}

		// Extract params from payload
		signedBlocksWindow := int64(binary.BigEndian.Uint64(payload[:8]))
		minSignedPerWindow := int64(binary.BigEndian.Uint64(payload[8:16]))
		downtimeJailDuration := int64(binary.BigEndian.Uint64(payload[16:24]))
		slashFractionDoubleSign := int64(binary.BigEndian.Uint64(payload[24:32]))
		slashFractionDowntime := int64(binary.BigEndian.Uint64(payload[32:40]))

		// Update slashing params
		params := slashingtypes.NewParams(
			signedBlocksWindow,
			sdk.NewDecWithPrec(minSignedPerWindow, 18),
			time.Duration(downtimeJailDuration),
			sdk.NewDecWithPrec(slashFractionDoubleSign, 18),
			sdk.NewDecWithPrec(slashFractionDowntime, 18),
		)

		// Set the new params
		err := k.slashingKeeper.SetParams(ctx, params)
		if err != nil {
			return nil, err
		}
	default:
		return nil, types.ErrUnknownGovernanceAction

	}

	return &types.MsgExecuteGovernanceVAAResponse{}, nil
}
