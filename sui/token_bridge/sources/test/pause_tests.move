// SPDX-License-Identifier: Apache 2

#[test_only]
module token_bridge::pause_tests {
    use sui::test_scenario::{Self};

    use token_bridge::pause::{Self};
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

        // Default state: not paused, pauser/unpauser = @0x0.
        assert!(!state::is_paused(&token_bridge_state), 0);
        assert!(state::pauser(&token_bridge_state) == @0x0, 0);
        assert!(state::unpauser(&token_bridge_state) == @0x0, 0);

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
        assert!(state::pauser(&token_bridge_state) == @0x0, 0);
        assert!(state::unpauser(&token_bridge_state) == @0x0, 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  Set Pauser Addresses (via test helper)
    // ========================================================================

    #[test]
    fun test_set_pauser_addresses() {
        let (caller, pauser_addr, unpauser_addr) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        // Set pauser and unpauser.
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            pauser_addr,
            unpauser_addr
        );

        assert!(state::pauser(&token_bridge_state) == pauser_addr, 0);
        assert!(state::unpauser(&token_bridge_state) == unpauser_addr, 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_rotate_pauser_addresses() {
        let (caller, pauser_addr, unpauser_addr) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        // Set initial addresses.
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            pauser_addr,
            unpauser_addr
        );

        // Rotate to new addresses (swap them).
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            unpauser_addr,
            pauser_addr
        );

        assert!(state::pauser(&token_bridge_state) == unpauser_addr, 0);
        assert!(state::unpauser(&token_bridge_state) == pauser_addr, 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_set_pauser_to_zero_unassigns() {
        let (caller, pauser_addr, unpauser_addr) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        // Set addresses.
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            pauser_addr,
            unpauser_addr
        );

        // Unassign by setting to @0x0.
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            @0x0,
            @0x0
        );

        assert!(state::pauser(&token_bridge_state) == @0x0, 0);
        assert!(state::unpauser(&token_bridge_state) == @0x0, 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  Pause / Unpause
    // ========================================================================

    #[test]
    fun test_pauser_can_pause() {
        let (caller, pauser_addr, unpauser_addr) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            pauser_addr,
            unpauser_addr
        );

        return_state(token_bridge_state);

        // Pause as the configured pauser.
        test_scenario::next_tx(scenario, pauser_addr);
        let token_bridge_state = take_state(scenario);

        pause::pause(
            &mut token_bridge_state,
            test_scenario::ctx(scenario)
        );

        assert!(state::is_paused(&token_bridge_state), 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_unpauser_can_unpause() {
        let (caller, pauser_addr, unpauser_addr) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            pauser_addr,
            unpauser_addr
        );

        // Set paused directly for this test.
        state::set_paused_test_only(&mut token_bridge_state, true);
        assert!(state::is_paused(&token_bridge_state), 0);

        return_state(token_bridge_state);

        // Unpause as the configured unpauser.
        test_scenario::next_tx(scenario, unpauser_addr);
        let token_bridge_state = take_state(scenario);

        pause::unpause(
            &mut token_bridge_state,
            test_scenario::ctx(scenario)
        );

        assert!(!state::is_paused(&token_bridge_state), 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_pause_is_idempotent() {
        let (caller, pauser_addr, unpauser_addr) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            pauser_addr,
            unpauser_addr
        );

        return_state(token_bridge_state);

        // Pause twice — should not revert.
        test_scenario::next_tx(scenario, pauser_addr);
        let token_bridge_state = take_state(scenario);

        pause::pause(
            &mut token_bridge_state,
            test_scenario::ctx(scenario)
        );
        assert!(state::is_paused(&token_bridge_state), 0);

        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, pauser_addr);
        let token_bridge_state = take_state(scenario);

        pause::pause(
            &mut token_bridge_state,
            test_scenario::ctx(scenario)
        );
        assert!(state::is_paused(&token_bridge_state), 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_unpause_is_idempotent() {
        let (caller, _pauser_addr, unpauser_addr) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            caller,
            unpauser_addr
        );

        return_state(token_bridge_state);

        // Unpause twice — should not revert (already unpaused).
        test_scenario::next_tx(scenario, unpauser_addr);
        let token_bridge_state = take_state(scenario);

        pause::unpause(
            &mut token_bridge_state,
            test_scenario::ctx(scenario)
        );
        assert!(!state::is_paused(&token_bridge_state), 0);

        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, unpauser_addr);
        let token_bridge_state = take_state(scenario);

        pause::unpause(
            &mut token_bridge_state,
            test_scenario::ctx(scenario)
        );
        assert!(!state::is_paused(&token_bridge_state), 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  Access Control — Negative Tests
    // ========================================================================

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_PAUSER)]
    fun test_non_pauser_cannot_pause() {
        let (caller, pauser_addr, unpauser_addr) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            pauser_addr,
            unpauser_addr
        );

        return_state(token_bridge_state);

        // Try to pause as a random (non-pauser) address.
        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);

        // You shall not pass!
        pause::pause(
            &mut token_bridge_state,
            test_scenario::ctx(scenario)
        );

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_UNPAUSER)]
    fun test_non_unpauser_cannot_unpause() {
        let (caller, pauser_addr, unpauser_addr) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);

        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            pauser_addr,
            unpauser_addr
        );

        state::set_paused_test_only(&mut token_bridge_state, true);

        return_state(token_bridge_state);

        // Try to unpause as a random (non-unpauser) address.
        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);

        // You shall not pass!
        pause::unpause(
            &mut token_bridge_state,
            test_scenario::ctx(scenario)
        );

        abort 42
    }

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

        // Pauser is @0x0 (default / unassigned).

        // You shall not pass!
        pause::pause(
            &mut token_bridge_state,
            test_scenario::ctx(scenario)
        );

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

        // Unpauser is @0x0 (default / unassigned).

        // You shall not pass!
        pause::unpause(
            &mut token_bridge_state,
            test_scenario::ctx(scenario)
        );

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
}
