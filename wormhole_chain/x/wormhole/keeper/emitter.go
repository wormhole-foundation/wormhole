package keeper

import (
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) PostMessage(ctx sdk.Context, emitter sdk.AccAddress, nonce uint32, data []byte) error {
	sequence, found := k.GetSequenceCounter(ctx, emitter.String())
	if !found {
		sequence = types.SequenceCounter{
			Index:    emitter.String(),
			Sequence: 0,
		}
	}

	err := ctx.EventManager().EmitTypedEvent(&types.EventPostedMessage{
		Emitter:  emitter,
		Sequence: sequence.Sequence,
		Nonce:    nonce,
		Payload:  data,
	})
	if err != nil {
		panic(err)
	}

	// Increment sequence counter
	sequence.Sequence++
	k.SetSequenceCounter(ctx, sequence)

	return nil
}
