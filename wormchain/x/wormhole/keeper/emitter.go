package keeper

import (
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func (k Keeper) PostMessage(ctx sdk.Context, emitter types.EmitterAddress, nonce uint32, data []byte) error {
	emitterHex := hex.EncodeToString(emitter.Bytes())
	sequence, found := k.GetSequenceCounter(ctx, emitterHex)
	if !found {
		sequence = types.SequenceCounter{
			Index:    emitterHex,
			Sequence: 0,
		}
	}

	// Retrieve the number of seconds since the unix epoch from the block header
	time := ctx.BlockTime().Unix()

	err := ctx.EventManager().EmitTypedEvent(&types.EventPostedMessage{
		Emitter:  emitter.Bytes(),
		Sequence: sequence.Sequence,
		Nonce:    nonce,
		Time:     uint64(time),
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
