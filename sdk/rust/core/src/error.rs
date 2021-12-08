/// Ergonomic error handler for use within the Wormhole core/SDK libraries.
#[macro_export]
macro_rules! require {
    ($expr:expr, $name:ident) => {
        if !$expr {
            return Err($name.into());
        }
    }
}

/// This ErrorCode maps to the nom ParseError, we use an integer because the library is deprecating
/// the current error type, so we should avoid depending on it for now.
type ErrorCode = usize;

#[derive(Debug)]
pub enum WormholeError {
    InvalidGovernanceAction,
    InvalidGovernanceChain,
    InvalidGovernanceModule,
    DeserializeFailed,
    ParseError(ErrorCode),
}
