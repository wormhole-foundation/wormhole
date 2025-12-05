//! Public type definitions for the Wormhole Core contract.
//!
//! This module contains all data types that external users need to interact with
//! the Wormhole Core contract, including VAAs, guardian sets, messages, and payloads.

use crate::{BytesReader, VAA_HEADER_MIN_LENGTH, WormholeError};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env, Vec, contracttype};

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
    pub consistency_level: ConsistencyLevel,
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

impl TryFrom<u8> for ConsistencyLevel {
    type Error = crate::WormholeError;

    fn try_from(value: u8) -> Result<Self, Self::Error> {
        match value {
            1 => Ok(ConsistencyLevel::Confirmed),
            32 => Ok(ConsistencyLevel::Finalized),
            _ => Err(crate::WormholeError::InvalidConsistencyLevel),
        }
    }
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

// ========== Type Implementations ==========

impl Signature {
    /// Parse a signature from a BytesReader
    pub fn parse_from_reader(reader: &mut BytesReader) -> Result<Self, WormholeError> {
        let guardian_index = u32::from(reader.read_u8()?);
        let r = reader.read_bytes_n::<32>()?;
        let s = reader.read_bytes_n::<32>()?;
        let v = u32::from(reader.read_u8()?);

        Ok(Signature {
            guardian_index,
            r,
            s,
            v,
        })
    }
}

impl<'a> TryFrom<(&'a Env, &'a Bytes)> for VAA {
    type Error = WormholeError;

    fn try_from(value: (&'a Env, &'a Bytes)) -> Result<Self, Self::Error> {
        let (env, vaa_bytes) = value;

        if vaa_bytes.len() < VAA_HEADER_MIN_LENGTH {
            return Err(WormholeError::InvalidVAAFormat);
        }

        let mut reader = BytesReader::new(vaa_bytes);

        let version = u32::from(reader.read_u8()?);
        let guardian_set_index = reader.read_u32_be()?;
        let num_signatures = u32::from(reader.read_u8()?);

        let mut signatures = Vec::new(env);
        for _ in 0..num_signatures {
            let sig = Signature::parse_from_reader(&mut reader)?;
            signatures.push_back(sig);
        }

        let timestamp = reader.read_u32_be()?;
        let nonce = reader.read_u32_be()?;
        let emitter_chain = u32::from(reader.read_u16_be()?);
        let emitter_address = reader.read_bytes_n::<32>()?;
        let sequence = reader.read_u64_be()?;
        let consistency_level = ConsistencyLevel::try_from(reader.read_u8()?)?;
        let payload = reader.remaining_bytes();

        Ok(VAA {
            version,
            guardian_set_index,
            signatures,
            timestamp,
            nonce,
            emitter_chain,
            emitter_address,
            sequence,
            consistency_level,
            payload,
        })
    }
}

impl VAA {
    /// Serialize the VAA body for hashing (excludes version, guardian set index, and signatures)
    ///
    /// The body consists of:
    /// - timestamp (4 bytes, big-endian)
    /// - nonce (4 bytes, big-endian)
    /// - emitter_chain (2 bytes, big-endian)
    /// - emitter_address (32 bytes)
    /// - sequence (8 bytes, big-endian)
    /// - consistency_level (1 byte)
    /// - payload (variable length)
    pub fn serialize_body(&self, env: &Env) -> Bytes {
        let mut bytes = Bytes::new(env);

        // timestamp (4 bytes, big-endian)
        for i in (0..4).rev() {
            bytes.push_back(((self.timestamp >> (i * 8)) & 0xFF) as u8);
        }

        // nonce (4 bytes, big-endian)
        for i in (0..4).rev() {
            bytes.push_back(((self.nonce >> (i * 8)) & 0xFF) as u8);
        }

        // emitter_chain (2 bytes, big-endian)
        bytes.push_back((self.emitter_chain >> 8) as u8);
        bytes.push_back((self.emitter_chain & 0xFF) as u8);

        // emitter_address (32 bytes)
        for byte in self.emitter_address.to_array().iter() {
            bytes.push_back(*byte);
        }

        // sequence (8 bytes, big-endian)
        for i in (0..8).rev() {
            bytes.push_back(((self.sequence >> (i * 8)) & 0xFF) as u8);
        }

        // consistency_level (1 byte)
        bytes.push_back(self.consistency_level as u8);

        // payload (variable length)
        bytes.append(&self.payload);

        bytes
    }

    /// Extract the VAA body bytes from raw VAA bytes without full parsing
    ///
    /// This is useful for computing hashes when you have the raw VAA bytes
    /// but don't need to fully parse the structure.
    pub fn get_body_bytes(vaa_bytes: &Bytes) -> Result<Bytes, WormholeError> {
        if vaa_bytes.len() < VAA_HEADER_MIN_LENGTH {
            return Err(WormholeError::InvalidVAAFormat);
        }

        // Calculate body offset: 6 (header) + 66 * num_signatures
        let num_sigs = u32::from(vaa_bytes.get(5).ok_or(WormholeError::InvalidVAAFormat)?);
        let body_offset = 6u32.saturating_add(66u32.saturating_mul(num_sigs));

        if body_offset >= vaa_bytes.len() {
            return Err(WormholeError::InvalidVAAFormat);
        }

        Ok(vaa_bytes.slice(body_offset..))
    }
}
