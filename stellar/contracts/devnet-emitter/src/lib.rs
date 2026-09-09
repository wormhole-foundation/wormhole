//! Devnet emitter: the smallest Soroban contract that publishes Wormhole messages through
//! the core contract. The Tilt smoke test (`stellar/scripts/devnet_smoke_test.sh`) deploys
//! it and calls `send`, because the core contract only accepts contract emitters.
#![no_std]

use soroban_sdk::{Address, Bytes, Env, contract, contractimpl, contracttype};
use wormhole_soroban_client::{ConsistencyLevel, WormholeClient};

#[contracttype]
#[derive(Clone)]
enum DataKey {
    Core,
}

#[contract]
pub struct DevnetEmitter;

#[contractimpl]
impl DevnetEmitter {
    /// Stores the Wormhole core contract this emitter publishes through.
    pub fn __constructor(env: Env, core: Address) {
        env.storage().instance().set(&DataKey::Core, &core);
    }

    /// Publishes `payload` through the core contract with this contract as the emitter and
    /// returns the message sequence assigned by the core.
    pub fn send(env: Env, nonce: u32, payload: Bytes, consistency_level: ConsistencyLevel) -> u64 {
        let core: Address = env
            .storage()
            .instance()
            .get(&DataKey::Core)
            .expect("core address not set");
        WormholeClient::new(&env, &core).post_message(
            &env.current_contract_address(),
            &nonce,
            &payload,
            &consistency_level,
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use soroban_sdk::{BytesN, testutils::Events, vec};
    use wormhole_contract::Wormhole;
    use wormhole_soroban_client::GOVERNANCE_EMITTER;

    /// Registers an initialized core contract and an emitter pointing at it.
    fn deploy(env: &Env) -> (Address, Address) {
        let guardians = vec![env, BytesN::from_array(env, &[0u8; 20])];
        let governance_emitter = BytesN::from_array(env, &GOVERNANCE_EMITTER);
        let core = env.register(Wormhole, (guardians, governance_emitter));
        let emitter = env.register(DevnetEmitter, (core.clone(),));
        (core, emitter)
    }

    #[test]
    fn send_publishes_through_core_with_emitter_contract_as_sender() {
        // No mock_all_auths on purpose: the core calls `emitter.require_auth()`, which must be
        // satisfied by the emitter contract being the direct invoker, exactly as on a real network.
        let env = Env::default();
        let (core, emitter) = deploy(&env);
        let client = DevnetEmitterClient::new(&env, &emitter);
        let payload = Bytes::from_array(&env, &[0xAB, 0xCD]);

        assert_eq!(client.send(&1, &payload, &ConsistencyLevel::Confirmed), 0);

        let core_events = env.events().all().filter_by_contract(&core);
        assert_eq!(core_events.events().len(), 1);

        assert_eq!(client.send(&2, &payload, &ConsistencyLevel::Finalized), 1);

        let core_client = WormholeClient::new(&env, &core);
        assert_eq!(core_client.get_emitter_sequence(&emitter), 2);
    }
}
