package ante

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	wormholekeeper "github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
)

// Reject all wasmd message unless authorized
type WormholeWasmdDecorator struct {
	k           wormholekeeper.Keeper
	wasmdKeeper wasmkeeper.Keeper
}

// NewWormholeWasmdDecorator creates a new WormholeWasmdDecorator
func NewWormholeWasmdDecorator(k wormholekeeper.Keeper, wasmd wasmkeeper.Keeper) WormholeWasmdDecorator {
	return WormholeWasmdDecorator{
		k:           k,
		wasmdKeeper: wasmd,
	}
}

// AnteHandle implements the AnteHandler interface
//
// This handler rejects all wasmd messages except instantiate for allowed senders.
func (wh WormholeWasmdDecorator) AnteHandle(request sdk.Request, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Request, err error) {
	// if this is a recheck, we already know the tx is valid
	if request.IsReCheckTx() {
		return next(request, tx, simulate)
	}

	// don't reject gen_tx transactions
	if request.BlockHeight() < 1 {
		return next(request, tx, simulate)
	}

	// reject all wasmd messages except instantiate for allowed senders
	for _, msg := range tx.GetMsgs() {

		switch wasmMsg := msg.(type) {

		case *wasmtypes.MsgInstantiateContract:
			wasmMsg, _ = msg.(*wasmtypes.MsgInstantiateContract)
			if !wh.k.HasWasmInstantiateAllowlist(request, wasmMsg.Sender, wasmMsg.CodeID) {
				return request, errNotSupported()
			} else {
				continue
			}

		case *wasmtypes.MsgInstantiateContract2:
			wasmMsg, _ = msg.(*wasmtypes.MsgInstantiateContract2)
			if !wh.k.HasWasmInstantiateAllowlist(request, wasmMsg.Sender, wasmMsg.CodeID) {
				return request, errNotSupported()
			} else {
				continue
			}

		case *wasmtypes.MsgStoreCode,
			*wasmtypes.MsgMigrateContract,
			*wasmtypes.MsgUpdateAdmin,
			*wasmtypes.MsgClearAdmin,
			*wasmtypes.MsgUpdateInstantiateConfig,
			*wasmtypes.MsgUpdateParams,
			*wasmtypes.MsgPinCodes,
			*wasmtypes.MsgUnpinCodes,
			*wasmtypes.MsgSudoContract, // TODO: JOEL - May not be necessary as only executable by the chain
			*wasmtypes.MsgStoreAndInstantiateContract,
			*wasmtypes.MsgAddCodeUploadParamsAddresses,
			*wasmtypes.MsgRemoveCodeUploadParamsAddresses,
			*wasmtypes.MsgStoreAndMigrateContract,
			*wasmtypes.MsgUpdateContractLabel:
			return request, errNotSupported()
		}
	}

	// continue to next AnteHandler
	return next(request, tx, simulate)
}

// errNotSupported returns an error indicating the message type is not supported.
func errNotSupported() error {
	return sdkerrors.Wrapf(sdkerrors.ErrNotSupported, "must use x/wormhole")
}
