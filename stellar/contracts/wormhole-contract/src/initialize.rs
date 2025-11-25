#![allow(missing_docs)]

use crate::{
    constants::*,
    governance::guardian_set::{set_current_index, store},
    storage::StorageKey,
};
use soroban_sdk::{contractevent, BytesN, Env, Vec};
use wormhole_interface::{Error, GuardianSetInfo};

/// Event published when the contract is initialized.
///
/// Topics: ["wormhole_core", "init"]
/// Data: { chain_id: u32, guardian_count: u32 }
#[contractevent(topics = ["wormhole_core", "init"])]
struct InitializeEvent {
    chain_id: u32,
    guardian_count: u32,
}

pub(crate) fn is_initialized(env: &Env) -> bool {
    env.storage()
        .persistent()
        .get(&StorageKey::Initialized)
        .unwrap_or(false)
}

/// Ensures the contract is initialized, returning an error if not.
pub(crate) fn require_initialized(env: &Env) -> Result<(), Error> {
    if !is_initialized(env) {
        return Err(Error::NotInitialized);
    }
    Ok(())
}

fn set_initialized(env: &Env) {
    env.storage()
        .persistent()
        .set(&StorageKey::Initialized, &true);
}

pub(crate) fn initialize(env: &Env, initial_guardians: Vec<BytesN<20>>) -> Result<(), Error> {
    if is_initialized(env) {
        return Err(Error::AlreadyInitialized);
    }

    // Validate initial guardians
    if initial_guardians.is_empty() {
        return Err(Error::EmptyGuardianSet);
    }

    // Set contract as its own admin for upgrades
    let contract_address = env.current_contract_address();
    env.storage().instance().set(&StorageKey::Admin, &contract_address);

    // Create the initial guardian set (always index 0)
    let guardian_set = GuardianSetInfo {
        keys: initial_guardians.clone(),
        creation_time: env.ledger().timestamp(),
    };

    store(env, 0, guardian_set)?;

    set_current_index(env, 0);

    set_initialized(env);

    // Emit initialization event
    InitializeEvent {
        chain_id: u32::from(CHAIN_ID_STELLAR),
        guardian_count: initial_guardians.len(),
    }.publish(env);

    Ok(())
}

