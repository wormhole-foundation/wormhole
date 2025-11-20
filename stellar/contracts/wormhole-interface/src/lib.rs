//! Wormhole Core Contract Interface
//!
//! This crate defines the public API for the Wormhole Core contract on Stellar/Soroban.
//! External contracts and clients should depend only on this interface crate, which
//! provides type definitions and the contract interface without any implementation details.
//!
//! # Usage
//!
//! Add this to your `Cargo.toml`:
//! ```toml
//! [dependencies]
//! wormhole-interface = { path = "../wormhole-interface" }
//! ```
//!
//! Then use the types and interface:
//! ```ignore
//! use wormhole_interface::{WormholeCoreInterface, VAA, Error};
//! ```

#![no_std]

pub mod constants;
pub mod error;
pub mod interface;
pub mod types;

// Re-export everything for convenient imports
pub use constants::*;
pub use error::Error;
pub use interface::WormholeCoreInterface;
pub use types::*;
