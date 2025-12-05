use crate::{initialize, storage::StorageKey, vaa::verify_vaa};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env};
use wormhole_soroban_client::{
    VAA, BytesReader, WormholeError, CHAIN_ID_STELLAR, GOVERNANCE_CHAIN_ID, GOVERNANCE_EMITTER,
    MODULE_CORE, STORAGE_TTL_EXTENSION, STORAGE_TTL_THRESHOLD,
};

pub trait GovernanceAction {
    type Payload: for<'a> TryFrom<(&'a Env, &'a Bytes), Error = WormholeError>;

    fn validate_payload(env: &Env, payload: &Self::Payload) -> Result<(), WormholeError>;
    fn execute(env: &Env, vaa: &VAA, payload: &Self::Payload) -> Result<(), WormholeError>;

    fn submit(env: &Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
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

fn verify_governance_vaa(vaa: &VAA) -> Result<(), WormholeError> {
    if vaa.emitter_chain != GOVERNANCE_CHAIN_ID {
        return Err(WormholeError::InvalidGovernanceChain);
    }

    if vaa.emitter_address.to_array() != GOVERNANCE_EMITTER {
        return Err(WormholeError::InvalidGovernanceEmitter);
    }

    Ok(())
}

fn verify_and_hash_governance_vaa(
    env: &Env,
    vaa_bytes: &Bytes,
) -> Result<(VAA, BytesN<32>), WormholeError> {
    let vaa = VAA::try_from((env, vaa_bytes))?;
    verify_governance_vaa(&vaa)?;
    verify_vaa(env, vaa_bytes)?;

    let body_bytes = vaa.serialize_body(env);
    let vaa_hash = crate::utils::keccak256_hash(env, &body_bytes);

    Ok((vaa, vaa_hash))
}

fn is_vaa_consumed(env: &Env, hash: &BytesN<32>) -> bool {
    env.storage()
        .persistent()
        .has(&StorageKey::ConsumedGovernanceVAA(hash.clone()))
}

fn require_vaa_not_consumed(env: &Env, hash: &BytesN<32>) -> Result<(), WormholeError> {
    if is_vaa_consumed(env, hash) {
        return Err(WormholeError::GovernanceVAAAlreadyConsumed);
    }
    Ok(())
}

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

pub fn is_governance_vaa_consumed_from_bytes(
    env: &Env,
    vaa_bytes: &Bytes,
) -> Result<bool, WormholeError> {
    let body_bytes = VAA::get_body_bytes(vaa_bytes)?;
    let vaa_hash = crate::utils::keccak256_hash(env, &body_bytes);
    Ok(is_vaa_consumed(env, &vaa_hash))
}

pub fn parse_governance_header(
    env: &Env,
    reader: &mut BytesReader,
) -> Result<(BytesN<32>, u8, u16), WormholeError> {
    let module = reader.read_bytes_n::<32>()?;
    let action = reader.read_u8()?;
    let chain = reader.read_u16_be()?;

    Ok((BytesN::from_array(env, &module.to_array()), action, chain))
}

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
