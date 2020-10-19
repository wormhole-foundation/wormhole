use cosmwasm_std::{StdError};
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ContractError {
    // Generic error
    #[error("InvalidVersion")]
    InvalidVersion,
}

impl ContractError {
    pub fn std(&self) -> StdError {
        StdError::GenericErr{msg: format!("{}", self), backtrace: None}
    }
}