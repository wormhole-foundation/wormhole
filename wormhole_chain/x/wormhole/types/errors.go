package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/wormhole module sentinel errors
var (
	ErrSample                         = sdkerrors.Register(ModuleName, 1100, "sample error")
	ErrGuardianSetNotFound            = sdkerrors.Register(ModuleName, 1101, "guardian set not found")
	ErrSignaturesInvalid              = sdkerrors.Register(ModuleName, 1102, "invalid signatures on VAA")
	ErrNoQuorum                       = sdkerrors.Register(ModuleName, 1103, "no quorum on VAA")
	ErrVAAUnmarshal                   = sdkerrors.Register(ModuleName, 1104, "error decoding vaa")
	ErrUnknownGovernanceModule        = sdkerrors.Register(ModuleName, 1105, "invalid governance module")
	ErrNoConfig                       = sdkerrors.Register(ModuleName, 1106, "config not set")
	ErrInvalidGovernanceEmitter       = sdkerrors.Register(ModuleName, 1107, "invalid governance emitter")
	ErrUnknownGovernanceAction        = sdkerrors.Register(ModuleName, 1108, "unknown governance action")
	ErrGovernanceHeaderTooShort       = sdkerrors.Register(ModuleName, 1109, "governance header too short")
	ErrInvalidGovernanceTargetChain   = sdkerrors.Register(ModuleName, 1110, "governance target chain does not match")
	ErrInvalidGovernancePayloadLength = sdkerrors.Register(ModuleName, 1111, "governance payload has incorrect length")
	ErrGuardianSetNotSequential       = sdkerrors.Register(ModuleName, 1112, "guardian set updates must be submitted sequentially")
	ErrVAAAlreadyExecuted             = sdkerrors.Register(ModuleName, 1113, "VAA was already executed")
	ErrGuardianSignatureMismatch      = sdkerrors.Register(ModuleName, 1114, "guardian signature mismatch")
	ErrSignerMismatch                 = sdkerrors.Register(ModuleName, 1115, "transaction signer doesn't match validator key")
	ErrGuardianNotFound               = sdkerrors.Register(ModuleName, 1116, "guardian not found in guardian set")
	ErrConsensusSetUndefined          = sdkerrors.Register(ModuleName, 1117, "no consensus set defined")
	ErrGuardianSetExpired             = sdkerrors.Register(ModuleName, 1118, "guardian set expired")
	ErrNewGuardianSetHasExpiry        = sdkerrors.Register(ModuleName, 1119, "new guardian set should not have expiry time")
)
