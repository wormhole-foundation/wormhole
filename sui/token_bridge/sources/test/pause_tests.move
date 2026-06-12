// SPDX-License-Identifier: Apache 2

#[test_only]
module token_bridge::pause_tests {
    use std::option::{Self};
    use std::vector::{Self};
    use sui::test_scenario::{Self};

    use token_bridge::pause::{Self, PauserCap, UnpauserCap};
    use token_bridge::set_pauser_addresses::{Self};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        person,
        return_state,
        set_up_wormhole_and_token_bridge,
        take_state,
        three_people
    };

    // ========================================================================
    //  Pause State Initialization
    // ========================================================================

    #[test]
    fun test_default_pause_state() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);

        // Initialize pause state (simulating migration).
        state::init_pause_state_test_only(&mut token_bridge_state);

        // Default state: not paused, pauser/unpauser unassigned (none).
        assert!(!state::is_paused(&token_bridge_state), 0);
        assert!(option::is_none(&state::pauser(&token_bridge_state)), 0);
        assert!(option::is_none(&state::unpauser(&token_bridge_state)), 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_is_paused_returns_false_before_init() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);

        // Before pause state init, is_paused returns false (backwards compat).
        assert!(!state::is_paused(&token_bridge_state), 0);
        assert!(option::is_none(&state::pauser(&token_bridge_state)), 0);
        assert!(option::is_none(&state::unpauser(&token_bridge_state)), 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  Set Pauser Addresses (mint + transfer to owner, record cap id)
    // ========================================================================

    #[test]
    fun test_set_pauser_addresses_mints_and_records() {
        let (caller, pauser_owner, unpauser_owner) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        let (pauser_id, unpauser_id) =
            set_pauser_addresses::set_pauser_addresses_test_only(
                &mut token_bridge_state,
                option::some(pauser_owner),
                option::some(unpauser_owner),
                test_scenario::ctx(scenario)
            );

        // Recorded ids are the minted caps' ids (present).
        assert!(option::is_some(&pauser_id), 0);
        assert!(option::is_some(&unpauser_id), 0);
        assert!(state::pauser(&token_bridge_state) == pauser_id, 0);
        assert!(state::unpauser(&token_bridge_state) == unpauser_id, 0);

        return_state(token_bridge_state);

        // Caps were transferred to the owners.
        test_scenario::next_tx(scenario, pauser_owner);
        let pauser_cap = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);
        assert!(option::contains(&pauser_id, &pause::pauser_cap_id(&pauser_cap)), 0);
        test_scenario::return_to_address(pauser_owner, pauser_cap);

        test_scenario::next_tx(scenario, unpauser_owner);
        let unpauser_cap =
            test_scenario::take_from_address<UnpauserCap>(scenario, unpauser_owner);
        assert!(option::contains(&unpauser_id, &pause::unpauser_cap_id(&unpauser_cap)), 0);
        test_scenario::return_to_address(unpauser_owner, unpauser_cap);

        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_set_pauser_to_none_unassigns() {
        let (caller, pauser_owner, unpauser_owner) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        // Assign owners.
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::some(unpauser_owner),
            test_scenario::ctx(scenario)
        );

        // Unassign by setting owners to none (mints nothing).
        let (pauser_id, unpauser_id) =
            set_pauser_addresses::set_pauser_addresses_test_only(
                &mut token_bridge_state,
                option::none(),
                option::none(),
                test_scenario::ctx(scenario)
            );

        assert!(option::is_none(&pauser_id), 0);
        assert!(option::is_none(&unpauser_id), 0);
        assert!(option::is_none(&state::pauser(&token_bridge_state)), 0);
        assert!(option::is_none(&state::unpauser(&token_bridge_state)), 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  Pause / Unpause (capability-gated)
    // ========================================================================

    #[test]
    fun test_pauser_cap_can_pause() {
        let (caller, pauser_owner, _u) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        // The owner retrieves its cap and pauses.
        test_scenario::next_tx(scenario, pauser_owner);
        let token_bridge_state = take_state(scenario);
        let pauser_cap = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);

        pause::pause(
            &mut token_bridge_state,
            &pauser_cap,
            test_scenario::ctx(scenario)
        );

        assert!(state::is_paused(&token_bridge_state), 0);

        test_scenario::return_to_address(pauser_owner, pauser_cap);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_unpauser_cap_can_unpause() {
        let (caller, _p, unpauser_owner) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::none(),
            option::some(unpauser_owner),
            test_scenario::ctx(scenario)
        );
        state::set_paused_test_only(&mut token_bridge_state, true);
        assert!(state::is_paused(&token_bridge_state), 0);
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, unpauser_owner);
        let token_bridge_state = take_state(scenario);
        let unpauser_cap =
            test_scenario::take_from_address<UnpauserCap>(scenario, unpauser_owner);

        pause::unpause(
            &mut token_bridge_state,
            &unpauser_cap,
            test_scenario::ctx(scenario)
        );

        assert!(!state::is_paused(&token_bridge_state), 0);

        test_scenario::return_to_address(unpauser_owner, unpauser_cap);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_pause_is_idempotent() {
        let (caller, pauser_owner, _u) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, pauser_owner);
        let token_bridge_state = take_state(scenario);
        let pauser_cap = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);

        // Pause twice — should not revert.
        pause::pause(&mut token_bridge_state, &pauser_cap, test_scenario::ctx(scenario));
        assert!(state::is_paused(&token_bridge_state), 0);
        pause::pause(&mut token_bridge_state, &pauser_cap, test_scenario::ctx(scenario));
        assert!(state::is_paused(&token_bridge_state), 0);

        test_scenario::return_to_address(pauser_owner, pauser_cap);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_unpause_is_idempotent() {
        let (caller, _p, unpauser_owner) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::none(),
            option::some(unpauser_owner),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, unpauser_owner);
        let token_bridge_state = take_state(scenario);
        let unpauser_cap =
            test_scenario::take_from_address<UnpauserCap>(scenario, unpauser_owner);

        // Unpause twice — should not revert (already unpaused).
        pause::unpause(&mut token_bridge_state, &unpauser_cap, test_scenario::ctx(scenario));
        assert!(!state::is_paused(&token_bridge_state), 0);
        pause::unpause(&mut token_bridge_state, &unpauser_cap, test_scenario::ctx(scenario));
        assert!(!state::is_paused(&token_bridge_state), 0);

        test_scenario::return_to_address(unpauser_owner, unpauser_cap);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_rotate_deprecates_old_cap() {
        let (caller, owner_a, owner_b) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        // Assign owner_a.
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(owner_a),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        // owner_a's cap can pause.
        test_scenario::next_tx(scenario, owner_a);
        let token_bridge_state = take_state(scenario);
        let cap_a = test_scenario::take_from_address<PauserCap>(scenario, owner_a);
        pause::pause(&mut token_bridge_state, &cap_a, test_scenario::ctx(scenario));
        assert!(state::is_paused(&token_bridge_state), 0);
        state::set_paused_test_only(&mut token_bridge_state, false);

        // Rotate: governance mints a new cap for owner_b, deprecating cap_a.
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(owner_b),
            option::none(),
            test_scenario::ctx(scenario)
        );
        test_scenario::return_to_address(owner_a, cap_a);
        return_state(token_bridge_state);

        // owner_b's new cap can pause.
        test_scenario::next_tx(scenario, owner_b);
        let token_bridge_state = take_state(scenario);
        let cap_b = test_scenario::take_from_address<PauserCap>(scenario, owner_b);
        pause::pause(&mut token_bridge_state, &cap_b, test_scenario::ctx(scenario));
        assert!(state::is_paused(&token_bridge_state), 0);

        test_scenario::return_to_address(owner_b, cap_b);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_PAUSER)]
    fun test_deprecated_cap_cannot_pause() {
        let (caller, owner_a, owner_b) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        // Assign owner_a, then rotate to owner_b (deprecating cap_a).
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(owner_a),
            option::none(),
            test_scenario::ctx(scenario)
        );
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(owner_b),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        // owner_a's now-deprecated cap must fail.
        test_scenario::next_tx(scenario, owner_a);
        let token_bridge_state = take_state(scenario);
        let cap_a = test_scenario::take_from_address<PauserCap>(scenario, owner_a);

        // You shall not pass!
        pause::pause(&mut token_bridge_state, &cap_a, test_scenario::ctx(scenario));

        abort 42
    }

    // ========================================================================
    //  Access Control — Negative Tests
    // ========================================================================

    #[test]
    #[expected_failure(abort_code = pause::E_PAUSER_NOT_CONFIGURED)]
    fun test_cannot_pause_when_pauser_unassigned() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        // Pauser unassigned; mint a stray cap to attempt with.
        let stray_cap = pause::new_pauser_cap_test_only(test_scenario::ctx(scenario));

        // You shall not pass!
        pause::pause(&mut token_bridge_state, &stray_cap, test_scenario::ctx(scenario));

        pause::destroy_pauser_cap_test_only(stray_cap);
        abort 42
    }

    #[test]
    #[expected_failure(abort_code = pause::E_UNPAUSER_NOT_CONFIGURED)]
    fun test_cannot_unpause_when_unpauser_unassigned() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);
        state::set_paused_test_only(&mut token_bridge_state, true);

        let stray_cap = pause::new_unpauser_cap_test_only(test_scenario::ctx(scenario));

        // You shall not pass!
        pause::unpause(&mut token_bridge_state, &stray_cap, test_scenario::ctx(scenario));

        pause::destroy_unpauser_cap_test_only(stray_cap);
        abort 42
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_PAUSER)]
    fun test_stray_cap_cannot_pause() {
        let (caller, pauser_owner, _u) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        // Designate a real pauser owner, but attempt with a stray (undesignated) cap.
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::none(),
            test_scenario::ctx(scenario)
        );
        let stray_cap = pause::new_pauser_cap_test_only(test_scenario::ctx(scenario));

        // You shall not pass!
        pause::pause(&mut token_bridge_state, &stray_cap, test_scenario::ctx(scenario));

        pause::destroy_pauser_cap_test_only(stray_cap);
        abort 42
    }

    // ========================================================================
    //  assert_not_paused Guard
    // ========================================================================

    #[test]
    #[expected_failure(abort_code = state::E_PAUSED)]
    fun test_assert_not_paused_reverts_when_paused() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        state::set_paused_test_only(&mut token_bridge_state, true);

        // You shall not pass!
        state::assert_not_paused_test_only(&token_bridge_state);

        abort 42
    }

    #[test]
    fun test_assert_not_paused_passes_when_not_paused() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        // Should not revert.
        state::assert_not_paused_test_only(&token_bridge_state);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  Governance Payload Decode (wire format)
    // ========================================================================

    #[test]
    fun test_parse_payload_both_owners() {
        // [32][A..][32][B..]
        let a = x"00000000000000000000000000000000000000000000000000000000000000aa";
        let b = x"00000000000000000000000000000000000000000000000000000000000000bb";
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 32);
        vector::append(&mut payload, a);
        vector::push_back(&mut payload, 32);
        vector::append(&mut payload, b);

        let (pauser, unpauser) =
            set_pauser_addresses::parse_payload_test_only(payload);
        assert!(option::contains(&pauser, &@0xaa), 0);
        assert!(option::contains(&unpauser, &@0xbb), 0);
    }

    #[test]
    fun test_parse_payload_pauser_unassigned() {
        // [0][32][B..]  -> pauser unassigned, unpauser = B
        let b = x"00000000000000000000000000000000000000000000000000000000000000bb";
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 32);
        vector::append(&mut payload, b);

        let (pauser, unpauser) =
            set_pauser_addresses::parse_payload_test_only(payload);
        assert!(option::is_none(&pauser), 0);
        assert!(option::contains(&unpauser, &@0xbb), 0);
    }

    #[test]
    fun test_parse_payload_both_unassigned() {
        // [0][0] -> both unassigned
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 0);

        let (pauser, unpauser) =
            set_pauser_addresses::parse_payload_test_only(payload);
        assert!(option::is_none(&pauser), 0);
        assert!(option::is_none(&unpauser), 0);
    }

    #[test]
    fun test_parse_payload_all_zero_addr_is_unassigned() {
        // [32][0x00..00][0] -> all-zero 32-byte decodes to none (unassigned)
        let zeros = x"0000000000000000000000000000000000000000000000000000000000000000";
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 32);
        vector::append(&mut payload, zeros);
        vector::push_back(&mut payload, 0);

        let (pauser, unpauser) =
            set_pauser_addresses::parse_payload_test_only(payload);
        assert!(option::is_none(&pauser), 0);
        assert!(option::is_none(&unpauser), 0);
    }

    #[test]
    #[expected_failure(abort_code = set_pauser_addresses::E_INVALID_ADDRESS_LENGTH)]
    fun test_parse_payload_bad_length_aborts() {
        // [31][...] -> invalid length
        let bad = x"00000000000000000000000000000000000000000000000000000000000000";
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 31);
        vector::append(&mut payload, bad);

        // You shall not pass!
        let (_p, _u) = set_pauser_addresses::parse_payload_test_only(payload);
    }

    #[test]
    #[expected_failure]
    fun test_parse_payload_trailing_bytes_aborts() {
        // [0][0][extra] -> cursor not empty after parsing both
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 0xff);

        // You shall not pass! (cursor::destroy_empty aborts)
        let (_p, _u) = set_pauser_addresses::parse_payload_test_only(payload);
    }
}
