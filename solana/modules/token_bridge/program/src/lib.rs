
#![feature(adt_const_params)]
#![deny(unused_must_use)]

// #![cfg(all(target_arch = "bpf", not(feature = "no-entrypoint")))]

#[cfg(feature = "no-entrypoint")]
pub mod instructions;

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
extern crate wasm_bindgen;

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
pub mod wasm;

pub mod accounts;
pub mod api;
pub mod messages;
pub mod types;

pub use api::{
    attest_token,
    complete_native,
    complete_wrapped,
    create_wrapped,
    initialize,
    register_chain,
    transfer_native,
    transfer_wrapped,
    upgrade_contract,
    AttestToken,
    AttestTokenData,
    CompleteNative,
    CompleteNativeData,
    CompleteWrapped,
    CompleteWrappedData,
    CreateWrapped,
    CreateWrappedData,
    Initialize,
    InitializeData,
    RegisterChain,
    RegisterChainData,
    TransferNative,
    TransferNativeData,
    TransferWrapped,
    TransferWrappedData,
    UpgradeContract,
    UpgradeContractData,
};

use solitaire::*;
use std::error::Error;

pub enum TokenBridgeError {
    AlreadyExecuted,
    InvalidChain,
    InvalidGovernanceKey,
    InvalidMetadata,
    InvalidMint,
    InvalidPayload,
    InvalidUTF8String,
    TokenNotNative,
    UninitializedMint,
    WrongAccountOwner,
    InvalidFee,
    InvalidRecipient,
}

impl From<TokenBridgeError> for SolitaireError {
    fn from(t: TokenBridgeError) -> SolitaireError {
        SolitaireError::Custom(t as u64)
    }
}

solitaire! {
    Initialize      => initialize,
    AttestToken     => attest_token,
    CompleteNative  => complete_native,
    CompleteWrapped => complete_wrapped,
    TransferWrapped => transfer_wrapped,
    TransferNative  => transfer_native,
    RegisterChain   => register_chain,
    CreateWrapped   => create_wrapped,
    UpgradeContract => upgrade_contract,
}
