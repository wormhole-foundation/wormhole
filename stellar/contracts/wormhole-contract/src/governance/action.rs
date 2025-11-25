use crate::{constants::*, initialize, storage::StorageKey, utils::BytesReader, vaa::VAA};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env};
use wormhole_interface::Error;

/// Trait for governance actions with a default submit() implementation.
///
/// All governance actions follow the same flow:
/// 1. Verify contract is initialized
/// 2. Parse and verify VAA
/// 3. Check replay protection
/// 4. Parse and validate payload
/// 5. Consume VAA (prevents replay)
/// 6. Execute the action
pub(crate) trait GovernanceAction {
    type Payload: for<'a> TryFrom<(&'a Env, &'a Bytes), Error = Error>;

    fn validate_payload(env: &Env, payload: &Self::Payload) -> Result<(), Error>;
    fn execute(env: &Env, vaa: &VAA, payload: &Self::Payload) -> Result<(), Error>;

    fn submit(env: &Env, vaa_bytes: Bytes) -> Result<(), Error> {
        initialize::require_initialized(env)?;

        let (vaa, vaa_hash) = verify_and_hash_governance_vaa(env, &vaa_bytes)?;
        require_vaa_not_consumed(env, &vaa_hash)?;

        let payload = Self::Payload::try_from((env, &vaa.payload))?;
        Self::validate_payload(env, &payload)?;

        consume_vaa(env, &vaa_hash);

        Self::execute(env, &vaa, &payload)?;

        Ok(())
    }
}

/// Verify this is a valid governance VAA.
fn verify_governance_vaa(vaa: &VAA) -> Result<(), Error> {
    if vaa.emitter_chain != GOVERNANCE_CHAIN_ID {
        return Err(Error::InvalidGovernanceChain);
    }

    if vaa.emitter_address.to_array() != GOVERNANCE_EMITTER {
        return Err(Error::InvalidGovernanceEmitter);
    }

    Ok(())
}

/// Performs the complete governance VAA verification sequence.
///
/// This combines:
/// 1. Parse VAA from bytes
/// 2. Verify it's from the governance source
/// 3. Verify guardian signatures
/// 4. Serialize body and compute hash for replay protection
fn verify_and_hash_governance_vaa(
    env: &Env,
    vaa_bytes: &Bytes,
) -> Result<(VAA, BytesN<32>), Error> {
    let vaa = VAA::try_from((env, vaa_bytes))?;
    verify_governance_vaa(&vaa)?;
    vaa.verify(env)?;

    let body_bytes = vaa.serialize_body(env);
    let vaa_hash = crate::utils::keccak256_hash(env, &body_bytes);

    Ok((vaa, vaa_hash))
}

/// Check if a governance VAA has already been consumed.
fn is_vaa_consumed(env: &Env, hash: &BytesN<32>) -> bool {
    env.storage()
        .persistent()
        .has(&StorageKey::ConsumedGovernanceVAA(hash.clone()))
}

/// Ensures a governance VAA has not been consumed, returning an error if it has.
fn require_vaa_not_consumed(env: &Env, hash: &BytesN<32>) -> Result<(), Error> {
    if is_vaa_consumed(env, hash) {
        return Err(Error::GovernanceVAAAlreadyConsumed);
    }
    Ok(())
}

/// Mark a governance VAA as consumed for replay protection.
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

/// Public wrapper for checking VAA consumption from bytes.
pub(crate) fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<bool, Error> {
    let body_bytes = VAA::get_body_bytes(&vaa_bytes)?;
    let vaa_hash = crate::utils::keccak256_hash(&env, &body_bytes);
    Ok(is_vaa_consumed(&env, &vaa_hash))
}

/// Parse common governance payload header (module, action, chain).
pub(crate) fn parse_governance_header(
    env: &Env,
    reader: &mut BytesReader,
) -> Result<(BytesN<32>, u8, u16), Error> {
    let module = reader.read_bytes_n::<32>(env)?;
    let action = reader.read_u8()?;
    let chain = reader.read_u16_be()?;
    Ok((module, action, chain))
}

/// Validate common governance header fields.
pub(crate) fn validate_governance_header(
    module: &BytesN<32>,
    action: u8,
    chain: u16,
    expected_action: u8,
) -> Result<(), Error> {
    if module.to_array() != MODULE_CORE {
        return Err(Error::InvalidGovernanceModule);
    }

    if action != expected_action {
        return Err(Error::InvalidGovernanceAction);
    }

    if chain != 0 && chain != CHAIN_ID_STELLAR {
        return Err(Error::InvalidGovernanceChain);
    }

    Ok(())
}
