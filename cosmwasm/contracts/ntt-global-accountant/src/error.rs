use std::ops::{Deref, DerefMut};

use anyhow::anyhow;
use cosmwasm_std::StdError;
use thiserror::Error;
use wormhole_sdk::Chain;

#[derive(Error, Debug)]
pub enum ContractError {
    #[error("failed to verify quorum")]
    VerifyQuorum,
    #[error("no registered emitter for chain {0}")]
    MissingChainRegistration(Chain),
    #[error("failed to calculate digest of observation")]
    ObservationDigest,
    #[error("message already processed")]
    DuplicateMessage,
    #[error("digest mismatch for processed message")]
    DigestMismatch,
}

// This is a workaround for the fact that `cw_multi_test::ContractWrapper` doesn't support contract
// functions returning `anyhow::Error` directly.
#[derive(Error, Debug)]
#[repr(transparent)]
#[error("{0:#}")]
pub struct AnyError(#[from] anyhow::Error);

impl Deref for AnyError {
    type Target = anyhow::Error;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for AnyError {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl From<StdError> for AnyError {
    fn from(e: StdError) -> AnyError {
        anyhow!(e).into()
    }
}

impl From<ContractError> for AnyError {
    fn from(e: ContractError) -> AnyError {
        anyhow!(e).into()
    }
}

// Workaround for not being able to use the `bail!` macro directly.
#[doc(hidden)]
#[macro_export]
macro_rules! bail {
    ($msg:literal $(,)?) => {
        return ::core::result::Result::Err(::anyhow::anyhow!($msg).into())
    };
    ($err:expr $(,)?) => {
        return ::core::result::Result::Err(::anyhow::anyhow!($err).into())
    };
    ($fmt:expr, $($arg:tt)*) => {
        return ::core::result::Result::Err(::anyhow::anyhow!($fmt, $($arg)*).into())
    };
}
