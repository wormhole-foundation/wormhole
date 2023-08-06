#![feature(adt_const_params)]
#![allow(incomplete_features)]
#![deny(unused_must_use)]
// #![cfg(all(target_arch = "bpf", not(feature = "no-entrypoint")))]

#[cfg(feature = "instructions")]
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
    NonexistentTokenMetadataAccount,
    InvalidAssociatedAccount,
    InvalidRecipient,
    NotMetadataV1Account,
}

impl From<TokenBridgeError> for SolitaireError {
    fn from(t: TokenBridgeError) -> SolitaireError {
        SolitaireError::Custom(t as u64)
    }
}

solitaire! {
    Initialize          => initialize,
    CompleteNative      => complete_native,
    CompleteWrapped     => complete_wrapped,
    CompleteWrappedMeta => complete_wrapped_meta,
    TransferWrapped     => transfer_wrapped,
    TransferNative      => transfer_native,
    RegisterChain       => register_chain,
    UpgradeContract     => upgrade_contract,
}
