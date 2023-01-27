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

	verified_addresses := make(map[string]bool)

	// For every message we should check:
	// 1. There is an allowlist entry for the signer(s), OR
	// 2. The signer(s) are validators in current or future guardian set.
	// We cache the result for performance, since this is a ante handler that runs for everything.
	for _, msg := range tx.GetMsgs() {
		for _, signer := range msg.GetSigners() {
			addr := signer.String()
			// check for an address we may have already verified
			if ok := verified_addresses[addr]; ok {
				// ok
				continue
			}
			// check for an allowlist
			if wh.k.HasValidatorAllowedAddress(request, addr) {
				allowed_entry := wh.k.GetValidatorAllowedAddress(request, addr)
				// authenticate that the validator that made the allowlist is still valid
				if wh.k.IsAddressValidatorOrFutureValidator(request, allowed_entry.ValidatorAddress) {
					// ok
					verified_addresses[addr] = true
					continue
				}
			}

			// if allowlist did not pass, check if signer is a current or future validator
			if wh.k.IsAddressValidatorOrFutureValidator(request, addr) {
				// ok
				verified_addresses[addr] = true
				continue
			}

			// by this point, this signer is not allowlisted or a valid validator.
			// not authorized!
			return request, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "signer must be current validator or allowlisted by current validator")
		}
	}

	return next(request, tx, simulate)
}
