use anchor_lang::prelude::error_code;

#[error_code]
pub enum WormholeVaaVerificationComparisonError {
    #[msg("EmptyGuardianSignatures")]
    EmptyGuardianSignatures = 0x100,

    #[msg("WriteAuthorityMismatch")]
    WriteAuthorityMismatch = 0x101,

    #[msg("GuardianSetExpired")]
    GuardianSetExpired = 0x102,

    #[msg("InvalidGuardianKeyRecovery")]
    InvalidGuardianKeyRecovery = 0x103,

    #[msg("NoQuorum")]
    NoQuorum = 0x104,

    #[msg("InvalidSignature")]
    InvalidSignature = 0x105,

    #[msg("InvalidGuardianIndexNonIncreasing")]
    InvalidGuardianIndexNonIncreasing = 0x106,

    #[msg("InvalidGuardianIndexOutOfRange")]
    InvalidGuardianIndexOutOfRange = 0x107,
}
