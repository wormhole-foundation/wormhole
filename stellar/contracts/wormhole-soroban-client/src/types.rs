//! Public type definitions for the Wormhole Core contract.
//!
//! Contains the core data structures for VAA handling, guardian sets, and
//! cross-chain messaging. External contracts should use these types when
//! integrating with Wormhole.

use crate::{BytesReader, VAA_HEADER_MIN_LENGTH, WormholeError};
use core::convert::TryFrom;
use soroban_sdk::{Address, Bytes, BytesN, Env, Vec, contracttype};

// ========== VAA Types ==========

/// ECDSA signature from a Wormhole guardian.
///
/// Each signature contains the guardian's index for lookup and the secp256k1
/// ECDSA signature components (r, s, v) used to recover the signer's Ethereum
/// address.
#[contracttype]
#[derive(Clone, Debug, PartialEq)]
pub struct Signature {
    /// Position of the signing guardian in the guardian set (0-indexed).
    pub guardian_index: u32,
    /// ECDSA signature r component (32 bytes).
    pub r: BytesN<32>,
    /// ECDSA signature s component (32 bytes).
    pub s: BytesN<32>,
    /// ECDSA recovery ID (0 or 1, stored as u32 for Soroban compatibility).
    pub v: u32,
}

/// Verifiable Action Approval (VAA) - a cross-chain message attested by
/// guardians.
///
/// VAAs are the core primitive for Wormhole's cross-chain messaging. They
/// contain a message body signed by a quorum of guardians (13 of 19 on
/// mainnet). VAAs can carry arbitrary payloads including governance actions,
/// token transfers, or application-specific data.
///
/// Parse from bytes using `VAA::try_from((&env, &bytes))`.
#[contracttype]
#[derive(Clone, Debug, PartialEq)]
pub struct VAA {
    // ===== Header =====
    /// VAA format version (currently always 1).
    pub version: u32,
    /// Index of the guardian set that produced these signatures.
    pub guardian_set_index: u32,
    /// Guardian signatures attesting to the message body.
    pub signatures: Vec<Signature>,

    // ===== Body (signed data) =====
    /// Unix timestamp when the message was observed by guardians.
    pub timestamp: u32,
    /// Application-defined nonce for deduplication.
    pub nonce: u32,
    /// Wormhole chain ID of the source blockchain.
    pub emitter_chain: u32,
    /// Address of the emitting contract on the source chain.
    pub emitter_address: BytesN<32>,
    /// Auto-incrementing sequence number from the emitter.
    pub sequence: u64,
    /// Finality level requested when the message was posted.
    pub consistency_level: ConsistencyLevel,
    /// Application-specific payload data.
    pub payload: Bytes,
}

// ========== Guardian Set Types ==========

/// Metadata for a guardian set stored on-chain.
///
/// Guardian sets are indexed sequentially starting from 0. Only the current set
/// can sign new VAAs, but expired sets remain valid for a grace period to allow
/// in-flight VAAs to be processed.
#[contracttype]
#[derive(Clone, Debug)]
pub struct GuardianSetInfo {
    /// Ethereum addresses (20 bytes each) of guardians in this set.
    pub keys: Vec<BytesN<20>>,
    /// Ledger timestamp when this guardian set was activated.
    pub creation_time: u64,
}

// ========== Message Types ==========

/// Finality requirement for cross-chain message attestation.
///
/// Determines how many block confirmations guardians wait before signing.
/// Higher levels provide stronger finality guarantees but increase latency.
#[contracttype]
#[derive(Clone, Copy, Debug, PartialEq)]
#[repr(u8)]
pub enum ConsistencyLevel {
    /// Standard confirmation (~1 block, fastest).
    Confirmed = 1,
    /// Full finality (chain-specific, most secure).
    Finalized = 32,
}

impl TryFrom<u8> for ConsistencyLevel {
    type Error = crate::WormholeError;

    /// Converts a raw byte to a `ConsistencyLevel`.
    ///
    /// Only values 1 (`Confirmed`) and 32 (`Finalized`) are valid.
    fn try_from(value: u8) -> Result<Self, Self::Error> {
        match value {
            1 => Ok(ConsistencyLevel::Confirmed),
            32 => Ok(ConsistencyLevel::Finalized),
            _ => Err(crate::WormholeError::InvalidConsistencyLevel),
        }
    }
}

/// Serializable representation of a posted cross-chain message.
///
/// Created when a message is posted via `post_message`. The message data is
/// hashed and stored for later verification. Guardians observe these events
/// and produce VAAs attesting to their contents.
#[contracttype]
#[derive(Clone, Debug, PartialEq)]
pub struct PostedMessageData {
    /// Ledger timestamp when the message was posted.
    pub timestamp: u32,
    /// Caller-provided nonce for application-level deduplication.
    pub nonce: u32,
    /// Wormhole chain ID of this blockchain (61 for Stellar).
    pub emitter_chain: u32,
    /// 32-byte contract ID of the emitter contract.
    pub emitter_address: BytesN<32>,
    /// Auto-incrementing sequence number for this emitter.
    pub sequence: u64,
    /// Requested finality level for guardian attestation.
    pub consistency_level: ConsistencyLevel,
    /// Application-specific message payload.
    pub payload: Bytes,
}

// ========== Executor Types ==========

/// Off-chain quote submitted alongside a delivery request to the Wormhole
/// Executor contract.
///
/// A `SignedQuote` binds a `quoter` and a `payee` to a (`src_chain`,
/// `dst_chain`, `expiry`) tuple so that the payer knows who will relay the
/// request, who must be paid, and until when the quote is valid.
///
/// # Authentication is NOT performed on-chain
///
/// **Neither the [`prefix`](Self::prefix) domain tag nor the `quoter`'s
/// signature over the quote are verified by the Executor contract.** The
/// name "SignedQuote" refers to the fact that relayer SDKs produce these
/// structures with an off-chain signature, but the on-chain Executor treats
/// the struct as plain untrusted data: only `src_chain`, `dst_chain` and
/// `expiry` are checked, and the payer's authorization is required for the
/// token transfer.
///
/// Validating that a given `quoter` actually authored the quote is the
/// caller's responsibility and must be performed off-chain (typically by
/// the relayer SDK) before invoking `request_execution`.
#[contracttype]
#[derive(Clone, Debug, PartialEq)]
pub struct SignedQuote {
    /// 4-byte domain separator from the off-chain signing scheme (e.g.
    /// `b"EQ01"`)
    pub prefix: BytesN<4>,
    /// Address of the party that issued (and off-chain-signed) this quote.
    pub quoter: Address,
    /// Address credited with the `amount` native token transfer when
    /// `request_execution` succeeds.
    pub payee: Address,
    /// Wormhole chain id the quote was issued for as the source chain.
    /// Must equal the chain id configured at construction, otherwise
    /// `ExecutorError::QuoteSrcChainMismatch` is returned.
    pub src_chain: u32,
    /// Wormhole chain id of the intended destination chain. Must equal the
    /// `dst_chain` argument passed to `request_execution`, otherwise
    /// `ExecutorError::QuoteDstChainMismatch` is returned.
    pub dst_chain: u32,
    /// Unix timestamp (seconds) after which the quote is no longer valid.
    /// The contract rejects the call with `ExecutorError::QuoteExpired`
    /// whenever `expiry <= env.ledger().timestamp()`.
    pub expiry: u64,
}

// ========== Type Implementations ==========

impl Signature {
    /// Parses a 66-byte signature from the reader.
    ///
    /// Wire format: guardian_index (1) + r (32) + s (32) + v (1) = 66 bytes.
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

    /// Parses a VAA from its wire format representation.
    ///
    /// Validates minimum length but does not verify signatures; use
    /// `verify_vaa` or `verify_vaa_signatures` for full verification.
    fn try_from(value: (&'a Env, &'a Bytes)) -> Result<Self, Self::Error> {
        let (env, vaa_bytes) = value;

        if vaa_bytes.len() < VAA_HEADER_MIN_LENGTH {
            return Err(WormholeError::InvalidVAAFormat);
        }

        let mut reader = BytesReader::new(vaa_bytes);

        let version = u32::from(reader.read_u8()?);
        if version != 1 {
            return Err(WormholeError::InvalidVAAFormat);
        }
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
    /// Serializes the VAA body for hashing during signature verification.
    ///
    /// Returns the 51-byte fixed header plus variable payload. The body
    /// excludes the version, guardian set index, and signatures—guardians
    /// sign only this data. Output is big-endian encoded to match the
    /// Wormhole wire format.
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

    /// Extracts the body bytes from raw VAA data without full parsing.
    ///
    /// Computes the body offset by reading the signature count and slicing
    /// past the header and signatures. Useful for hash computation when
    /// only the body is needed.
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
