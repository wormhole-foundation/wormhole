//! Wormhole Core and Executor Contract Interfaces for Stellar/Soroban.
//!
//! This crate provides the public API for interacting with the Wormhole Core
//! and Executor contracts. External contracts should depend only on this
//! interface crate for smaller WASM binaries, while the implementations live
//! in `wormhole-contract` and `wormhole-executor`.
//!
//! # Key Types
//!
//! Wormhole Core:
//! - [`VAA`] - Verifiable Action Approval, the core cross-chain message format
//! - [`GuardianSetInfo`] - Guardian set metadata stored on-chain
//! - [`ConsistencyLevel`] - Finality requirements for message attestation
//! - [`WormholeError`] - Wormhole Core error conditions
//!
//! Executor:
//! - [`SignedQuote`] - Off-chain quote payload
//! - [`ExecutorError`] - Executor error conditions
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
//! // Post a cross-chain message (emitter is always treated as a contract)
//! let sequence = client.post_message(&emitter, nonce, &payload, ConsistencyLevel::Finalized)?;
//! ```

#![no_std]

pub mod bytes_reader;
pub mod constants;
pub mod error;
pub mod types;

pub use bytes_reader::BytesReader;
pub use constants::*;
pub use error::{ExecutorError, WormholeError};
pub use types::*;

use soroban_sdk::{Address, Bytes, BytesN, Env, String, contractclient};

/// Computes the canonical Wormhole lookup hash for a Soroban address.
///
/// The hash input is the address StrKey string bytes.
pub fn hash_address(env: &Env, address: &Address) -> BytesN<32> {
    env.crypto()
        .keccak256(&address.to_string().to_bytes())
        .to_bytes()
}

/// Complete public interface for the Wormhole Core contract.
///
/// Defines all contract entry points for VAA verification, governance actions,
/// cross-chain message posting, and state queries. The `wormhole-contract`
/// crate implements this trait with the `#[contractimpl]` macro.
///
/// # Security Model
///
/// - Governance actions require VAAs signed by a quorum (13/19) of guardians
/// - VAAs are consumed after use to prevent replay attacks
/// - The contract is its own admin—upgrades require guardian consensus
///
/// # Initialization
///
/// The contract is initialized via `__constructor` at deployment time.
/// Constructor arguments (initial guardians and governance emitter) are passed
/// during the `stellar contract deploy` command after `--`.
#[contractclient(name = "WormholeClient")]
pub trait WormholeCoreInterface {
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

    /// Parse and verify a VAA in a single call.
    ///
    /// Equivalent to calling `verify_vaa` then `parse_vaa`, but parses the
    /// VAA only once. This is the recommended entry point for integrators
    /// who need both the parsed structure and signature verification.
    ///
    /// # Arguments
    /// * `vaa_bytes` - Serialized VAA bytes
    ///
    /// # Returns
    /// Parsed and verified VAA structure
    ///
    /// # Errors
    /// * `Error::InvalidVAAFormat` - Malformed VAA bytes
    /// * `Error::GuardianSetNotFound` - Guardian set not found
    /// * `Error::GuardianSetExpired` - Guardian set has expired
    /// * `Error::InsufficientSignatures` - Not enough signatures for quorum
    /// * `Error::InvalidSignature` - Invalid guardian signature
    fn parse_and_verify_vaa(env: Env, vaa_bytes: Bytes) -> Result<VAA, WormholeError>;

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
    /// * `vaa_bytes` - Serialized governance VAA containing guardian set
    ///   upgrade payload
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
    /// * `vaa_bytes` - Serialized governance VAA containing fee transfer
    ///   payload
    ///
    /// # Errors
    /// * VAA verification errors
    /// * Governance validation errors
    /// * `Error::InsufficientFees` - Not enough fees to transfer
    /// * `Error::TransferFailed` - Token transfer failed
    fn submit_transfer_fees(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

    // ========== Message Posting ==========

    /// Post a cross-chain message to be attested by Guardians.
    /// The emitter is always treated as a contract address. Collects message
    /// fee if configured.
    ///
    /// # Arguments
    /// * `emitter` - Contract address acting as the message emitter (must
    ///   authorize)
    /// * `nonce` - Unique nonce for the message
    /// * `payload` - Message payload bytes
    /// * `consistency_level` - Finality requirement (Confirmed or Finalized)
    ///
    /// # Returns
    /// Sequence number assigned to the message
    ///
    /// # Errors
    /// * `Error::InsufficientFeePaid` - Fee not paid (requires prior token
    ///   approval)
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

    /// Get the current message fee in stroops (10^-7 XLM).
    ///
    /// # Returns
    /// Message fee in stroops
    fn get_message_fee(env: Env) -> u64;

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
    fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<bool, WormholeError>;

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

    /// Record a Soroban address in the hash lookup table.
    ///
    /// Returns the canonical hash used as the lookup key.
    fn record_address(env: Env, address: Address) -> Result<BytesN<32>, WormholeError>;

    /// Resolve a previously recorded address hash.
    fn get_address_from_hash(env: Env, hash: BytesN<32>) -> Result<Address, WormholeError>;
}

#[cfg(test)]
mod tests {
    use super::*;
    use soroban_sdk::testutils::Address as _;

    #[test]
    fn test_hash_address_uses_strkey_string_bytes() {
        let env = Env::default();
        let address = Address::generate(&env);
        let expected = env
            .crypto()
            .keccak256(&address.to_string().to_bytes())
            .to_bytes();

        assert_eq!(hash_address(&env, &address), expected);
    }
}

/// Public interface for the Wormhole Executor contract.
///
/// The Executor is a prepaid cross-chain delivery payment rail. A `payer`
/// submits a [`SignedQuote`] alongside a delivery request, the contract
/// validates the quote, transfers the agreed `amount` of native token from
/// the payer to the quote's `payee`, and emits an event consumed by
/// off-chain relayers that fulfil the delivery on the destination chain.
///
/// # Quote authentication is NOT performed on-chain
///
/// Despite its name, [`SignedQuote`] is **not** verified by the Executor.
/// Neither the [`SignedQuote::prefix`] domain tag nor the `quoter`'s
/// signature over the quote are checked on chain. Quote authentication is a
/// caller-side responsibility, performed off-chain by the relayer SDK before
/// the transaction is submitted.
#[contractclient(name = "ExecutorClient")]
pub trait ExecutorInterface {
    /// Returns the Wormhole chain id configured at construction.
    fn chain_id(env: Env) -> u32;

    /// Returns the version string of the Executor implementation
    /// (e.g. `"Executor-0.0.1"`).
    fn executor_version(env: Env) -> String;

    /// Records a prepaid cross-chain delivery request.
    ///
    /// Validates the [`SignedQuote`], requires the payer's authorization,
    /// transfers `amount` native tokens from `payer` to
    /// `signed_quote.payee`, and emits a `RequestForExecution` event
    /// consumed by off-chain relayers.
    ///
    /// # Arguments
    ///
    /// * `dst_chain` - Wormhole chain id of the destination chain. Must equal
    ///   `signed_quote.dst_chain`.
    /// * `dst_addr_wa32` - 32-byte destination address in Wormhole's
    ///   left-zero-padded encoding. Pass-through; not validated.
    /// * `refund` - Address the off-chain relayer should refund on delivery
    ///   failure. Pass-through metadata; not used on chain.
    /// * `payer` - Address that pays the `amount` and whose authorization is
    ///   required for the token transfer.
    /// * `amount` - Amount in stroops of the native token (XLM via the Stellar
    ///   Asset Contract at [`NATIVE_TOKEN_ADDRESS`]) transferred from `payer`
    ///   to `signed_quote.payee`. Must be non-negative; may be zero.
    /// * `signed_quote` - Off-chain-signed quote (see [`SignedQuote`] for the
    ///   authentication caveat). Only `src_chain`, `dst_chain`, `expiry` and
    ///   `payee` are used by on-chain logic.
    /// * `request` - Opaque delivery request payload forwarded to off-chain
    ///   relayers via the emitted event.
    /// * `relay_instructions` - Opaque relaying instructions forwarded to
    ///   off-chain relayers via the emitted event.
    ///
    /// # Errors
    ///
    /// Returns:
    ///
    /// - [`ExecutorError::InvalidAmount`] if `amount < 0`.
    /// - [`ExecutorError::QuoteSrcChainMismatch`] if `signed_quote.src_chain`
    ///   does not match the configured chain id.
    /// - [`ExecutorError::QuoteDstChainMismatch`] if `signed_quote.dst_chain`
    ///   does not match the `dst_chain` argument.
    /// - [`ExecutorError::QuoteExpired`] if `signed_quote.expiry <=
    ///   env.ledger().timestamp()`.
    ///
    /// # Authorization
    ///
    /// Calls `payer.require_auth()` before transferring the native token;
    /// the transaction must therefore include the payer's authorization.
    ///
    /// # Events
    ///
    /// On success, publishes exactly one `RequestForExecution` event with
    /// topics `["Executor", "RequestForExecution"]` after the token
    /// transfer has completed.
    #[allow(clippy::too_many_arguments)]
    fn request_execution(
        env: Env,
        dst_chain: u32,
        dst_addr_wa32: BytesN<32>,
        refund: Address,
        payer: Address,
        amount: i128,
        signed_quote: SignedQuote,
        request: Bytes,
        relay_instructions: Bytes,
    ) -> Result<(), ExecutorError>;
}
