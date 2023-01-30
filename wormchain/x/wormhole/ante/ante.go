package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
)

// Reject all messages if we're expecting a software update.
type WormholeAllowlistDecorator struct {
	k keeper.Keeper
}

func NewWormholeAllowlistDecorator(k keeper.Keeper) WormholeAllowlistDecorator {
	return WormholeAllowlistDecorator{
		k: k,
	}
}

func (wh WormholeAllowlistDecorator) AnteHandle(request sdk.Request, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Request, err error) {
	if request.IsReCheckTx() {
		return next(request, tx, simulate)
	}
	if request.BlockHeight() < 1 {
		// Don't reject gen_tx transactions
		return next(request, tx, simulate)
	}

	// We permit if there is a message with a signer satisfying either condition:
	// 1. There is an allowlist entry for the signer(s), OR
	// 2. The signer is a validators in a current or future guardian set.
	// I.e. If one message has an allowed signer, then the transaction has a signature from that address.
	for _, msg := range tx.GetMsgs() {
		for _, signer := range msg.GetSigners() {
			addr := signer.String()
			// check for an allowlist
			if wh.k.HasValidatorAllowedAddress(request, addr) {
				allowed_entry := wh.k.GetValidatorAllowedAddress(request, addr)
				// authenticate that the validator that made the allowlist is still valid
				if wh.k.IsAddressValidatorOrFutureValidator(request, allowed_entry.ValidatorAddress) {
					// ok
					return next(request, tx, simulate)
				}
			}

			// if allowlist did not pass, check if signer is a current or future validator
			if wh.k.IsAddressValidatorOrFutureValidator(request, addr) {
				// ok
				return next(request, tx, simulate)
			}
		}
	}

	// By this point, there is no signer that is not allowlisted or is a validator.
	// Not authorized!
	return request, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "signer must be current validator or allowlisted by current validator")
}
