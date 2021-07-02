#![feature(const_generics)]
#![feature(const_generics_defaults)]
#![allow(warnings)]

// #![cfg(all(target_arch = "bpf", not(feature = "no-entrypoint")))]

#[cfg(feature = "no-entrypoint")]
pub mod instructions;

pub mod accounts;
pub mod api;
pub mod messages;
pub mod types;

use api::{
    attest_token, complete_native, complete_wrapped, create_wrapped, initialize, register_chain,
    transfer_native, transfer_wrapped, AttestToken, AttestTokenData, CompleteNative,
    CompleteNativeData, CompleteWrapped, CompleteWrappedData, CreateWrapped, CreateWrappedData,
    Initialize, RegisterChain, RegisterChainData, TransferNative, TransferNativeData,
    TransferWrapped, TransferWrappedData,
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
    Initialize(Pubkey) => initialize,
    AttestToken(AttestTokenData) => attest_token,
    CompleteNative(CompleteNativeData) => complete_native,
    CompleteWrapped(CompleteWrappedData) => complete_wrapped,
    TransferWrapped(TransferWrappedData) => transfer_wrapped,
    TransferNative(TransferNativeData) => transfer_native,
    RegisterChain(RegisterChainData) => register_chain,
    CreateWrapped(CreateWrappedData) => create_wrapped,
}
