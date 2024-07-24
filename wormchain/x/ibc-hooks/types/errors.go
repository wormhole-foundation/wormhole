package types

import errors "cosmossdk.io/errors"

var (
	ErrBadMetadataFormatMsg = "wasm metadata not properly formatted for: '%v'. %s"
	ErrBadExecutionMsg      = "cannot execute contract: %v"

	ErrMsgValidation = errors.Register(ModuleName, 2, "error in wasmhook message validation")
	ErrMarshaling    = errors.Register("wasm-hooks", 3, "cannot marshal the ICS20 packet")
	ErrInvalidPacket = errors.Register("wasm-hooks", 4, "invalid packet data")
	ErrBadResponse   = errors.Register("wasm-hooks", 5, "cannot create response")
	ErrWasmError     = errors.Register("wasm-hooks", 6, "wasm error")
	ErrBadSender     = errors.Register("wasm-hooks", 7, "bad sender")
)
