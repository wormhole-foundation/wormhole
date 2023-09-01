use anchor_lang::prelude::error_code;

#[error_code]
/// Errors relevant to Token Bridge's malfunction.
///
/// >= 0x0    -- General program related.
/// >= 0x10   -- General Token Bridge.
/// >= 0x20   -- General Token Bridge Governance.
/// >= 0x30   -- General SPL Handling.
/// >= 0x40   -- Inbound Token Bridge Transfers.
/// >= 0x60   -- Outbound Token Bridge Transfers.
/// >= 0x100  -- Legacy Attest Token.
/// >= 0x200  -- Legacy Complete Transfer Native.
/// >= 0x300  -- Legacy Complete Transfer Wrapped.
/// >= 0x400  -- Legacy Transfer Tokens Wrapped.
/// >= 0x500  -- Legacy Transfer Tokens Native.
/// >= 0x600  -- Legacy Register Chain.
/// >= 0x700  -- Legacy Create Or Update Wrapped.
/// >= 0x800  -- Legacy Upgrade Contract.
/// >= 0x900  -- Legacy Complete Transfer with Payload Native.
/// >= 0x1000 -- Legacy Complete Transfer with Payload Wrapped.
/// >= 0x1100 -- Legacy Transfer Tokens with Payload Wrapped.
/// >= 0x1200 -- Legacy Transfer Tokens with Payload Native.
/// >= 0x2000 -- Token Bridge Anchor Instruction.
///
/// NOTE: All of these error codes when triggered are offset by `ERROR_CODE_OFFSET` (6000). So for
/// example, `U64Overflow` will return as 6006.
pub enum TokenBridgeError {
    #[msg("CannotParseMessage")]
    CannotParseMessage = 0x02,

    #[msg("U64Overflow")]
    U64Overflow = 0x06,

    #[msg("InvalidTokenBridgeVaa")]
    InvalidTokenBridgeVaa = 0x10,

    #[msg("InvalidTokenBridgePayload")]
    InvalidTokenBridgePayload = 0x12,

    #[msg("InvalidTokenBridgeEmitter")]
    InvalidTokenBridgeEmitter = 0x14,

    #[msg("InvalidLegacyTokenBridgeEmitter")]
    InvalidLegacyTokenBridgeEmitter = 0x15,

    #[msg("CoreFeeCollectorRequired")]
    CoreFeeCollectorRequired = 0x16,

    #[msg("InvalidGovernanceEmitter")]
    InvalidGovernanceEmitter = 0x20,

    #[msg("InvalidGovernanceAction")]
    InvalidGovernanceAction = 0x22,

    #[msg("GovernanceForAnotherChain")]
    GovernanceForAnotherChain = 0x24,

    #[msg("InvalidGovernanceVaa")]
    InvalidGovernanceVaa = 0x26,

    #[msg("InvalidMint")]
    InvalidMint = 0x30,

    #[msg("NativeAsset")]
    NativeAsset = 0x32,

    #[msg("WrappedAsset")]
    WrappedAsset = 0x34,

    #[msg("RecipientChainNotSolana")]
    RecipientChainNotSolana = 0x40,

    #[msg("RedeemerChainNotSolana")]
    RedeemerChainNotSolana = 0x42,

    #[msg("InvalidProgramRedeemer")]
    InvalidProgramRedeemer = 0x44,

    #[msg("NestedTokenAccount")]
    NestedTokenAccount = 0x46,

    #[msg("InvalidRecipient")]
    InvalidRecipient = 0x48,

    #[msg("CannotSerializeJson")]
    CannotSerializeJson = 0x700,

    #[msg("InvalidRelayerFee")]
    InvalidRelayerFee = 0x60,

    #[msg("ImplementationMismatch")]
    ImplementationMismatch = 0x800,

    #[msg("UnsupportedInstructionDirective")]
    UnsupportedInstructionDirective = 0x2000,

    #[msg("EmitterAlreadyRegistered")]
    EmitterAlreadyRegistered = 0x2002,

    #[msg("RegisteredEmitterMismatch")]
    RegisteredEmitterMismatch = 0x2004,
}
