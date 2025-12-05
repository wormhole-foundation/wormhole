use crate::governance::guardian_set::{set_current_index, store};
use crate::storage::StorageKey;
use soroban_sdk::{BytesN, Env, Vec};
use wormhole_soroban_client::{GuardianSetInfo, WormholeError, CHAIN_ID_STELLAR};

pub fn is_initialized(env: &Env) -> bool {
    env.storage()
        .persistent()
        .get(&StorageKey::Initialized)
        .unwrap_or(false)
}

pub fn require_initialized(env: &Env) -> Result<(), WormholeError> {
    if !is_initialized(env) {
        return Err(WormholeError::NotInitialized);
    }
    Ok(())
}

fn set_initialized(env: &Env) {
    env.storage()
        .persistent()
        .set(&StorageKey::Initialized, &true);
}

pub fn initialize(env: &Env, initial_guardians: Vec<BytesN<20>>) -> Result<(), WormholeError> {
    if is_initialized(env) {
        return Err(WormholeError::AlreadyInitialized);
    }

    if initial_guardians.is_empty() {
        return Err(WormholeError::EmptyGuardianSet);
    }

    let contract_address = env.current_contract_address();
    env.storage()
        .instance()
        .set(&StorageKey::Admin, &contract_address);

    let guardian_set = GuardianSetInfo {
        keys: initial_guardians.clone(),
        creation_time: env.ledger().timestamp(),
    };

    store(env, 0, guardian_set)?;
    set_current_index(env, 0);
    set_initialized(env);

    env.events().publish(
        ("wormhole_core", "init"),
        (u32::from(CHAIN_ID_STELLAR), initial_guardians.len()),
    );

    Ok(())
}
