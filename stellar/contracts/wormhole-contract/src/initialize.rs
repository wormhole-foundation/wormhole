//! Contract initialization logic.
//!
//! Handles setup of the Wormhole Core contract via the `__constructor` function,
//! including guardian set creation and governance emitter validation.
//!
//! This module is only called by the constructor, which is executed atomically
//! during deployment (Protocol 22+). The runtime guarantees single execution.

use crate::governance::guardian_set::{set_current_index, store};
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
/// Sets up the initial guardian set (index 0) and validates the governance
/// emitter against the protocol constant.
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

    // Reject deployment with wrong governance emitter
    if governance_emitter.to_array() != GOVERNANCE_EMITTER {
        env.panic_with_error(WormholeError::InvalidGovernanceEmitter);
    }

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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{Wormhole, WormholeClient};
    use soroban_sdk::{IntoVal, Symbol, Val, map, testutils::Events, vec};

    #[test]
    fn test_constructor_initializes_state() {
        let env = Env::default();
        let guardian = BytesN::from_array(&env, &[0u8; 20]);
        let initial_guardians = vec![&env, guardian.clone()];
        let governance_emitter = BytesN::from_array(&env, &GOVERNANCE_EMITTER);
        let contract_id = env.register(
            Wormhole,
            (initial_guardians.clone(), governance_emitter.clone()),
        );
        let client = WormholeClient::new(&env, &contract_id);

        assert_eq!(client.get_current_guardian_set_index(), 0);
        assert_eq!(
            client.get_governance_emitter(),
            BytesN::from_array(&env, &GOVERNANCE_EMITTER)
        );
        let guardian_set = client.get_guardian_set(&0);
        assert_eq!(guardian_set.keys.len(), 1);
        assert_eq!(guardian_set.keys.get(0).unwrap(), guardian);

        assert_eq!(guardian_set.creation_time, env.ledger().timestamp());
    }

    #[test]
    fn test_constructor_preserves_multiple_guardians_order() {
        let env = Env::default();
        let guardian_1 = BytesN::from_array(&env, &[1u8; 20]);
        let guardian_2 = BytesN::from_array(&env, &[2u8; 20]);
        let guardian_3 = BytesN::from_array(&env, &[3u8; 20]);
        let initial_guardians = vec![
            &env,
            guardian_1.clone(),
            guardian_2.clone(),
            guardian_3.clone(),
        ];
        let governance_emitter = BytesN::from_array(&env, &GOVERNANCE_EMITTER);
        let contract_id = env.register(Wormhole, (initial_guardians, governance_emitter));
        let client = WormholeClient::new(&env, &contract_id);

        let guardian_set = client.get_guardian_set(&0);
        assert_eq!(guardian_set.keys.len(), 3);
        assert_eq!(guardian_set.keys.get(0).unwrap(), guardian_1);
        assert_eq!(guardian_set.keys.get(1).unwrap(), guardian_2);
        assert_eq!(guardian_set.keys.get(2).unwrap(), guardian_3);
    }

    #[test]
    #[should_panic(expected = "Error(Contract, #35)")]
    fn test_constructor_rejects_empty_guardians() {
        let env = Env::default();
        let initial_guardians: Vec<BytesN<20>> = Vec::new(&env);
        let governance_emitter = BytesN::from_array(&env, &GOVERNANCE_EMITTER);

        let _ = env.register(Wormhole, (initial_guardians, governance_emitter));
    }

    #[test]
    fn test_constructor_emits_single_init_event() {
        let env = Env::default();
        let guardian = BytesN::from_array(&env, &[0u8; 20]);
        let initial_guardians = vec![&env, guardian.clone()];
        let governance_emitter = BytesN::from_array(&env, &GOVERNANCE_EMITTER);
        let contract_id = env.register(Wormhole, (initial_guardians, governance_emitter));

        let all_events = env.events().all();
        let contract_events = all_events.filter_by_contract(&contract_id);
        assert_eq!(contract_events.events().len(), 1);
        let chain_id_val: Val = 61u32.into_val(&env);
        let governance_chain_id_val: Val = 1u32.into_val(&env);
        let governance_emitter_val: Val =
            BytesN::from_array(&env, &GOVERNANCE_EMITTER).into_val(&env);
        let guardian_count_val: Val = 1u32.into_val(&env);

        assert_eq!(
            contract_events,
            vec![
                &env,
                (
                    contract_id.clone(),
                    vec![
                        &env,
                        Symbol::new(&env, "wormhole_core").into_val(&env),
                        Symbol::new(&env, "init").into_val(&env),
                    ],
                    map![
                        &env,
                        (Symbol::new(&env, "chain_id"), chain_id_val),
                        (
                            Symbol::new(&env, "governance_chain_id"),
                            governance_chain_id_val
                        ),
                        (
                            Symbol::new(&env, "governance_emitter"),
                            governance_emitter_val
                        ),
                        (Symbol::new(&env, "guardian_count"), guardian_count_val)
                    ]
                    .into_val(&env),
                )
            ]
        );
    }
}
