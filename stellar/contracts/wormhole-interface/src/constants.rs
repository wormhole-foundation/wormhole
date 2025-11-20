//! Public protocol constants for the Wormhole Core contract.
//!
//! These constants are part of the public API and define core protocol parameters
//! that external users may need to reference.

#![allow(missing_docs)]

/// Governance chain ID (Solana by convention)
pub const GOVERNANCE_CHAIN_ID: u32 = 1;

/// Standard governance emitter address (0x00...04)
pub const GOVERNANCE_EMITTER: [u8; 32] = [
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
];

/// Stellar/Soroban chain ID in Wormhole.
///
/// Note: Chain IDs are u16 in the Wormhole protocol (2 bytes on wire).
/// All Wormhole chain IDs fit within u16 range (max 65,535).
pub const CHAIN_ID_STELLAR: u16 = 61;

/// Guardian set expiration time in seconds (24 hours)
pub const GUARDIAN_SET_EXPIRATION_TIME: u32 = 86400;
