#[test_only]
module token_bridge::token_bridge_scenario {
    use std::vector::{Self};
    use sui::package::{UpgradeCap};
    use sui::test_scenario::{Self, Scenario};
    use wormhole::external_address::{Self};
    use wormhole::state::{State as WormholeState};
    use wormhole::wormhole_scenario::{set_up_wormhole, deployer};

    use token_bridge::setup::{Self, DeployerCap};
    use token_bridge::state::{Self, State};

    public fun set_up_wormhole_and_token_bridge(
        scenario: &mut Scenario,
        wormhole_fee: u64
    ) {
        // init and share wormhole core bridge
        set_up_wormhole(scenario, wormhole_fee);

        // Ignore effects.
        test_scenario::next_tx(scenario, deployer());

        // Publish Token Bridge.
        setup::init_test_only(test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, deployer());

        // Finally share `State`.
        let wormhole_state = test_scenario::take_shared<WormholeState>(scenario);
        setup::complete(
            &mut wormhole_state,
            test_scenario::take_from_sender<DeployerCap>(scenario),
            test_scenario::take_from_sender<UpgradeCap>(scenario),
            test_scenario::ctx(scenario)
        );

        // Clean up.
        test_scenario::return_shared<WormholeState>(wormhole_state);
    }

    /// Register arbitrary chain ID with the same emitter address (0xdeadbeef).
    public fun register_dummy_emitter(scenario: &mut Scenario, chain: u16) {
        // Ignore effects.
        test_scenario::next_tx(scenario, person());

        let token_bridge_state = take_state(scenario);
        state::register_new_emitter_test_only(
            &mut token_bridge_state,
            chain,
            external_address::from_bytes(
                x"00000000000000000000000000000000000000000000000000000000deadbeef"
            )
        );

        // Clean up.
        return_state(token_bridge_state);
    }

    /// Register 0xdeadbeef for multiple chains.
    public fun register_dummy_emitters(
        scenario: &mut Scenario,
        chains: vector<u16>
    ) {
        while (!vector::is_empty(&chains)) {
            register_dummy_emitter(scenario, vector::pop_back(&mut chains));
        };
        vector::destroy_empty(chains);
    }

    public fun person(): address {
        wormhole::wormhole_scenario::person()
    }

    public fun two_people(): (address, address) {
        wormhole::wormhole_scenario::two_people()
    }

    public fun three_people(): (address, address, address) {
        wormhole::wormhole_scenario::three_people()
    }

    public fun take_state(scenario: &Scenario): State {
        test_scenario::take_shared(scenario)
    }

    public fun return_state(token_bridge_state: State) {
        test_scenario::return_shared(token_bridge_state);
    }

    public fun take_states(scenario: &Scenario): (State, WormholeState) {
        (
            test_scenario::take_shared<State>(scenario),
            test_scenario::take_shared<WormholeState>(scenario)
        )
    }

    public fun return_states(
        token_bridge_state: State,
        worm_state: WormholeState
    ) {
        return_state(token_bridge_state);
        wormhole::wormhole_scenario::return_state(worm_state);
    }
}
