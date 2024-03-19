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

use std::fmt;

use serde::{Deserialize, Serialize};

pub mod accountant;
pub mod accountant_modification;
mod arraystring;
pub mod core;
pub mod ibc_receiver;
pub mod ibc_translator;
pub mod nft;
pub mod ntt_accountant;
pub mod relayer;
mod serde_array;
pub mod token;
pub mod vaa;

pub use vaa::Vaa;
pub use wormhole_supported_chains::Chain;

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

impl fmt::Display for Address {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        for b in self.0 {
            write!(f, "{b:02x}")?;
        }

        Ok(())
    }
}

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
}

impl GuardianSetInfo {
    pub fn quorum(&self) -> usize {
        (self.addresses.len() * 2) / 3 + 1
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn quorum() {
        let tests = [
            (0, 1),
            (1, 1),
            (2, 2),
            (3, 3),
            (4, 3),
            (5, 4),
            (6, 5),
            (7, 5),
            (8, 6),
            (9, 7),
            (10, 7),
            (11, 8),
            (12, 9),
            (13, 9),
            (14, 10),
            (15, 11),
            (16, 11),
            (17, 12),
            (18, 13),
            (19, 13),
            (50, 34),
            (100, 67),
            (1000, 667),
        ];

        for (count, quorum) in tests {
            let gs = GuardianSetInfo {
                addresses: vec![Default::default(); count],
            };

            assert_eq!(quorum, gs.quorum());
        }
    }
}
