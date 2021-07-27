#![feature(const_generics)]
#![allow(warnings)]

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
    InvalidPayload,
    Unknown(String),
    InvalidMint,
    WrongAccountOwner,
    InvalidUTF8String,
    AlreadyExecuted,
    InvalidChain,
    TokenNotNative,
    InvalidGovernanceKey,
}

impl<T: Error> From<T> for TokenBridgeError {
    fn from(t: T) -> Self {
        return TokenBridgeError::Unknown(t.to_string());
    }
}

impl Into<SolitaireError> for TokenBridgeError {
    fn into(self) -> SolitaireError {
        SolitaireError::Custom(0)
    }
}

solitaire! {
    Initialize(InitializeData) => initialize,
    AttestToken(AttestTokenData) => attest_token,
    CompleteNative(CompleteNativeData) => complete_native,
    CompleteWrapped(CompleteWrappedData) => complete_wrapped,
    TransferWrapped(TransferWrappedData) => transfer_wrapped,
    TransferNative(TransferNativeData) => transfer_native,
    RegisterChain(RegisterChainData) => register_chain,
    CreateWrapped(CreateWrappedData) => create_wrapped,
    UpgradeContract(UpgradeContractData) => upgrade_contract,
}
