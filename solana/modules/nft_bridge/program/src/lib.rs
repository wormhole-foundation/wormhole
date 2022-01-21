#![feature(const_generics)]
#![allow(incomplete_features)]
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
    complete_native,
    complete_wrapped,
    complete_wrapped_meta,
    initialize,
    register_chain,
    transfer_native,
    transfer_wrapped,
    upgrade_contract,
    CompleteNative,
    CompleteNativeData,
    CompleteWrapped,
    CompleteWrappedData,
    CompleteWrappedMeta,
    CompleteWrappedMetaData,
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
    TokenNotNFT,
    InvalidAssociatedAccount,
    InvalidRecipient,
}

impl From<TokenBridgeError> for SolitaireError {
    fn from(t: TokenBridgeError) -> SolitaireError {
        SolitaireError::Custom(t as u64)
    }
}

solitaire! {
    Initialize(InitializeData) => initialize,
    CompleteNative(CompleteNativeData) => complete_native,
    CompleteWrapped(CompleteWrappedData) => complete_wrapped,
    CompleteWrappedMeta(CompleteWrappedMetaData) => complete_wrapped_meta,
    TransferWrapped(TransferWrappedData) => transfer_wrapped,
    TransferNative(TransferNativeData) => transfer_native,
    RegisterChain(RegisterChainData) => register_chain,
    UpgradeContract(UpgradeContractData) => upgrade_contract,
}
