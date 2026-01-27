//! Contract initialization logic.
//!
//! Handles setup of the Wormhole Core contract via the `__constructor` function,
//! including guardian set creation, admin configuration, and governance emitter storage.
//!
//! This module is only called by the constructor, which is executed atomically
//! during deployment (Protocol 22+). The runtime guarantees single execution.

use crate::{
    governance::guardian_set::{set_current_index, store},
    storage::StorageKey,
};
use soroban_sdk::{BytesN, Env, Vec, contractevent};
use wormhole_soroban_client::*;

/// Emitted once when the contract is successfully initialized.
///
/// Guardians and indexers can observe this event to confirm deployment.
#[contractevent(topics = ["wormhole_core", "init"])]
struct InitializeEvent {
    /// Wormhole chain ID for this deployment (61 for Stellar).
    chain_id: u32,
    /// Number of guardians in the initial set.
    guardian_count: u32,
    /// Chain ID of the governance source (1 for Solana).
    governance_chain_id: u32,
    /// Address authorized to emit governance VAAs.
    governance_emitter: BytesN<32>,
}

/// Internal initialization logic called by `__constructor`.
///
/// Sets up the initial guardian set (index 0), configures the contract as its own
/// admin for upgrades, and stores the governance emitter address.
///
/// # Arguments
/// * `initial_guardians` - Ethereum addresses (20 bytes) of initial guardians
/// * `governance_emitter` - 32-byte address authorized to emit governance VAAs
///
/// # Panics
/// Panics with `EmptyGuardianSet` if no guardians are provided.
pub(crate) fn initialize_internal(
    env: &Env,
    initial_guardians: Vec<BytesN<20>>,
    governance_emitter: BytesN<32>,
) {
    // Validate initial guardians - panic on failure (constructor semantics)
    if initial_guardians.is_empty() {
        env.panic_with_error(WormholeError::EmptyGuardianSet);
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

    // Panic on storage failure (constructor semantics)
    if let Err(e) = store(env, 0, guardian_set) {
        env.panic_with_error(e);
    }

    set_current_index(env, 0);

    // Emit initialization event
    InitializeEvent {
        chain_id: u32::from(CHAIN_ID_STELLAR),
        guardian_count: initial_guardians.len(),
        governance_chain_id: GOVERNANCE_CHAIN_ID,
        governance_emitter: governance_emitter.clone(),
    }
    .publish(env);
}
