//! Common governance action trait and processing utilities.
//!
//! Defines the [`GovernanceAction`] trait that all governance actions implement,
//! providing a standard flow: verify VAA → check replay → parse payload →
//! validate → consume → execute.

use crate::{storage::StorageKey, utils::keccak256_hash, vaa::verify_vaa};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env};
use wormhole_soroban_client::{
    BytesReader, CHAIN_ID_STELLAR, GOVERNANCE_CHAIN_ID, GOVERNANCE_EMITTER, MODULE_CORE,
    STORAGE_TTL_EXTENSION, STORAGE_TTL_THRESHOLD, VAA, WormholeError,
};

/// Trait for implementing Wormhole governance actions.
///
/// Each action defines a `Payload` type with parsing via `TryFrom`, validation
/// logic, and execution behavior. The `submit` method provides the standard
/// governance flow with replay protection.
///
/// **Critical**: VAA consumption happens BEFORE execution to prevent replay
/// if the execution modifies contract behavior (e.g., contract upgrades).
pub trait GovernanceAction {
    /// Payload type for this action, parsed from VAA payload bytes.
    type Payload: for<'a> TryFrom<(&'a Env, &'a Bytes), Error = WormholeError>;

    /// Validates the parsed payload (e.g., sequential guardian set index).
    fn validate_payload(env: &Env, payload: &Self::Payload) -> Result<(), WormholeError>;

    /// Executes the governance action after all validation passes.
    fn execute(env: &Env, vaa: &VAA, payload: &Self::Payload) -> Result<(), WormholeError>;

    /// Standard governance submission flow with replay protection.
    ///
    /// 1. Parse and verify VAA (signatures + governance source)
    /// 2. Check VAA not already consumed
    /// 3. Parse and validate payload
    /// 4. **Consume VAA (before execution!)**
    /// 5. Execute the action
    fn submit(env: &Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        let (vaa, vaa_hash) = verify_and_hash_governance_vaa(env, &vaa_bytes)?;
        require_vaa_not_consumed(env, &vaa_hash)?;

        let payload = Self::Payload::try_from((env, &vaa.payload))?;
        Self::validate_payload(env, &payload)?;

        consume_vaa(env, &vaa_hash);

        Self::execute(env, &vaa, &payload)?;

        Ok(())
    }
}

/// Validates that a VAA originates from the authorized governance source.
fn verify_governance_vaa(vaa: &VAA) -> Result<(), WormholeError> {
    if vaa.emitter_chain != GOVERNANCE_CHAIN_ID {
        return Err(WormholeError::InvalidGovernanceChain);
    }

    if vaa.emitter_address.to_array() != GOVERNANCE_EMITTER {
        return Err(WormholeError::InvalidGovernanceEmitter);
    }

    Ok(())
}

/// Parses, verifies, and hashes a governance VAA for consumption tracking.
fn verify_and_hash_governance_vaa(
    env: &Env,
    vaa_bytes: &Bytes,
) -> Result<(VAA, BytesN<32>), WormholeError> {
    let vaa = VAA::try_from((env, vaa_bytes))?;
    verify_governance_vaa(&vaa)?;
    verify_vaa(env, vaa_bytes)?;

    let body_bytes = vaa.serialize_body(env);
    let vaa_hash = keccak256_hash(env, &body_bytes);

    Ok((vaa, vaa_hash))
}

/// Checks if a VAA body hash has been consumed.
fn is_vaa_consumed(env: &Env, hash: &BytesN<32>) -> bool {
    env.storage()
        .persistent()
        .has(&StorageKey::ConsumedGovernanceVAA(hash.clone()))
}

/// Returns error if VAA has already been consumed (replay protection).
fn require_vaa_not_consumed(env: &Env, hash: &BytesN<32>) -> Result<(), WormholeError> {
    if is_vaa_consumed(env, hash) {
        return Err(WormholeError::GovernanceVAAAlreadyConsumed);
    }
    Ok(())
}

/// Marks a VAA as consumed to prevent replay attacks.
fn consume_vaa(env: &Env, hash: &BytesN<32>) {
    env.storage()
        .persistent()
        .set(&StorageKey::ConsumedGovernanceVAA(hash.clone()), &true);

    env.storage().persistent().extend_ttl(
        &StorageKey::ConsumedGovernanceVAA(hash.clone()),
        STORAGE_TTL_THRESHOLD,
        STORAGE_TTL_EXTENSION,
    );
}

/// Checks if a governance VAA has been consumed without full verification.
pub fn is_governance_vaa_consumed_from_bytes(
    env: &Env,
    vaa_bytes: &Bytes,
) -> Result<bool, WormholeError> {
    let body_bytes = VAA::get_body_bytes(vaa_bytes)?;
    let vaa_hash = keccak256_hash(env, &body_bytes);
    Ok(is_vaa_consumed(env, &vaa_hash))
}

/// Parses the 35-byte governance header: module (32) + action (1) + chain (2).
pub fn parse_governance_header(
    env: &Env,
    reader: &mut BytesReader,
) -> Result<(BytesN<32>, u8, u16), WormholeError> {
    let module = reader.read_bytes_n::<32>()?;
    let action = reader.read_u8()?;
    let chain = reader.read_u16_be()?;

    Ok((BytesN::from_array(env, &module.to_array()), action, chain))
}

/// Validates governance header fields against expected values.
///
/// - Module must be "Core" (right-padded)
/// - Action must match the expected action ID
/// - Chain must be 0 (all chains) or 61 (Stellar)
pub fn validate_governance_header(
    module: &BytesN<32>,
    action: u8,
    chain: u16,
    expected_action: u8,
) -> Result<(), WormholeError> {
    if module.to_array() != MODULE_CORE {
        return Err(WormholeError::InvalidGovernanceModule);
    }

    if action != expected_action {
        return Err(WormholeError::InvalidGovernanceAction);
    }

    if chain != 0 && chain != CHAIN_ID_STELLAR {
        return Err(WormholeError::InvalidGovernanceChain);
    }

    Ok(())
}
