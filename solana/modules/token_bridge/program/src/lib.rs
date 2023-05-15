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
    attest_token,
    complete_native,
    complete_native_with_payload,
    complete_wrapped,
    complete_wrapped_with_payload,
    create_wrapped,
    initialize,
    register_chain,
    transfer_native,
    transfer_native_with_payload,
    transfer_wrapped,
    transfer_wrapped_with_payload,
    upgrade_contract,
    AttestToken,
    AttestTokenData,
    CompleteNative,
    CompleteNativeData,
    CompleteNativeWithPayload,
    CompleteNativeWithPayloadData,
    CompleteWrapped,
    CompleteWrappedData,
    CompleteWrappedWithPayload,
    CompleteWrappedWithPayloadData,
    CreateWrapped,
    CreateWrappedData,
    Initialize,
    InitializeData,
    RegisterChain,
    RegisterChainData,
    TransferNative,
    TransferNativeData,
    TransferNativeWithPayload,
    TransferNativeWithPayloadData,
    TransferWrapped,
    TransferWrappedData,
    TransferWrappedWithPayload,
    TransferWrappedWithPayloadData,
    UpgradeContract,
    UpgradeContractData,
};

use solitaire::*;

// Static list of invalid VAA Message accounts.
pub(crate) static INVALID_VAAS: &[&str; 7] = &[
    "28Tx7c3W8rggVNyUQEAL9Uq6pUng4xJLAeLA6V8nLH1Z",
    "32YEuzLCvSyHoV6NFpaTXfiAB8sHiAnYcvP2BBeLeGWq",
    "427N2RrDHYooLvyWCiEiNR4KtGsGFTMuXiGwtuChWRSd",
    "56Vf4Y2SCxJBf4TSR24fPF8qLHhC8ZuTJvHS6mLGWieD",
    "7SzK4pmh9fM9SWLTCKmbjQC8EvDgPmtwdaBeTRztkM98",
    "G2VJNjmQsz6wfVZkTUzYAB8ZzRS2hZbpUd5Cr4DTpz6t",
    "GvAarWUV8khMLrTRouzBh3xSr8AeLDXxoKNJ6FgxGyg5",
];

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
    InvalidVAA,
    NonexistentTokenMetadataAccount,
    NotMetadataV1Account,
}

impl From<TokenBridgeError> for SolitaireError {
    fn from(t: TokenBridgeError) -> SolitaireError {
        SolitaireError::Custom(t as u64)
    }
}

solitaire! {
    Initialize => initialize,
    AttestToken => attest_token,
    CompleteNative => complete_native,
    CompleteWrapped => complete_wrapped,
    TransferWrapped => transfer_wrapped,
    TransferNative => transfer_native,
    RegisterChain => register_chain,
    CreateWrapped => create_wrapped,
    UpgradeContract => upgrade_contract,
    CompleteNativeWithPayload => complete_native_with_payload,
    CompleteWrappedWithPayload => complete_wrapped_with_payload,
    TransferWrappedWithPayload => transfer_wrapped_with_payload,
    TransferNativeWithPayload => transfer_native_with_payload,
}
