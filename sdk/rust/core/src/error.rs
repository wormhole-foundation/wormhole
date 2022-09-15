/// Ergonomic error handler for use within the Wormhole core/SDK libraries.
#[macro_export]
macro_rules! require {
    ($expr:expr, $name:ident) => {
        if !$expr {
            return Err($name.into());
        }
    };
}

/// ErrorCode maps to the nom ParseError
///
/// We use an integer instead of embedding the type because the library is deprecating the current
/// error type, so we should avoid depending on it for now. We can always map back to our integer
/// later if we need to.
type ErrorCode = usize;

#[derive(Debug)]
pub enum WormholeError {
    // Governance Errors
    UnknownGovernanceAction,
    InvalidGovernanceChain,
    InvalidGovernanceModule,

    // VAA Errors
    GuardianSetExpired,
    InvalidExpirationTime,
    InvalidGuardianSet,
    InvalidSignature,
    InvalidSignatureKey,
    InvalidSignaturePosition,
    InvalidVersion,
    QuorumNotMet,
    UnsortedSignatures,

    // Serialization Errors
    DeserializeFailed,
    ParseFailed(usize, ErrorCode),
}

impl WormholeError {
    pub fn from_parse_error(start: &[u8], end: &[u8], error: ErrorCode) -> Self {
        WormholeError::ParseFailed(start.len() - end.len(), error)
    }
}
