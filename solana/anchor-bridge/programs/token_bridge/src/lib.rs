#![feature(const_generics)]
#![feature(const_generics_defaults)]
#![allow(warnings)]

// #![cfg(all(target_arch = "bpf", not(feature = "no-entrypoint")))]

mod api;
mod messages;
mod types;
mod vaa;

use api::{initialize, Initialize};

use solitaire::*;
use std::error::Error;

pub enum TokenBridgeError {
    InvalidPayload,
    Unknown(String),
    InvalidMint,
    WrongAccountOwner,
    InvalidUTF8String,
    AlreadyExecuted,
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
}
