//! Wormhole Core Contract Interface for Stellar/Soroban.
//!
//! This crate provides the public API for interacting with the Wormhole Core contract.
//! External contracts should depend only on this interface crate for smaller WASM
//! binaries, while the implementation lives in `wormhole-contract`.
//!
//! # Key Types
//!
//! - [`VAA`] - Verifiable Action Approval, the core cross-chain message format
//! - [`GuardianSetInfo`] - Guardian set metadata stored on-chain
//! - [`ConsistencyLevel`] - Finality requirements for message attestation
//! - [`WormholeError`] - All possible error conditions
//!
//! # Example
//!
//! ```ignore
//! use wormhole_soroban_client::{WormholeCoreInterface, VAA, WormholeError};
//!
//! // Parse and verify a VAA
//! let vaa = VAA::try_from((&env, &vaa_bytes))?;
//! client.verify_vaa(&vaa_bytes)?;
//!
//! // Post a cross-chain message
//! let sequence = client.post_message(&emitter, nonce, &payload, ConsistencyLevel::Finalized)?;
//! ```

#![no_std]

pub mod bytes_reader;
pub mod constants;
pub mod error;
pub mod types;

pub use bytes_reader::BytesReader;
pub use constants::*;
pub use error::WormholeError;
pub use types::*;

use soroban_sdk::{Address, Bytes, BytesN, Env, Vec};

/// Complete public interface for the Wormhole Core contract.
///
/// Defines all contract entry points for VAA verification, governance actions,
/// cross-chain message posting, and state queries. The `wormhole-contract` crate
/// implements this trait with the `#[contractimpl]` macro.
///
/// # Security Model
///
/// - Governance actions require VAAs signed by a quorum (13/19) of guardians
/// - VAAs are consumed after use to prevent replay attacks
/// - The contract is its own admin—upgrades require guardian consensus
pub trait WormholeCoreInterface {
    // ========== Initialization ==========

    /// Initialize the contract with the initial guardian set and governance emitter.
    /// Can only be called once.
    ///
    /// # Arguments
    /// * `initial_guardians` - Ethereum addresses (20 bytes) of initial guardians
    /// * `governance_emitter` - The governance emitter address (32 bytes) that can issue governance VAAs.
    ///   For mainnet, this should be 0x0000000000000000000000000000000000000000000000000000000000000004
    ///   For testnet, you may use a different address you control for testing.
    ///
    /// # Errors
    /// * `Error::AlreadyInitialized` - Contract already initialized
    /// * `Error::EmptyGuardianSet` - No guardians provided
    fn initialize(
        env: Env,
        initial_guardians: Vec<BytesN<20>>,
        governance_emitter: BytesN<32>,
    ) -> Result<(), WormholeError>;

    /// Check if the contract has been initialized.
    ///
    /// # Returns
    /// `true` if initialized, `false` otherwise
    fn is_initialized(env: Env) -> bool;

    // ========== VAA Verification ==========

    /// Verify a complete VAA (Verifiable Action Approval).
    /// Parses the VAA and verifies all guardian signatures.
    ///
    /// # Arguments
    /// * `vaa_bytes` - Serialized VAA bytes
    ///
    /// # Returns
    /// `true` if VAA is valid and properly signed
    ///
    /// # Errors
    /// * `Error::InvalidVAAFormat` - Malformed VAA bytes
    /// * `Error::GuardianSetNotFound` - Guardian set not found
    /// * `Error::GuardianSetExpired` - Guardian set has expired
    /// * `Error::InsufficientSignatures` - Not enough signatures for quorum
    /// * `Error::InvalidSignature` - Invalid guardian signature
    fn verify_vaa(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

    /// Parse a VAA structure without signature verification.
    ///
    /// # Arguments
    /// * `vaa_bytes` - Serialized VAA bytes
    ///
    /// # Returns
    /// Parsed VAA structure
    ///
    /// # Errors
    /// * `Error::InvalidVAAFormat` - Malformed VAA bytes
    fn parse_vaa(env: Env, vaa_bytes: Bytes) -> Result<VAA, WormholeError>;

    // ========== Governance Actions ==========

    /// Submit a contract upgrade governance VAA.
    /// Requires valid VAA signed by current guardian set.
    ///
    /// # Arguments
    /// * `vaa_bytes` - Serialized governance VAA containing upgrade payload
    ///
    /// # Errors
    /// * VAA verification errors
    /// * `Error::InvalidGovernanceModule` - Wrong module in payload
    /// * `Error::InvalidGovernanceAction` - Wrong action ID
    /// * `Error::InvalidGovernanceChain` - Wrong chain ID
    /// * `Error::GovernanceVAAAlreadyConsumed` - VAA already processed
    fn submit_contract_upgrade(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

    /// Submit a guardian set upgrade governance VAA.
    /// Requires valid VAA signed by current guardian set.
    ///
    /// # Arguments
    /// * `vaa_bytes` - Serialized governance VAA containing guardian set upgrade payload
    ///
    /// # Errors
    /// * VAA verification errors
    /// * Governance validation errors
    /// * `Error::InvalidGuardianSetSequence` - New index not sequential
    /// * `Error::EmptyGuardianSet` - No guardians in new set
    fn submit_guardian_set_upgrade(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

    /// Submit a set message fee governance VAA.
    /// Requires valid VAA signed by current guardian set.
    ///
    /// # Arguments
    /// * `vaa_bytes` - Serialized governance VAA containing fee update payload
    ///
    /// # Errors
    /// * VAA verification errors
    /// * Governance validation errors
    fn submit_set_message_fee(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

    /// Submit a transfer fees governance VAA.
    /// Requires valid VAA signed by current guardian set.
    ///
    /// # Arguments
    /// * `vaa_bytes` - Serialized governance VAA containing fee transfer payload
    ///
    /// # Errors
    /// * VAA verification errors
    /// * Governance validation errors
    /// * `Error::InsufficientFees` - Not enough fees to transfer
    /// * `Error::TransferFailed` - Token transfer failed
    fn submit_transfer_fees(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

    // ========== Message Posting ==========

    /// Post a cross-chain message to be attested by Guardians.
    /// Collects message fee if configured.
    ///
    /// # Arguments
    /// * `emitter` - Address of the message emitter (must authorize)
    /// * `nonce` - Unique nonce for the message
    /// * `payload` - Message payload bytes
    /// * `consistency_level` - Finality requirement (Confirmed or Finalized)
    ///
    /// # Returns
    /// Sequence number assigned to the message
    ///
    /// # Errors
    /// * `Error::NotInitialized` - Contract not initialized
    /// * `Error::InsufficientFeePaid` - Fee not paid
    fn post_message(
        env: Env,
        emitter: Address,
        nonce: u32,
        payload: Bytes,
        consistency_level: ConsistencyLevel,
    ) -> Result<u64, WormholeError>;

    // ========== State Queries ==========

    /// Get the current active guardian set index.
    ///
    /// # Returns
    /// Index of the current guardian set
    fn get_current_guardian_set_index(env: Env) -> u32;

    /// Get a guardian set by index.
    ///
    /// # Arguments
    /// * `index` - Guardian set index
    ///
    /// # Returns
    /// Guardian set information
    ///
    /// # Errors
    /// * `Error::GuardianSetNotFound` - Guardian set does not exist
    fn get_guardian_set(env: Env, index: u32) -> Result<GuardianSetInfo, WormholeError>;

    /// Get the expiry timestamp for a guardian set.
    ///
    /// # Arguments
    /// * `index` - Guardian set index
    ///
    /// # Returns
    /// Expiry timestamp, or None if not expired
    fn get_guardian_set_expiry(env: Env, index: u32) -> Option<u64>;

    /// Get the current sequence number for an emitter.
    ///
    /// # Arguments
    /// * `emitter` - Emitter address
    ///
    /// # Returns
    /// Next sequence number for the emitter
    fn get_emitter_sequence(env: Env, emitter: Address) -> u64;

    /// Get the hash of a posted message by emitter and sequence number.
    ///
    /// # Arguments
    /// * `emitter` - Emitter address
    /// * `sequence` - Message sequence number
    ///
    /// # Returns
    /// Message hash, or None if not found
    fn get_posted_message_hash(env: Env, emitter: Address, sequence: u64) -> Option<BytesN<32>>;

    /// Get the current message fee in stroops (10^-7 XLM).
    ///
    /// # Returns
    /// Message fee in stroops
    fn get_message_fee(env: Env) -> u64;

    /// Get the timestamp of the last fee transfer.
    ///
    /// # Returns
    /// Timestamp of last transfer, or None if never transferred
    fn get_last_fee_transfer(env: Env) -> Option<u64>;

    /// Get the contract's current XLM balance.
    ///
    /// # Returns
    /// Balance in stroops
    fn get_contract_balance(env: Env) -> i128;

    /// Check if a governance VAA has been consumed (replay protection).
    ///
    /// # Arguments
    /// * `vaa_bytes` - Serialized VAA bytes
    ///
    /// # Returns
    /// `true` if VAA has been consumed
    ///
    /// # Errors
    /// * `Error::InvalidVAAFormat` - Malformed VAA bytes
    fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

    // ========== Protocol Constants ==========

    /// Get the Wormhole chain ID for Stellar (61).
    ///
    /// # Returns
    /// Chain ID as u32
    fn get_chain_id() -> u32;

    /// Get the governance chain ID (Solana = 1).
    ///
    /// # Returns
    /// Governance chain ID
    fn get_governance_chain_id() -> u32;

    /// Get the governance emitter address.
    ///
    /// # Returns
    /// 32-byte governance emitter address
    fn get_governance_emitter(env: Env) -> BytesN<32>;
}
