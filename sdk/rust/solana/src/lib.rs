//! Crate providing Solana specific wormhole data types.
//!
//! This crate is a collection of data types used by the Solana implementation of the Wormhole
//! protocol. These are framework agnostic and can be used to interact with the Wormhole solana
//! deployment directly.

#![deny(warnings)]

mod accounts;
mod message;

// Re-export the entire instruction module as a namespace for functions that create wormhole
// instructions for CPI.
pub mod instructions;

// Re-export the minimal Wormhole API set required to develop contracts.
pub use {
    accounts::{
        Account,
        Config,
        Emitter,
        FeeCollector,
        GuardianSet,
        Sequence,
        VAA,
    },
    message::{
        post_message,
        Message,
    },
};
