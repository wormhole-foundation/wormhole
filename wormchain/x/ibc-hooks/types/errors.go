package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrBadMetadataFormatMsg = "wasm metadata not properly formatted for: '%v'. %s"
	ErrBadExecutionMsg      = "cannot execute contract: %v"

	ErrMsgValidation = sdkerrors.Register("wasm-hooks", 2, "error in wasmhook message validation")
	ErrMarshaling    = sdkerrors.Register("wasm-hooks", 3, "cannot marshal the ICS20 packet")
	ErrInvalidPacket = sdkerrors.Register("wasm-hooks", 4, "invalid packet data")
	ErrBadResponse   = sdkerrors.Register("wasm-hooks", 5, "cannot create response")
	ErrWasmError     = sdkerrors.Register("wasm-hooks", 6, "wasm error")
	ErrBadSender     = sdkerrors.Register("wasm-hooks", 7, "bad sender")
)
