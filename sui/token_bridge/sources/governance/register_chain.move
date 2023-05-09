// SPDX-License-Identifier: Apache 2

/// This module implements handling a governance VAA to enact registering a
/// foreign Token Bridge for a particular chain ID.
module token_bridge::register_chain {
    use sui::table::{Self};
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::governance_message::{Self, DecreeTicket, DecreeReceipt};

    use token_bridge::state::{Self, State, LatestOnly};

    /// Cannot register chain ID == 0.
    const E_INVALID_EMITTER_CHAIN: u64 = 0;
    /// Emitter already exists for a given chain ID.
    const E_EMITTER_ALREADY_REGISTERED: u64 = 1;

    /// Specific governance payload ID (action) for registering foreign Token
    /// Bridge contract address.
    const ACTION_REGISTER_CHAIN: u8 = 1;

    struct GovernanceWitness has drop {}

    struct RegisterChain {
        chain: u16,
        contract_address: ExternalAddress,
    }

    public fun authorize_governance(
        token_bridge_state: &State
    ): DecreeTicket<GovernanceWitness> {
        governance_message::authorize_verify_global(
            GovernanceWitness {},
            state::governance_chain(token_bridge_state),
            state::governance_contract(token_bridge_state),
            state::governance_module(),
            ACTION_REGISTER_CHAIN
        )
    }

    public fun register_chain(
        token_bridge_state: &mut State,
        receipt: DecreeReceipt<GovernanceWitness>
    ): (
        u16,
        ExternalAddress
    ) {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        let payload =
            governance_message::take_payload(
                state::borrow_mut_consumed_vaas(
                    &latest_only,
                    token_bridge_state
                ),
                receipt
            );

        handle_register_chain(&latest_only, token_bridge_state, payload)
    }

    fun handle_register_chain(
        latest_only: &LatestOnly,
        token_bridge_state: &mut State,
        governance_payload: vector<u8>
    ): (
        u16,
        ExternalAddress
    ) {
        // Deserialize the payload as amount to change the Wormhole fee.
        let RegisterChain {
            chain,
            contract_address
        } = deserialize(governance_payload);

        register_new_emitter(
            latest_only,
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

    /// Add a new Token Bridge emitter to the registry. This method will abort
    /// if an emitter is already registered for a particular chain ID.
    ///
    /// See `register_chain` module for more info.
    fun register_new_emitter(
        latest_only: &LatestOnly,
        token_bridge_state: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        assert!(chain != 0, E_INVALID_EMITTER_CHAIN);

        let registry =
            state::borrow_mut_emitter_registry(latest_only, token_bridge_state);
        assert!(
            !table::contains(registry, chain),
            E_EMITTER_ALREADY_REGISTERED
        );
        table::add(registry, chain, contract_address);
    }

    #[test_only]
    public fun register_new_emitter_test_only(
        token_bridge_state: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        register_new_emitter(
            &latest_only,
            token_bridge_state,
            chain,
            contract_address
        );
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
    use wormhole::wormhole_scenario::{
        parse_and_verify_vaa,
        verify_governance_vaa
    };

    use token_bridge::register_chain::{Self};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        person,
        return_state,
        set_up_wormhole_and_token_bridge,
        take_state
    };

    const VAA_REGISTER_CHAIN_1: vector<u8> =
        x"01000000000100dd8cf046ad6dd17b2b5130d236b3545350899ac33b5c9e93e4d8c3e0da718a351c3f76cb9ddb15a0f0d7db7b1dded2b5e79c2f6e76dde6d8ed4bcb9cb461eb480100bc614e0000000000010000000000000000000000000000000000000000000000000000000000000004000000000000000101000000000000000000000000000000000000000000546f6b656e4272696467650100000002000000000000000000000000deadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
    const VAA_REGISTER_SAME_CHAIN: vector<u8> =
        x"01000000000100847ca782db7616135de4a835ed5b12ba7946bbd39f70ecd9912ec55bdc9cb6c6215c98d6ad5c8d7253c2bb0fb0f8df0dc6591408c366cf0c09e58abcfb8c0abe0000bc614e0000000000010000000000000000000000000000000000000000000000000000000000000004000000000000000101000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deafbeef";

    #[test]
    fun test_register_chain() {
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

        let token_bridge_state = take_state(scenario);

        // Check that the emitter is not registered.
        let expected_chain = 2;
        {
            let registry = state::borrow_emitter_registry(&token_bridge_state);
            assert!(!table::contains(registry, expected_chain), 0);
        };

        let verified_vaa = parse_and_verify_vaa(scenario, VAA_REGISTER_CHAIN_1);
        let ticket = register_chain::authorize_governance(&token_bridge_state);
        let receipt =
            verify_governance_vaa(scenario, verified_vaa, ticket);
        let (
            chain,
            contract_address
        ) = register_chain(&mut token_bridge_state, receipt);
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
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = register_chain::E_EMITTER_ALREADY_REGISTERED)]
    fun test_cannot_register_chain_already_registered() {
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

        let token_bridge_state = take_state(scenario);

        let verified_vaa = parse_and_verify_vaa(scenario, VAA_REGISTER_CHAIN_1);
        let ticket = register_chain::authorize_governance(&token_bridge_state);
        let receipt =
            verify_governance_vaa(scenario, verified_vaa, ticket);
        let (
            chain,
            _
        ) = register_chain(&mut token_bridge_state, receipt);

        // Check registry.
        let expected_contract =
            *table::borrow(
                state::borrow_emitter_registry(&token_bridge_state),
                chain
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let verified_vaa =
            parse_and_verify_vaa(scenario, VAA_REGISTER_SAME_CHAIN);
        let payload =
            governance_message::take_decree(
                wormhole::vaa::payload(&verified_vaa)
            );
        let cur = cursor::new(payload);

        // Show this payload is attempting to register the same chain ID.
        let another_chain = bytes::take_u16_be(&mut cur);
        assert!(chain == another_chain, 0);

        let another_contract = external_address::take_bytes(&mut cur);
        assert!(another_contract != expected_contract, 0);

        // No more payload to read.
        cursor::destroy_empty(cur);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let ticket = register_chain::authorize_governance(&token_bridge_state);
        let receipt =
            verify_governance_vaa(scenario, verified_vaa, ticket);

        // You shall not pass!
        register_chain(&mut token_bridge_state, receipt);

        abort 42
    }
}




