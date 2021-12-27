package keeper

import (
	"encoding/hex"
	"fmt"
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) PostMessage(ctx sdk.Context, emitter []byte, nonce uint32, data []byte) error {
	if len(emitter) != 32 {
		return fmt.Errorf("emitter must be 32 bytes long, was %d", len(emitter))
	}
	emitterHex := hex.EncodeToString(emitter)
	sequence, found := k.GetSequenceCounter(ctx, emitterHex)
	if !found {
		sequence = types.SequenceCounter{
			Index:    emitterHex,
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
