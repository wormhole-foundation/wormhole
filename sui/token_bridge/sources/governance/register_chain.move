// SPDX-License-Identifier: Apache 2

/// This module implements handling a governance VAA to enact registering a
/// foreign Token Bridge for a particular chain ID.
module token_bridge::register_chain {
    use wormhole::bytes::{Self};
    use wormhole::consumed_vaas::{Self};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::governance_message::{Self, GovernanceMessage};

    use token_bridge::state::{Self, State};
    use token_bridge::version_control::{RegisterChain as RegisterChainControl};

    /// Specific governance payload ID (action) for registering foreign Token
    /// Bridge contract address.
    const ACTION_REGISTER_CHAIN: u8 = 1;

    struct RegisterChain {
        chain: u16,
        contract_address: ExternalAddress,
    }

    public fun register_chain(
        token_bridge_state: &mut State,
        msg: GovernanceMessage
    ): (u16, ExternalAddress) {
        state::check_minimum_requirement<RegisterChainControl>(
            token_bridge_state
        );

        // Protect against replaying the VAA.
        consumed_vaas::consume(
            state::borrow_mut_consumed_vaas(token_bridge_state),
            governance_message::vaa_hash(&msg)
        );

        handle_register_chain(token_bridge_state, msg)
    }

    fun handle_register_chain(
        token_bridge_state: &mut State,
        msg: GovernanceMessage
    ): (u16, ExternalAddress) {
        // Verify that this governance message is to update the Wormhole fee.
        let governance_payload =
            governance_message::take_global_action(
                msg,
                state::governance_module(),
                ACTION_REGISTER_CHAIN
            );

        // Deserialize the payload as amount to change the Wormhole fee.
        let RegisterChain {
            chain,
            contract_address
        } = deserialize(governance_payload);

        state::register_new_emitter(
            token_bridge_state,
            chain,
            contract_address
        );

        (chain, contract_address)
    }

    fun deserialize(payload: vector<u8>): RegisterChain {
        let cur = cursor::new(payload);

        // This amount cannot be greater than max u64.
        let chain = bytes::take_u16_be(&mut cur);
        let contract_address = external_address::take_bytes(&mut cur);

        cursor::destroy_empty(cur);

        RegisterChain { chain, contract_address}
    }

    #[test_only]
    public fun action(): u8 {
        ACTION_REGISTER_CHAIN
    }
}

#[test_only]
module token_bridge::register_chain_tests {
    use sui::table::{Self};
    use sui::test_scenario::{Self};
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self};
    use wormhole::governance_message::{Self};
    use wormhole::vaa::{Self};

    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        person,
        return_clock,
        return_states,
        set_up_wormhole_and_token_bridge,
        take_clock,
        take_states
    };

    const VAA_REGISTER_CHAIN_1: vector<u8> =
        x"01000000000100dd8cf046ad6dd17b2b5130d236b3545350899ac33b5c9e93e4d8c3e0da718a351c3f76cb9ddb15a0f0d7db7b1dded2b5e79c2f6e76dde6d8ed4bcb9cb461eb480100bc614e0000000000010000000000000000000000000000000000000000000000000000000000000004000000000000000101000000000000000000000000000000000000000000546f6b656e4272696467650100000002000000000000000000000000deadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
    const VAA_REGISTER_SAME_CHAIN: vector<u8> =
        x"01000000000100847ca782db7616135de4a835ed5b12ba7946bbd39f70ecd9912ec55bdc9cb6c6215c98d6ad5c8d7253c2bb0fb0f8df0dc6591408c366cf0c09e58abcfb8c0abe0000bc614e0000000000010000000000000000000000000000000000000000000000000000000000000004000000000000000101000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deafbeef";

    #[test]
    public fun test_register_chain() {
        // Testing this method.
        use token_bridge::register_chain::{register_chain};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Prepare test to execute `set_fee`.
        test_scenario::next_tx(scenario, caller);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Check that the emitter is not registered.
        let expected_chain = 2;
        {
            let registry = state::borrow_emitter_registry(&token_bridge_state);
            assert!(!table::contains(registry, expected_chain), 0);
        };

        let parsed =
            vaa::parse_and_verify(
                &worm_state,
                VAA_REGISTER_CHAIN_1,
                &the_clock
            );
        let msg = governance_message::verify_vaa(&worm_state, parsed);
        let (
            chain,
            contract_address
        ) = register_chain(&mut token_bridge_state, msg);
        assert!(chain == expected_chain, 0);

        let expected_contract =
            external_address::from_address(
                @0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef
            );
        assert!(contract_address == expected_contract, 0);
        {
            let registry = state::borrow_emitter_registry(&token_bridge_state);
            assert!(*table::borrow(registry, expected_chain) == expected_contract, 0);
        };

        // Clean up.
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = state::E_EMITTER_ALREADY_REGISTERED)]
    public fun test_cannot_register_chain_already_registered() {
        // Testing this method.
        use token_bridge::register_chain::{register_chain};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Prepare test to execute `set_fee`.
        test_scenario::next_tx(scenario, caller);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        let parsed =
            vaa::parse_and_verify(
                &worm_state,
                VAA_REGISTER_CHAIN_1,
                &the_clock
            );
        let msg = governance_message::verify_vaa(&worm_state, parsed);
        let (
            chain,
            _
        ) = register_chain(&mut token_bridge_state, msg);

        // Check registry.
        let expected_contract =
            *table::borrow(
                state::borrow_emitter_registry(&token_bridge_state),
                chain
            );

        let payload =
            governance_message::take_payload(
                governance_message::verify_vaa(
                    &worm_state,
                    vaa::parse_and_verify(
                        &worm_state,
                        VAA_REGISTER_SAME_CHAIN,
                        &the_clock
                    )
                )
            );
        let cur = cursor::new(payload);

        // Show this payload is attempting to register the same chain ID.
        let another_chain = bytes::take_u16_be(&mut cur);
        assert!(chain == another_chain, 0);

        let another_contract = external_address::take_bytes(&mut cur);
        assert!(another_contract != expected_contract, 0);

        let parsed =
            vaa::parse_and_verify(
                &worm_state,
                VAA_REGISTER_SAME_CHAIN,
                &the_clock
            );
        let msg = governance_message::verify_vaa(&worm_state, parsed);

        // You shall not pass!
        register_chain(&mut token_bridge_state, msg);

        // Clean up.
        cursor::destroy_empty(cur);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }
}




