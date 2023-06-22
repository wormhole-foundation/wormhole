use thiserror::Error;

#[derive(Error, Debug)]
pub enum ContractError {
    #[error("failed to verify quorum")]
    VerifyQuorum,
}
