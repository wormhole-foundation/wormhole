//! The `core` module provides all the pure Rust Wormhole primitives.
//!
//! This crate provides chain-agnostic types from Wormhole for consumption in on-chain contracts
//! and within other chain-specific Wormhole Rust SDK's. It includes:
//!
//! - Constants containing known network data/addresses.
//! - Parsers for VAA's and Payloads.
//! - Data types for Wormhole primitives such as GuardianSets and signatures.
//! - Verification Primitives for securely checking payloads.

#![deny(warnings)]
#![deny(unused_results)]

use serde::{Deserialize, Serialize};

mod arraystring;
mod chain;
pub mod core;
pub mod nft;
mod serde_array;
pub mod token;
pub mod vaa;

pub use {chain::Chain, vaa::Vaa};

/// The `GOVERNANCE_EMITTER` is a special address Wormhole guardians trust to observe governance
/// actions from. The value is "0000000000000000000000000000000000000000000000000000000000000004".
pub const GOVERNANCE_EMITTER: Address = Address([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
]);

#[derive(
    Serialize, Deserialize, Debug, Default, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash,
)]
pub struct GuardianAddress(pub [u8; 20]);

/// Wormhole specifies addresses as 32 bytes. Addresses that are shorter, for example 20 byte
/// Ethereum addresses, are left zero padded to 32.
#[derive(
    Serialize, Deserialize, Debug, Default, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash,
)]
pub struct Address(pub [u8; 32]);

/// Wormhole specifies an amount as a uint256 encoded in big-endian order.
#[derive(
    Serialize, Deserialize, Debug, Default, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash,
)]
pub struct Amount(pub [u8; 32]);

/// A `GuardianSet` is a versioned set of keys that can sign Wormhole messages.
#[derive(Serialize, Deserialize, Debug, Default, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct GuardianSetInfo {
    /// The set of guardians public keys, in Ethereum's compressed format.
    pub addresses: Vec<GuardianAddress>,

    /// How long after a GuardianSet change before this set is expired.
    #[serde(skip)]
    pub expiration_time: u64,
}

impl GuardianSetInfo {
    pub fn quorum(&self) -> usize {
        // allow quorum of 0 for testing purposes...
        if self.addresses.is_empty() {
            0
        } else {
            ((self.addresses.len() * 10 / 3) * 2) / 10 + 1
        }
    }
}
