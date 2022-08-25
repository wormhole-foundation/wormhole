package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/tokenbridge module sentinel errors
var (
	ErrUnknownGovernanceModule        = sdkerrors.Register(ModuleName, 1105, "invalid governance module")
	ErrInvalidGovernanceEmitter       = sdkerrors.Register(ModuleName, 1107, "invalid governance emitter")
	ErrUnknownGovernanceAction        = sdkerrors.Register(ModuleName, 1108, "unknown governance action")
	ErrGovernanceHeaderTooShort       = sdkerrors.Register(ModuleName, 1109, "governance header too short")
	ErrInvalidGovernanceTargetChain   = sdkerrors.Register(ModuleName, 1110, "governance target chain does not match")
	ErrInvalidGovernancePayloadLength = sdkerrors.Register(ModuleName, 1111, "governance payload has incorrect length")
	ErrVAAAlreadyExecuted             = sdkerrors.Register(ModuleName, 1113, "VAA was already executed")
	ErrChainAlreadyRegistered         = sdkerrors.Register(ModuleName, 1114, "Chain already registered")
	ErrUnregisteredEmitter            = sdkerrors.Register(ModuleName, 1115, "emitter is not registered")
	ErrVAAPayloadInvalid              = sdkerrors.Register(ModuleName, 1116, "invalid VAA payload")
	ErrAssetNotRegistered             = sdkerrors.Register(ModuleName, 1117, "asset not registered")
	ErrInvalidTargetChain             = sdkerrors.Register(ModuleName, 1118, "invalid target chain on transfer")
	ErrUnknownPayloadType             = sdkerrors.Register(ModuleName, 1119, "unknown payload type")
	ErrNativeAssetRegistration        = sdkerrors.Register(ModuleName, 1120, "cannot register native asset")
	ErrNoDenomMetadata                = sdkerrors.Register(ModuleName, 1121, "denom does not have metadata")
	ErrAttestWormholeToken            = sdkerrors.Register(ModuleName, 1122, "cannot attest wormhole wrapped asset")
	ErrNameTooLong                    = sdkerrors.Register(ModuleName, 1124, "name too long for attestation")
	ErrSymbolTooLong                  = sdkerrors.Register(ModuleName, 1125, "symbol too long for attestation")
	ErrDisplayUnitNotFound            = sdkerrors.Register(ModuleName, 1126, "display denom unit not found")
	ErrExponentTooLarge               = sdkerrors.Register(ModuleName, 1127, "exponent of display unit must be uint8")
	ErrInvalidToAddress               = sdkerrors.Register(ModuleName, 1128, "to address is invalid (must be 32 bytes)")
	ErrInvalidFee                     = sdkerrors.Register(ModuleName, 1130, "fee is invalid (must fit in uint256)")
	ErrInvalidAmount                  = sdkerrors.Register(ModuleName, 1131, "amount is invalid (must fit in uint256)")
	ErrFeeTooHigh                     = sdkerrors.Register(ModuleName, 1132, "fee must be < amount")
	ErrAmountTooHigh                  = sdkerrors.Register(ModuleName, 1133, "the amount would exceed the bridges capacity of u64")
	ErrAssetMetaRollback              = sdkerrors.Register(ModuleName, 1134, "asset meta must have a higher sequence than the last update")
	ErrNegativeFee                    = sdkerrors.Register(ModuleName, 1135, "fee cannot be negative")
	ErrRegisterWormholeChain          = sdkerrors.Register(ModuleName, 1136, "cannot register an emitter for wormhole-chain on wormhole-chain")
	ErrChangeDecimals                 = sdkerrors.Register(ModuleName, 1137, "cannot change decimals of registered asset metadata")
)
