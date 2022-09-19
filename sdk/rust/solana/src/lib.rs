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
pub mod instruction;

// Re-export minimal set of specific types/functions to develop contracts. This set acts as the
// definition of the Wormhole API for Solana.
pub use {
    accounts::{
        Config,
        Emitter,
        FeeCollector,
        Sequence,
        VAA,
    },
    message::{
        post_message,
        Message,
    },
};
