// SPDX-License-Identifier: Apache 2

#[test_only]
module token_bridge::token_bridge_scenario {
    use std::vector::{Self};
    use sui::balance::{Self};
    use sui::package::{UpgradeCap};
    use sui::test_scenario::{Self, Scenario};
    use wormhole::external_address::{Self};
    use wormhole::wormhole_scenario::{
        deployer,
        return_state as return_wormhole_state,
        set_up_wormhole,
        take_state as take_wormhole_state
    };

    use token_bridge::native_asset::{Self};
    use token_bridge::setup::{Self, DeployerCap};
    use token_bridge::state::{Self, State};
    use token_bridge::token_registry::{Self};

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

        let wormhole_state = take_wormhole_state(scenario);

        let upgrade_cap =
            test_scenario::take_from_sender<UpgradeCap>(scenario);
        let emitter_cap =
            wormhole::emitter::new(
                &wormhole_state,
                test_scenario::ctx(scenario)
            );
        let governance_chain = 1;
        let governance_contract =
            x"0000000000000000000000000000000000000000000000000000000000000004";

        // Finally share `State`.
        setup::complete(
            test_scenario::take_from_sender<DeployerCap>(scenario),
            upgrade_cap,
            emitter_cap,
            governance_chain,
            governance_contract,
            test_scenario::ctx(scenario)
        );

        // Clean up.
        return_wormhole_state(wormhole_state);
    }

    /// Perform an upgrade (which just upticks the current version of what the
    /// `State` believes is true).
    public fun upgrade_token_bridge(scenario: &mut Scenario) {
        // Clean up from activity prior.
        test_scenario::next_tx(scenario, person());

        let token_bridge_state = take_state(scenario);
        state::test_upgrade(&mut token_bridge_state);

        // Clean up.
        return_state(token_bridge_state);
    }

    /// Register arbitrary chain ID with the same emitter address (0xdeadbeef).
    public fun register_dummy_emitter(scenario: &mut Scenario, chain: u16) {
        // Ignore effects.
        test_scenario::next_tx(scenario, person());

        let token_bridge_state = take_state(scenario);
        token_bridge::register_chain::register_new_emitter_test_only(
            &mut token_bridge_state,
            chain,
            external_address::from_address(@0xdeadbeef)
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

    public fun deposit_native<CoinType>(
        token_bridge_state: &mut State,
        deposit_amount: u64
    ) {
        native_asset::deposit_test_only(
            token_registry::borrow_mut_native_test_only(
                state::borrow_mut_token_registry_test_only(token_bridge_state)
            ),
            balance::create_for_testing<CoinType>(deposit_amount)
        )
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
}
