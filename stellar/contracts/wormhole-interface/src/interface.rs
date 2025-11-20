//! Wormhole Core Contract Interface
//!
//! This module defines the complete public API for the Wormhole Core contract
//! using a Rust trait.

use soroban_sdk::{Address, Bytes, BytesN, Env, Vec};

use crate::error::Error;
use crate::types::{ConsistencyLevel, GuardianSetInfo, VAA};

/// The complete public interface for the Wormhole Core contract.
///
/// This trait defines all functions that external contracts, clients,
/// and users can call on the Wormhole Core contract.
///
/// The `#[contractimpl]` macro automatically generates a client for this interface.
pub trait WormholeCoreInterface {
    // ========== Initialization ==========

    /// Initialize the contract with the initial guardian set.
    /// Can only be called once.
    ///
    /// # Arguments
    /// * `initial_guardians` - Ethereum addresses (20 bytes) of initial guardians
    ///
    /// # Errors
    /// * `Error::AlreadyInitialized` - Contract already initialized
    /// * `Error::EmptyGuardianSet` - No guardians provided
    fn initialize(env: Env, initial_guardians: Vec<BytesN<20>>) -> Result<(), Error>;

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
    fn verify_vaa(env: Env, vaa_bytes: Bytes) -> Result<bool, Error>;

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
    fn parse_vaa(env: Env, vaa_bytes: Bytes) -> Result<VAA, Error>;

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
    fn submit_contract_upgrade(env: Env, vaa_bytes: Bytes) -> Result<(), Error>;

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
    fn submit_guardian_set_upgrade(env: Env, vaa_bytes: Bytes) -> Result<(), Error>;

    /// Submit a set message fee governance VAA.
    /// Requires valid VAA signed by current guardian set.
    ///
    /// # Arguments
    /// * `vaa_bytes` - Serialized governance VAA containing fee update payload
    ///
    /// # Errors
    /// * VAA verification errors
    /// * Governance validation errors
    fn submit_set_message_fee(env: Env, vaa_bytes: Bytes) -> Result<(), Error>;

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
    fn submit_transfer_fees(env: Env, vaa_bytes: Bytes) -> Result<(), Error>;

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
    ) -> Result<u64, Error>;

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
    fn get_guardian_set(env: Env, index: u32) -> Result<GuardianSetInfo, Error>;

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
    fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<bool, Error>;

    // ========== Protocol Constants ==========

    /// Get the Wormhole chain ID for Stellar (61).
    ///
    /// # Returns
    /// Chain ID as u32
    fn get_chain_id(env: Env) -> u32;

    /// Get the governance chain ID (Solana = 1).
    ///
    /// # Returns
    /// Governance chain ID
    fn get_governance_chain_id(env: Env) -> u32;

    /// Get the governance emitter address.
    ///
    /// # Returns
    /// 32-byte governance emitter address
    fn get_governance_emitter(env: Env) -> BytesN<32>;
}
