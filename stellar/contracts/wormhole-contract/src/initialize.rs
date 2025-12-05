use crate::{
    governance::guardian_set::{set_current_index, store},
    storage::StorageKey,
};
use soroban_sdk::{BytesN, Env, Vec, contractevent};
use wormhole_soroban_client::*;

/// Event published when the contract is initialized.
///
/// Topics: ["wormhole_core", "init"]
/// - "wormhole_core": Namespace for all core contract governance/lifecycle events
/// - "init": Event type for initialization
#[contractevent(topics = ["wormhole_core", "init"])]
struct InitializeEvent {
    chain_id: u32,
    guardian_count: u32,
    governance_chain_id: u32,
    governance_emitter: BytesN<32>,
}

pub fn is_initialized(env: &Env) -> bool {
    env.storage()
        .persistent()
        .get(&StorageKey::Initialized)
        .unwrap_or(false)
}

/// Ensures the contract is initialized, returning an Error if not.
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

pub fn initialize(
    env: &Env,
    initial_guardians: Vec<BytesN<20>>,
    governance_emitter: BytesN<32>,
) -> Result<(), WormholeError> {
    if is_initialized(env) {
        return Err(WormholeError::AlreadyInitialized);
    }

    // Validate initial guardians
    if initial_guardians.is_empty() {
        return Err(WormholeError::EmptyGuardianSet);
    }

    // Set contract as its own admin for upgrades
    let contract_address = env.current_contract_address();
    env.storage()
        .instance()
        .set(&StorageKey::Admin, &contract_address);

    // Store governance emitter address
    env.storage()
        .persistent()
        .set(&StorageKey::GovernanceEmitter, &governance_emitter);

    // Extend TTL for governance emitter (it's permanent config)
    env.storage().persistent().extend_ttl(
        &StorageKey::GovernanceEmitter,
        STORAGE_TTL_THRESHOLD,
        STORAGE_TTL_EXTENSION,
    );

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
        governance_chain_id: GOVERNANCE_CHAIN_ID,
        governance_emitter: governance_emitter.clone(),
    }
    .publish(env);

    Ok(())
}
