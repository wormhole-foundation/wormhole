// SPDX-License-Identifier: Apache 2

#[test_only]
module token_bridge::pause_tests {
    use std::option::{Self};
    use std::vector::{Self};
    use sui::clock::{Self, Clock};
    use sui::test_scenario::{Self, Scenario};

    use token_bridge::pause::{Self, PauserCap, FreezerCap, UnpauserCap};
    use token_bridge::set_pauser_addresses::{Self};
    use token_bridge::state::{Self, State};
    use token_bridge::token_bridge_scenario::{
        person,
        return_state,
        set_up_wormhole_and_token_bridge,
        take_state,
        three_people
    };

    const WORMHOLE_FEE: u64 = 350;

    // ------------------------------------------------------------------------
    //  Helpers
    // ------------------------------------------------------------------------

    /// A fresh Clock for testing (starts at timestamp 0).
    fun new_clock(scenario: &mut Scenario): Clock {
        clock::create_for_testing(test_scenario::ctx(scenario))
    }

    /// Set up wormhole + token bridge, init pause state, return State at `caller`.
    fun set_up(scenario: &mut Scenario, caller: address): State {
        set_up_wormhole_and_token_bridge(scenario, WORMHOLE_FEE);
        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        state::init_pause_state_test_only(&mut token_bridge_state);
        token_bridge_state
    }

    // ========================================================================
    //  Pause State Initialization
    // ========================================================================

    #[test]
    fun test_default_pause_state() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);

        // Default: not paused, expiry 0, all roles unassigned.
        assert!(!state::is_paused(&token_bridge_state), 0);
        assert!(state::pause_expiry(&token_bridge_state) == 0, 0);
        assert!(option::is_none(&state::pauser(&token_bridge_state)), 0);
        assert!(option::is_none(&state::freezer(&token_bridge_state)), 0);
        assert!(option::is_none(&state::unpauser(&token_bridge_state)), 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_pre_init_defaults() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        set_up_wormhole_and_token_bridge(scenario, WORMHOLE_FEE);
        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);

        // Before init, getters return safe defaults (backwards compat).
        assert!(!state::is_paused(&token_bridge_state), 0);
        assert!(state::pause_expiry(&token_bridge_state) == 0, 0);
        assert!(option::is_none(&state::pauser(&token_bridge_state)), 0);
        assert!(option::is_none(&state::freezer(&token_bridge_state)), 0);
        assert!(option::is_none(&state::unpauser(&token_bridge_state)), 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  Set Pauser Addresses (mint + transfer to owner, record cap id)
    // ========================================================================

    #[test]
    fun test_set_pauser_addresses_mints_three_roles() {
        let (caller, pauser_owner, unpauser_owner) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);

        // Use `caller` as the freezer owner (distinct from the other two).
        let freezer_owner = caller;

        let (pauser_id, freezer_id, unpauser_id) =
            set_pauser_addresses::set_pauser_addresses_test_only(
                &mut token_bridge_state,
                option::some(pauser_owner),
                option::some(freezer_owner),
                option::some(unpauser_owner),
                test_scenario::ctx(scenario)
            );

        assert!(option::is_some(&pauser_id), 0);
        assert!(option::is_some(&freezer_id), 0);
        assert!(option::is_some(&unpauser_id), 0);
        assert!(state::pauser(&token_bridge_state) == pauser_id, 0);
        assert!(state::freezer(&token_bridge_state) == freezer_id, 0);
        assert!(state::unpauser(&token_bridge_state) == unpauser_id, 0);

        return_state(token_bridge_state);

        // Caps landed with their owners.
        test_scenario::next_tx(scenario, pauser_owner);
        let pc = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);
        assert!(option::contains(&pauser_id, &pause::pauser_cap_id(&pc)), 0);
        test_scenario::return_to_address(pauser_owner, pc);

        test_scenario::next_tx(scenario, freezer_owner);
        let fc = test_scenario::take_from_address<FreezerCap>(scenario, freezer_owner);
        assert!(option::contains(&freezer_id, &pause::freezer_cap_id(&fc)), 0);
        test_scenario::return_to_address(freezer_owner, fc);

        test_scenario::next_tx(scenario, unpauser_owner);
        let uc = test_scenario::take_from_address<UnpauserCap>(scenario, unpauser_owner);
        assert!(option::contains(&unpauser_id, &pause::unpauser_cap_id(&uc)), 0);
        test_scenario::return_to_address(unpauser_owner, uc);

        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_set_to_none_unassigns_all() {
        let (caller, pauser_owner, unpauser_owner) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);

        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::some(caller),
            option::some(unpauser_owner),
            test_scenario::ctx(scenario)
        );

        let (p, f, u) = set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::none(),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );

        assert!(option::is_none(&p), 0);
        assert!(option::is_none(&f), 0);
        assert!(option::is_none(&u), 0);
        assert!(option::is_none(&state::pauser(&token_bridge_state)), 0);
        assert!(option::is_none(&state::freezer(&token_bridge_state)), 0);
        assert!(option::is_none(&state::unpauser(&token_bridge_state)), 0);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  pause() — temporary, timed
    // ========================================================================

    #[test]
    fun test_pause_sets_paused_and_expiry() {
        let (caller, pauser_owner, _u) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, pauser_owner);
        let token_bridge_state = take_state(scenario);
        let cap = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);
        let the_clock = new_clock(scenario);
        clock::set_for_testing(&mut the_clock, 1_000);

        pause::pause(&mut token_bridge_state, &cap, &the_clock, test_scenario::ctx(scenario));

        assert!(state::is_paused(&token_bridge_state), 0);
        assert!(
            state::pause_expiry(&token_bridge_state) == 1_000 + pause::pause_duration_ms(),
            0
        );

        clock::destroy_for_testing(the_clock);
        test_scenario::return_to_address(pauser_owner, cap);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_pause_pushes_expiry_forward() {
        let (caller, pauser_owner, _u) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, pauser_owner);
        let token_bridge_state = take_state(scenario);
        let cap = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);
        let the_clock = new_clock(scenario);

        clock::set_for_testing(&mut the_clock, 1_000);
        pause::pause(&mut token_bridge_state, &cap, &the_clock, test_scenario::ctx(scenario));
        let first_expiry = state::pause_expiry(&token_bridge_state);

        // Advance time and re-pause: expiry should move forward.
        clock::set_for_testing(&mut the_clock, 100_000);
        pause::pause(&mut token_bridge_state, &cap, &the_clock, test_scenario::ctx(scenario));
        let second_expiry = state::pause_expiry(&token_bridge_state);

        assert!(second_expiry > first_expiry, 0);
        assert!(second_expiry == 100_000 + pause::pause_duration_ms(), 0);

        clock::destroy_for_testing(the_clock);
        test_scenario::return_to_address(pauser_owner, cap);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = pause::E_PAUSE_NOT_EXTENDED)]
    fun test_pause_reverts_when_frozen() {
        // freeze sets max expiry; a subsequent pause cannot push it forward, so it must abort
        // with E_PAUSE_NOT_EXTENDED (a lower-trust pauser can never curtail a freeze).
        let (caller, pauser_owner, _u) = three_people();
        let freezer_owner = caller;
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::some(freezer_owner),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        // Freeze first.
        test_scenario::next_tx(scenario, freezer_owner);
        let token_bridge_state = take_state(scenario);
        let fc = test_scenario::take_from_address<FreezerCap>(scenario, freezer_owner);
        pause::freeze_bridge(&mut token_bridge_state, &fc, test_scenario::ctx(scenario));
        assert!(state::pause_expiry(&token_bridge_state) == pause::max_timestamp_ms(), 0);
        test_scenario::return_to_address(freezer_owner, fc);
        return_state(token_bridge_state);

        // Pause on a frozen bridge: cannot extend past max, so this aborts.
        test_scenario::next_tx(scenario, pauser_owner);
        let token_bridge_state = take_state(scenario);
        let pc = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);
        let the_clock = new_clock(scenario);
        clock::set_for_testing(&mut the_clock, 1_000);
        pause::pause(&mut token_bridge_state, &pc, &the_clock, test_scenario::ctx(scenario));

        // Unreachable — the pause above aborts. Cleanup keeps the borrow checker happy.
        clock::destroy_for_testing(the_clock);
        test_scenario::return_to_address(pauser_owner, pc);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  freeze_bridge()
    // ========================================================================

    #[test]
    fun test_freeze_sets_max_expiry() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::none(),
            option::some(caller),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        let fc = test_scenario::take_from_address<FreezerCap>(scenario, caller);

        pause::freeze_bridge(&mut token_bridge_state, &fc, test_scenario::ctx(scenario));
        assert!(state::is_paused(&token_bridge_state), 0);
        assert!(state::pause_expiry(&token_bridge_state) == pause::max_timestamp_ms(), 0);

        // Idempotent: freezing again is a no-op effect.
        pause::freeze_bridge(&mut token_bridge_state, &fc, test_scenario::ctx(scenario));
        assert!(state::pause_expiry(&token_bridge_state) == pause::max_timestamp_ms(), 0);

        test_scenario::return_to_address(caller, fc);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  unpause()
    // ========================================================================

    #[test]
    fun test_unpause_clears_and_sets_expiry_now() {
        let (caller, _p, unpauser_owner) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::none(),
            option::none(),
            option::some(unpauser_owner),
            test_scenario::ctx(scenario)
        );
        // Start paused.
        state::set_paused_test_only(&mut token_bridge_state, true);
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, unpauser_owner);
        let token_bridge_state = take_state(scenario);
        let uc = test_scenario::take_from_address<UnpauserCap>(scenario, unpauser_owner);
        let the_clock = new_clock(scenario);
        clock::set_for_testing(&mut the_clock, 5_000);

        pause::unpause(&mut token_bridge_state, &uc, &the_clock, test_scenario::ctx(scenario));

        assert!(!state::is_paused(&token_bridge_state), 0);
        // Expiry set to now (so a later pause is not blocked by a stale value).
        assert!(state::pause_expiry(&token_bridge_state) == 5_000, 0);

        clock::destroy_for_testing(the_clock);
        test_scenario::return_to_address(unpauser_owner, uc);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_unpause_after_freeze_then_pause_works() {
        // freeze -> unpause (expiry=now) -> pause must succeed and set a normal
        // 5-day expiry (the stale max expiry must not linger).
        let (caller, pauser_owner, unpauser_owner) = three_people();
        let freezer_owner = caller;
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::some(freezer_owner),
            option::some(unpauser_owner),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, caller);
        let the_clock = new_clock(scenario);
        clock::set_for_testing(&mut the_clock, 10_000);

        // Freeze.
        test_scenario::next_tx(scenario, freezer_owner);
        let token_bridge_state = take_state(scenario);
        let fc = test_scenario::take_from_address<FreezerCap>(scenario, freezer_owner);
        pause::freeze_bridge(&mut token_bridge_state, &fc, test_scenario::ctx(scenario));
        test_scenario::return_to_address(freezer_owner, fc);
        return_state(token_bridge_state);

        // Unpause (expiry -> now = 10_000).
        test_scenario::next_tx(scenario, unpauser_owner);
        let token_bridge_state = take_state(scenario);
        let uc = test_scenario::take_from_address<UnpauserCap>(scenario, unpauser_owner);
        pause::unpause(&mut token_bridge_state, &uc, &the_clock, test_scenario::ctx(scenario));
        assert!(state::pause_expiry(&token_bridge_state) == 10_000, 0);
        test_scenario::return_to_address(unpauser_owner, uc);
        return_state(token_bridge_state);

        // Pause now sets a normal 5-day expiry (10_000 + duration > 10_000).
        test_scenario::next_tx(scenario, pauser_owner);
        let token_bridge_state = take_state(scenario);
        let pc = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);
        pause::pause(&mut token_bridge_state, &pc, &the_clock, test_scenario::ctx(scenario));
        assert!(state::is_paused(&token_bridge_state), 0);
        assert!(
            state::pause_expiry(&token_bridge_state) == 10_000 + pause::pause_duration_ms(),
            0
        );
        test_scenario::return_to_address(pauser_owner, pc);
        return_state(token_bridge_state);

        clock::destroy_for_testing(the_clock);
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_PAUSED)]
    fun test_unpause_reverts_when_not_paused() {
        let (caller, _p, unpauser_owner) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::none(),
            option::none(),
            option::some(unpauser_owner),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, unpauser_owner);
        let token_bridge_state = take_state(scenario);
        let uc = test_scenario::take_from_address<UnpauserCap>(scenario, unpauser_owner);
        let the_clock = new_clock(scenario);

        // Not paused — must revert.
        pause::unpause(&mut token_bridge_state, &uc, &the_clock, test_scenario::ctx(scenario));

        clock::destroy_for_testing(the_clock);
        test_scenario::return_to_address(unpauser_owner, uc);
        return_state(token_bridge_state);
        abort 42
    }

    // ========================================================================
    //  unpause_expired() — permissionless
    // ========================================================================

    #[test]
    fun test_unpause_expired_after_expiry() {
        let (caller, pauser_owner, _u) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        // Pause at t=1000 (expiry = 1000 + 5d).
        test_scenario::next_tx(scenario, pauser_owner);
        let token_bridge_state = take_state(scenario);
        let pc = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);
        let the_clock = new_clock(scenario);
        clock::set_for_testing(&mut the_clock, 1_000);
        pause::pause(&mut token_bridge_state, &pc, &the_clock, test_scenario::ctx(scenario));
        test_scenario::return_to_address(pauser_owner, pc);
        return_state(token_bridge_state);

        // Advance past expiry; anyone (caller != pauser) can unpause.
        clock::set_for_testing(&mut the_clock, 1_000 + pause::pause_duration_ms() + 1);
        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        pause::unpause_expired(&mut token_bridge_state, &the_clock, test_scenario::ctx(scenario));
        assert!(!state::is_paused(&token_bridge_state), 0);

        clock::destroy_for_testing(the_clock);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_unpause_expired_at_exact_expiry() {
        // Boundary: now == pauseExpiry must succeed (the guard is `>=`).
        let (caller, pauser_owner, _u) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, pauser_owner);
        let token_bridge_state = take_state(scenario);
        let pc = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);
        let the_clock = new_clock(scenario);
        clock::set_for_testing(&mut the_clock, 1_000);
        pause::pause(&mut token_bridge_state, &pc, &the_clock, test_scenario::ctx(scenario));
        let expiry = state::pause_expiry(&token_bridge_state);
        test_scenario::return_to_address(pauser_owner, pc);
        return_state(token_bridge_state);

        // Set time to exactly the expiry; unpause_expired must succeed.
        clock::set_for_testing(&mut the_clock, expiry);
        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        pause::unpause_expired(&mut token_bridge_state, &the_clock, test_scenario::ctx(scenario));
        assert!(!state::is_paused(&token_bridge_state), 0);

        clock::destroy_for_testing(the_clock);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_EXPIRED)]
    fun test_unpause_expired_reverts_before_expiry() {
        let (caller, pauser_owner, _u) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, pauser_owner);
        let token_bridge_state = take_state(scenario);
        let pc = test_scenario::take_from_address<PauserCap>(scenario, pauser_owner);
        let the_clock = new_clock(scenario);
        clock::set_for_testing(&mut the_clock, 1_000);
        pause::pause(&mut token_bridge_state, &pc, &the_clock, test_scenario::ctx(scenario));
        test_scenario::return_to_address(pauser_owner, pc);
        return_state(token_bridge_state);

        // Still within the window — must revert.
        clock::set_for_testing(&mut the_clock, 2_000);
        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);

        pause::unpause_expired(&mut token_bridge_state, &the_clock, test_scenario::ctx(scenario));

        clock::destroy_for_testing(the_clock);
        return_state(token_bridge_state);
        abort 42
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_PAUSED)]
    fun test_unpause_expired_reverts_when_not_paused() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        let the_clock = new_clock(scenario);

        // Not paused — must revert (even though now >= expiry==0).
        pause::unpause_expired(&mut token_bridge_state, &the_clock, test_scenario::ctx(scenario));

        clock::destroy_for_testing(the_clock);
        return_state(token_bridge_state);
        abort 42
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_EXPIRED)]
    fun test_unpause_expired_cannot_lift_freeze() {
        // freeze sets expiry to max, so unpause_expired can never lift it.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::none(),
            option::some(caller),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, caller);
        let token_bridge_state = take_state(scenario);
        let fc = test_scenario::take_from_address<FreezerCap>(scenario, caller);
        pause::freeze_bridge(&mut token_bridge_state, &fc, test_scenario::ctx(scenario));
        test_scenario::return_to_address(caller, fc);

        let the_clock = new_clock(scenario);
        clock::set_for_testing(&mut the_clock, pause::max_timestamp_ms() - 1);

        // now < max expiry — must revert.
        pause::unpause_expired(&mut token_bridge_state, &the_clock, test_scenario::ctx(scenario));

        clock::destroy_for_testing(the_clock);
        return_state(token_bridge_state);
        abort 42
    }

    // ========================================================================
    //  Access Control — wrong / unconfigured caps
    // ========================================================================

    #[test]
    #[expected_failure(abort_code = pause::E_PAUSER_NOT_CONFIGURED)]
    fun test_pause_reverts_when_pauser_unassigned() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        let stray = pause::new_pauser_cap_test_only(test_scenario::ctx(scenario));
        let the_clock = new_clock(scenario);

        pause::pause(&mut token_bridge_state, &stray, &the_clock, test_scenario::ctx(scenario));

        clock::destroy_for_testing(the_clock);
        pause::destroy_pauser_cap_test_only(stray);
        return_state(token_bridge_state);
        abort 42
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_PAUSER)]
    fun test_stray_cap_cannot_pause() {
        let (caller, pauser_owner, _u) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(pauser_owner),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );
        let stray = pause::new_pauser_cap_test_only(test_scenario::ctx(scenario));
        let the_clock = new_clock(scenario);

        pause::pause(&mut token_bridge_state, &stray, &the_clock, test_scenario::ctx(scenario));

        clock::destroy_for_testing(the_clock);
        pause::destroy_pauser_cap_test_only(stray);
        return_state(token_bridge_state);
        abort 42
    }

    #[test]
    #[expected_failure(abort_code = pause::E_FREEZER_NOT_CONFIGURED)]
    fun test_freeze_reverts_when_freezer_unassigned() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        let stray = pause::new_freezer_cap_test_only(test_scenario::ctx(scenario));

        pause::freeze_bridge(&mut token_bridge_state, &stray, test_scenario::ctx(scenario));

        pause::destroy_freezer_cap_test_only(stray);
        return_state(token_bridge_state);
        abort 42
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_FREEZER)]
    fun test_stray_cap_cannot_freeze() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::none(),
            option::some(caller),
            option::none(),
            test_scenario::ctx(scenario)
        );
        let stray = pause::new_freezer_cap_test_only(test_scenario::ctx(scenario));

        pause::freeze_bridge(&mut token_bridge_state, &stray, test_scenario::ctx(scenario));

        pause::destroy_freezer_cap_test_only(stray);
        return_state(token_bridge_state);
        abort 42
    }

    #[test]
    #[expected_failure(abort_code = pause::E_UNPAUSER_NOT_CONFIGURED)]
    fun test_unpause_reverts_when_unpauser_unassigned() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        state::set_paused_test_only(&mut token_bridge_state, true);
        let stray = pause::new_unpauser_cap_test_only(test_scenario::ctx(scenario));
        let the_clock = new_clock(scenario);

        pause::unpause(&mut token_bridge_state, &stray, &the_clock, test_scenario::ctx(scenario));

        clock::destroy_for_testing(the_clock);
        pause::destroy_unpauser_cap_test_only(stray);
        return_state(token_bridge_state);
        abort 42
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_UNPAUSER)]
    fun test_stray_cap_cannot_unpause() {
        let (caller, _p, unpauser_owner) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::none(),
            option::none(),
            option::some(unpauser_owner),
            test_scenario::ctx(scenario)
        );
        state::set_paused_test_only(&mut token_bridge_state, true);
        let stray = pause::new_unpauser_cap_test_only(test_scenario::ctx(scenario));
        let the_clock = new_clock(scenario);

        pause::unpause(&mut token_bridge_state, &stray, &the_clock, test_scenario::ctx(scenario));

        clock::destroy_for_testing(the_clock);
        pause::destroy_unpauser_cap_test_only(stray);
        return_state(token_bridge_state);
        abort 42
    }

    // ========================================================================
    //  Rotation
    // ========================================================================

    #[test]
    fun test_rotate_deprecates_old_pauser_cap() {
        let (caller, owner_a, owner_b) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(owner_a),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        // owner_a pauses.
        test_scenario::next_tx(scenario, owner_a);
        let token_bridge_state = take_state(scenario);
        let cap_a = test_scenario::take_from_address<PauserCap>(scenario, owner_a);
        let the_clock = new_clock(scenario);
        clock::set_for_testing(&mut the_clock, 1_000);
        pause::pause(&mut token_bridge_state, &cap_a, &the_clock, test_scenario::ctx(scenario));
        assert!(state::is_paused(&token_bridge_state), 0);
        state::set_paused_test_only(&mut token_bridge_state, false);

        // Rotate to owner_b, deprecating cap_a.
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(owner_b),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );
        test_scenario::return_to_address(owner_a, cap_a);
        return_state(token_bridge_state);

        // owner_b's new cap works. Advance the clock so the pause pushes the expiry strictly
        // forward (the prior pause left `pause_expiry` at 1_000 + 5d).
        test_scenario::next_tx(scenario, owner_b);
        let token_bridge_state = take_state(scenario);
        let cap_b = test_scenario::take_from_address<PauserCap>(scenario, owner_b);
        clock::set_for_testing(&mut the_clock, 2_000);
        pause::pause(&mut token_bridge_state, &cap_b, &the_clock, test_scenario::ctx(scenario));
        assert!(state::is_paused(&token_bridge_state), 0);

        clock::destroy_for_testing(the_clock);
        test_scenario::return_to_address(owner_b, cap_b);
        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = pause::E_NOT_PAUSER)]
    fun test_deprecated_pauser_cap_fails() {
        let (caller, owner_a, owner_b) = three_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(owner_a),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );
        set_pauser_addresses::set_pauser_addresses_test_only(
            &mut token_bridge_state,
            option::some(owner_b),
            option::none(),
            option::none(),
            test_scenario::ctx(scenario)
        );
        return_state(token_bridge_state);

        test_scenario::next_tx(scenario, owner_a);
        let token_bridge_state = take_state(scenario);
        let cap_a = test_scenario::take_from_address<PauserCap>(scenario, owner_a);
        let the_clock = new_clock(scenario);

        pause::pause(&mut token_bridge_state, &cap_a, &the_clock, test_scenario::ctx(scenario));

        clock::destroy_for_testing(the_clock);
        test_scenario::return_to_address(owner_a, cap_a);
        return_state(token_bridge_state);
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
        let token_bridge_state = set_up(scenario, caller);

        state::set_paused_test_only(&mut token_bridge_state, true);
        state::assert_not_paused_test_only(&token_bridge_state);

        return_state(token_bridge_state);
        abort 42
    }

    #[test]
    fun test_assert_not_paused_passes_when_not_paused() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;
        let token_bridge_state = set_up(scenario, caller);

        state::assert_not_paused_test_only(&token_bridge_state);

        return_state(token_bridge_state);
        test_scenario::end(my_scenario);
    }

    // ========================================================================
    //  Governance Payload Decode (wire format, 3 owners)
    // ========================================================================

    #[test]
    fun test_parse_payload_three_owners() {
        // [32][A][32][B][32][C]
        let a = x"00000000000000000000000000000000000000000000000000000000000000aa";
        let b = x"00000000000000000000000000000000000000000000000000000000000000bb";
        let c = x"00000000000000000000000000000000000000000000000000000000000000cc";
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 32);
        vector::append(&mut payload, a);
        vector::push_back(&mut payload, 32);
        vector::append(&mut payload, b);
        vector::push_back(&mut payload, 32);
        vector::append(&mut payload, c);

        let (p, f, u) = set_pauser_addresses::parse_payload_test_only(payload);
        assert!(option::contains(&p, &@0xaa), 0);
        assert!(option::contains(&f, &@0xbb), 0);
        assert!(option::contains(&u, &@0xcc), 0);
    }

    #[test]
    fun test_parse_payload_middle_unassigned() {
        // [32][A][0][32][C] -> freezer unassigned
        let a = x"00000000000000000000000000000000000000000000000000000000000000aa";
        let c = x"00000000000000000000000000000000000000000000000000000000000000cc";
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 32);
        vector::append(&mut payload, a);
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 32);
        vector::append(&mut payload, c);

        let (p, f, u) = set_pauser_addresses::parse_payload_test_only(payload);
        assert!(option::contains(&p, &@0xaa), 0);
        assert!(option::is_none(&f), 0);
        assert!(option::contains(&u, &@0xcc), 0);
    }

    #[test]
    fun test_parse_payload_all_unassigned() {
        // [0][0][0]
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 0);

        let (p, f, u) = set_pauser_addresses::parse_payload_test_only(payload);
        assert!(option::is_none(&p), 0);
        assert!(option::is_none(&f), 0);
        assert!(option::is_none(&u), 0);
    }

    #[test]
    fun test_parse_payload_all_zero_addr_is_unassigned() {
        // [32][0x00..00][0][0] -> all-zero 32-byte decodes to none
        let zeros = x"0000000000000000000000000000000000000000000000000000000000000000";
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 32);
        vector::append(&mut payload, zeros);
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 0);

        let (p, f, u) = set_pauser_addresses::parse_payload_test_only(payload);
        assert!(option::is_none(&p), 0);
        assert!(option::is_none(&f), 0);
        assert!(option::is_none(&u), 0);
    }

    #[test]
    #[expected_failure(abort_code = set_pauser_addresses::E_INVALID_ADDRESS_LENGTH)]
    fun test_parse_payload_bad_length_aborts() {
        // [31][...] -> invalid length
        let bad = x"00000000000000000000000000000000000000000000000000000000000000";
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 31);
        vector::append(&mut payload, bad);

        let (_p, _f, _u) = set_pauser_addresses::parse_payload_test_only(payload);
    }

    #[test]
    #[expected_failure]
    fun test_parse_payload_trailing_bytes_aborts() {
        // [0][0][0][extra] -> cursor not empty after three owners
        let payload = vector::empty<u8>();
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 0);
        vector::push_back(&mut payload, 0xff);

        let (_p, _f, _u) = set_pauser_addresses::parse_payload_test_only(payload);
    }
}
