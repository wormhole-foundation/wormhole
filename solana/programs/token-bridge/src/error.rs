use anchor_lang::prelude::error_code;

#[error_code]
/// Errors relevant to Core Bridge's malfunction.
pub enum TokenBridgeError {
    #[msg("CannotParseMessage")]
    CannotParseMessage = 0x02,

    #[msg("InvalidTokenBridgeVaa")]
    InvalidTokenBridgeVaa = 0x04,

    #[msg("InvalidMint")]
    InvalidMint = 0x06,

    #[msg("InvalidGovernanceEmitter")]
    InvalidGovernanceEmitter = 0x20,

    #[msg("InvalidGovernanceAction")]
    InvalidGovernanceAction = 0x22,

    #[msg("GovernanceForAnotherChain")]
    GovernanceForAnotherChain = 0x23,

    #[msg("InvalidPostedVaa")]
    InvalidPostedVaa = 0x100,

    #[msg("InvalidTokenBridgeEmitter")]
    InvalidTokenBridgeEmitter = 0x101,

    #[msg("RecipientChainNotSolana")]
    RecipientChainNotSolana = 0x200,

    #[msg("RedeemerChainNotSolana")]
    RedeemerChainNotSolana = 0x201,

    #[msg("EmitterZeroAddress")]
    EmitterZeroAddress = 0x300,

    #[msg("TransferRedeemerNotSigner")]
    TransferRedeemerNotSigner = 0x900,

    #[msg("CannotSerializeJson")]
    CannotSerializeJson = 0x6969,

    #[msg("InvalidRelayerFee")]
    InvalidRelayerFee = 0x420,

    #[msg("NativeAsset")]
    NativeAsset = 0x555,

    #[msg("WrappedAsset")]
    WrappedAsset = 0x556,

    #[msg("U64Overflow")]
    U64Overflow = 0x558,

    #[msg("InvalidProgramRedeemer")]
    InvalidProgramRedeemer = 0x560,
}
