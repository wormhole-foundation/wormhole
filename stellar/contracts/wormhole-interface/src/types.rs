//! Public type definitions for the Wormhole Core contract.
//!
//! This module contains all data types that external users need to interact with
//! the Wormhole Core contract, including VAAs, guardian sets, messages, and payloads.

#![allow(missing_docs)]

use soroban_sdk::{contracttype, Bytes, BytesN, Vec};

// ========== VAA Types ==========

/// ECDSA signature from a guardian.
#[contracttype]
#[derive(Clone, Debug, PartialEq)]
pub struct Signature {
    /// Index of the guardian in the guardian set
    pub guardian_index: u32,
    /// ECDSA signature r value
    pub r: BytesN<32>,
    /// ECDSA signature s value
    pub s: BytesN<32>,
    /// ECDSA recovery ID
    pub v: u32,
}

/// Verifiable Action Approval - a signed message from Wormhole guardians.
#[contracttype]
#[derive(Clone, Debug, PartialEq)]
pub struct VAA {
    // Header
    /// VAA format version (always 1)
    pub version: u32,
    /// Index of the guardian set that signed this VAA
    pub guardian_set_index: u32,
    /// Guardian signatures (quorum required)
    pub signatures: Vec<Signature>,

    // Body
    /// Unix timestamp when the VAA was created
    pub timestamp: u32,
    /// Unique nonce for the VAA
    pub nonce: u32,
    /// Source blockchain chain ID
    pub emitter_chain: u32,
    /// Source contract/account address (32 bytes)
    pub emitter_address: BytesN<32>,
    /// Message sequence number
    pub sequence: u64,
    /// Finality requirement level
    pub consistency_level: u32,
    /// Action-specific payload data
    pub payload: Bytes,
}

// ========== Guardian Set Types ==========

/// Information about a guardian set.
#[contracttype]
#[derive(Clone, Debug)]
pub struct GuardianSetInfo {
    /// Ethereum addresses of guardians in this set
    pub keys: Vec<BytesN<20>>,
    /// Ledger timestamp when this guardian set was created
    pub creation_time: u64,
}

// ========== Message Types ==========

/// Finality level for cross-chain messages.
#[contracttype]
#[derive(Clone, Copy, Debug, PartialEq)]
#[repr(u8)]
pub enum ConsistencyLevel {
    /// Standard confirmation (1 block)
    Confirmed = 1,
    /// Full finality (multiple blocks)
    Finalized = 32,
}

/// Data for a posted cross-chain message.
#[contracttype]
#[derive(Clone, Debug, PartialEq)]
pub struct PostedMessageData {
    /// Unix timestamp when the message was posted
    pub timestamp: u32,
    /// Unique nonce for the message
    pub nonce: u32,
    /// Source blockchain chain ID
    pub emitter_chain: u32,
    /// Source contract/account address (32 bytes)
    pub emitter_address: BytesN<32>,
    /// Message sequence number
    pub sequence: u64,
    /// Finality requirement level
    pub consistency_level: ConsistencyLevel,
    /// Message payload data
    pub payload: Bytes,
}

