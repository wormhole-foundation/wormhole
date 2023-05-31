use anchor_lang::prelude::error_code;

#[error_code]
/// Errors relevant to Core Bridge's malfunction.
pub enum TokenBridgeError {
    #[msg("InvalidGovernanceEmitter")]
    InvalidGovernanceEmitter = 0x20,

    #[msg("InvalidGovernanceAction")]
    InvalidGovernanceAction = 0x22,

    #[msg("InvalidPostedVaa")]
    InvalidPostedVaa = 0x100,

    #[msg("InvalidTokenBridgeEmitter")]
    InvalidTokenBridgeEmitter = 0x101,

    #[msg("RecipientChainNotSolana")]
    RecipientChainNotSolana = 0x200,

    #[msg("EmitterZeroAddress")]
    EmitterZeroAddress = 0x300,

    #[msg("TransferRedeemerNotSigner")]
    TransferRedeemerNotSigner = 0x900,

    #[msg("CannotSerializeJson")]
    CannotSerializeJson = 0x6969,

    #[msg("InvalidRelayerFee")]
    InvalidRelayerFee = 0x420,
}
