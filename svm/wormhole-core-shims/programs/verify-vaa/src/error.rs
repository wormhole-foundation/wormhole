use anchor_lang::prelude::error_code;

#[error_code]
pub enum WormholeVerifyVaaShim {
    #[msg("GuardianSetExpired")]
    GuardianSetExpired = 0x2,

    #[msg("NoQuorum")]
    NoQuorum = 0x3,

    #[msg("InvalidSignature")]
    InvalidSignature = 0x4,

    #[msg("InvalidGuardianIndexNonIncreasing")]
    InvalidGuardianIndexNonIncreasing = 0x5,

    #[msg("InvalidGuardianIndexOutOfRange")]
    InvalidGuardianIndexOutOfRange = 0x6,

    #[msg("InvalidGuardianKeyRecovery")]
    InvalidGuardianKeyRecovery = 0x7,
}
