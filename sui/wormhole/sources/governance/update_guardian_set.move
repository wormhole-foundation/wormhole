// SPDX-License-Identifier: Apache 2

/// This module implements handling a governance VAA to enact updating the
/// current guardian set to be a new set of guardian public keys. As a part of
/// this process, the previous guardian set's expiration time is set. Keep in
/// mind that the current guardian set has no expiration.
module wormhole::update_guardian_set {
    use std::vector::{Self};
    use sui::clock::{Clock};

    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::governance_message::{Self, DecreeTicket, DecreeReceipt};
    use wormhole::guardian::{Self, Guardian};
    use wormhole::guardian_set::{Self};
    use wormhole::state::{Self, State, LatestOnly};

    /// No guardians public keys found in VAA.
    const E_NO_GUARDIANS: u64 = 0;
    /// Guardian set index is not incremented from last known guardian set.
    const E_NON_INCREMENTAL_GUARDIAN_SETS: u64 = 1;

    /// Specific governance payload ID (action) for updating the guardian set.
    const ACTION_UPDATE_GUARDIAN_SET: u8 = 2;

    struct GovernanceWitness has drop {}

    /// Event reflecting a Guardian Set update.
    struct GuardianSetAdded has drop, copy {
        new_index: u32
    }

    struct UpdateGuardianSet {
        new_index: u32,
        guardians: vector<Guardian>,
    }

    public fun authorize_governance(
        wormhole_state: &State
    ): DecreeTicket<GovernanceWitness> {
        governance_message::authorize_verify_global(
            GovernanceWitness {},
            state::governance_chain(wormhole_state),
            state::governance_contract(wormhole_state),
            state::governance_module(),
            ACTION_UPDATE_GUARDIAN_SET
        )
    }

    /// Redeem governance VAA to update the current Guardian set with a new
    /// set of Guardian public keys. This governance action is applied globally
    /// across all networks.
    ///
    /// NOTE: This method is guarded by a minimum build version check. This
    /// method could break backward compatibility on an upgrade.
    public fun update_guardian_set(
        wormhole_state: &mut State,
        receipt: DecreeReceipt<GovernanceWitness>,
        the_clock: &Clock
    ): u32 {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(wormhole_state);

        // Even though this disallows the VAA to be replayed, it may be
        // impossible to redeem the same VAA again because `governance_message`
        // requires new governance VAAs being signed by the most recent guardian
        // set).
        let payload =
            governance_message::take_payload(
                state::borrow_mut_consumed_vaas(&latest_only, wormhole_state),
                receipt
            );

        // Proceed with the update.
        handle_update_guardian_set(&latest_only, wormhole_state, payload, the_clock)
    }

    fun handle_update_guardian_set(
        latest_only: &LatestOnly,
        wormhole_state: &mut State,
        governance_payload: vector<u8>,
        the_clock: &Clock
    ): u32 {
        // Deserialize the payload as the updated guardian set.
        let UpdateGuardianSet {
            new_index,
            guardians
        } = deserialize(governance_payload);

        // Every new guardian set index must be incremental from the last known
        // guardian set.
        assert!(
            new_index == state::guardian_set_index(wormhole_state) + 1,
            E_NON_INCREMENTAL_GUARDIAN_SETS
        );

        // Expire the existing guardian set.
        state::expire_guardian_set(latest_only, wormhole_state, the_clock);

        // And store the new one.
        state::add_new_guardian_set(
            latest_only,
            wormhole_state,
            guardian_set::new(new_index, guardians)
        );

        sui::event::emit(GuardianSetAdded { new_index });

        new_index
    }

    fun deserialize(payload: vector<u8>): UpdateGuardianSet {
        let cur = cursor::new(payload);
        let new_index = bytes::take_u32_be(&mut cur);
        let num_guardians = bytes::take_u8(&mut cur);
        assert!(num_guardians > 0, E_NO_GUARDIANS);

        let guardians = vector::empty<Guardian>();
        let i = 0;
        while (i < num_guardians) {
            let key = bytes::take_bytes(&mut cur, 20);
            vector::push_back(&mut guardians, guardian::new(key));
            i = i + 1;
        };
        cursor::destroy_empty(cur);

        UpdateGuardianSet { new_index, guardians }
    }

    #[test_only]
    public fun action(): u8 {
        ACTION_UPDATE_GUARDIAN_SET
    }
}

#[test_only]
module wormhole::update_guardian_set_tests {
    use std::vector::{Self};
    use sui::clock::{Self};
    use sui::test_scenario::{Self};

    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::governance_message::{Self};
    use wormhole::guardian::{Self};
    use wormhole::guardian_set::{Self};
    use wormhole::state::{Self};
    use wormhole::update_guardian_set::{Self};
    use wormhole::vaa::{Self};
    use wormhole::version_control::{Self};
    use wormhole::wormhole_scenario::{
        person,
        return_clock,
        return_state,
        set_up_wormhole,
        take_clock,
        take_state,
        upgrade_wormhole
    };

    const VAA_UPDATE_GUARDIAN_SET_1: vector<u8> =
        x"010000000001004f74e9596bd8246ef456918594ae16e81365b52c0cf4490b2a029fb101b058311f4a5592baeac014dc58215faad36453467a85a4c3e1c6cf5166e80f6e4dc50b0100bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f72650200000000000113befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe88d7d8b32a9105d228100e72dffe2fae0705d31c58076f561cc62a47087b567c86f986426dfcd000bd6e9833490f8fa87c733a183cd076a6cbd29074b853fcf0a5c78c1b56d15fce7a154e6ebe9ed7a2af3503dbd2e37518ab04d7ce78b630f98b15b78a785632dea5609064803b1c8ea8bb2c77a6004bd109a281a698c0f5ba31f158585b41f4f33659e54d3178443ab76a60e21690dbfb17f7f59f09ae3ea1647ec26ae49b14060660504f4da1c2059e1c5ab6810ac3d8e1258bd2f004a94ca0cd4c68fc1c061180610e96d645b12f47ae5cf4546b18538739e90f2edb0d8530e31a218e72b9480202acbaeb06178da78858e5e5c4705cdd4b668ffe3be5bae4867c9d5efe3a05efc62d60e1d19faeb56a80223cdd3472d791b7d32c05abb1cc00b6381fa0c4928f0c56fc14bc029b8809069093d712a3fd4dfab31963597e246ab29fc6ebedf2d392a51ab2dc5c59d0902a03132a84dfd920b35a3d0ba5f7a0635df298f9033e";
    const VAA_UPDATE_GUARDIAN_SET_2A: vector<u8> =
        x"010000000001005fb17d5e0e736e3014756bf7e7335722c4fe3ad18b5b1b566e8e61e562cc44555f30b298bc6a21ea4b192a6f1877a5e638ecf90a77b0b028f297a3a70d93614d0100bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f72650200000000000101befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe";
    const VAA_UPDATE_GUARDIAN_SET_2B: vector<u8> =
        x"01000000010100195f37abd29438c74db6e57bf527646b36fa96e36392221e869debe0e911f2f319abc0fd5c5a454da76fc0ffdd23a71a60bca40aa4289a841ad07f2964cde9290000bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000020100000000000000000000000000000000000000000000000000000000436f72650200000000000201befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe";
    const VAA_UPDATE_GUARDIAN_SET_EMPTY: vector<u8> =
        x"0100000000010098f9e45f836661d2932def9c74c587168f4f75d0282201ee6f5a98557e6212ff19b0f8881c2750646250f60dd5d565530779ecbf9442aa5ffc2d6afd7303aaa40000bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f72650200000000000100";

    #[test]
    fun test_update_guardian_set() {
        // Testing this method.
        use wormhole::update_guardian_set::{update_guardian_set};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `update_guardian_set`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_UPDATE_GUARDIAN_SET_1,
                &the_clock
            );
        let ticket = update_guardian_set::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        let new_index =
            update_guardian_set(&mut worm_state, receipt, &the_clock);
        assert!(new_index == 1, 0);

        let new_guardian_set =
            state::guardian_set_at(&worm_state, new_index);

        // Verify new guardian set index.
        assert!(state::guardian_set_index(&worm_state) == new_index, 0);
        assert!(
            guardian_set::index(new_guardian_set) == state::guardian_set_index(&worm_state),
            0
        );

        // Check that the guardians agree with what we expect.
        let guardians = guardian_set::guardians(new_guardian_set);
        let expected = vector[
            guardian::new(x"befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe"),
            guardian::new(x"88d7d8b32a9105d228100e72dffe2fae0705d31c"),
            guardian::new(x"58076f561cc62a47087b567c86f986426dfcd000"),
            guardian::new(x"bd6e9833490f8fa87c733a183cd076a6cbd29074"),
            guardian::new(x"b853fcf0a5c78c1b56d15fce7a154e6ebe9ed7a2"),
            guardian::new(x"af3503dbd2e37518ab04d7ce78b630f98b15b78a"),
            guardian::new(x"785632dea5609064803b1c8ea8bb2c77a6004bd1"),
            guardian::new(x"09a281a698c0f5ba31f158585b41f4f33659e54d"),
            guardian::new(x"3178443ab76a60e21690dbfb17f7f59f09ae3ea1"),
            guardian::new(x"647ec26ae49b14060660504f4da1c2059e1c5ab6"),
            guardian::new(x"810ac3d8e1258bd2f004a94ca0cd4c68fc1c0611"),
            guardian::new(x"80610e96d645b12f47ae5cf4546b18538739e90f"),
            guardian::new(x"2edb0d8530e31a218e72b9480202acbaeb06178d"),
            guardian::new(x"a78858e5e5c4705cdd4b668ffe3be5bae4867c9d"),
            guardian::new(x"5efe3a05efc62d60e1d19faeb56a80223cdd3472"),
            guardian::new(x"d791b7d32c05abb1cc00b6381fa0c4928f0c56fc"),
            guardian::new(x"14bc029b8809069093d712a3fd4dfab31963597e"),
            guardian::new(x"246ab29fc6ebedf2d392a51ab2dc5c59d0902a03"),
            guardian::new(x"132a84dfd920b35a3d0ba5f7a0635df298f9033e"),
        ];
        assert!(vector::length(&expected) == vector::length(guardians), 0);

        let cur = cursor::new(expected);
        let i = 0;
        while (!cursor::is_empty(&cur)) {
            let left = guardian::as_bytes(vector::borrow(guardians, i));
            let right = guardian::to_bytes(cursor::poke(&mut cur));
            assert!(left == right, 0);
            i = i + 1;
        };
        cursor::destroy_empty(cur);

        // Make sure old guardian set is still active.
        let old_guardian_set =
            state::guardian_set_at(&worm_state, new_index - 1);
        assert!(guardian_set::is_active(old_guardian_set, &the_clock), 0);

        // Fast forward time beyond expiration by
        // `guardian_set_seconds_to_live`.
        let tick_ms =
            (state::guardian_set_seconds_to_live(&worm_state) as u64) * 1000;
        clock::increment_for_testing(&mut the_clock, tick_ms + 1);

        // Now the old guardian set should be expired (because in the test setup
        // time to live is set to 2 epochs).
        assert!(!guardian_set::is_active(old_guardian_set, &the_clock), 0);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_update_guardian_set_after_upgrade() {
        // Testing this method.
        use wormhole::update_guardian_set::{update_guardian_set};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Upgrade.
        upgrade_wormhole(scenario);

        // Prepare test to execute `update_guardian_set`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_UPDATE_GUARDIAN_SET_1,
                &the_clock
            );
        let ticket = update_guardian_set::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        let new_index =
            update_guardian_set(&mut worm_state, receipt, &the_clock);
        assert!(new_index == 1, 0);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_OLD_GUARDIAN_SET_GOVERNANCE
    )]
    fun test_cannot_update_guardian_set_again_with_same_vaa() {
        // Testing this method.
        use wormhole::update_guardian_set::{update_guardian_set};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `update_guardian_set`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_UPDATE_GUARDIAN_SET_2A,
                &the_clock
            );
        let ticket = update_guardian_set::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        update_guardian_set(&mut worm_state, receipt, &the_clock);

        // Update guardian set again with new VAA.
        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_UPDATE_GUARDIAN_SET_2B,
                &the_clock
            );
        let ticket = update_guardian_set::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        let new_index =
            update_guardian_set(&mut worm_state, receipt, &the_clock);
        assert!(new_index == 2, 0);
        assert!(state::guardian_set_index(&worm_state) == 2, 0);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_UPDATE_GUARDIAN_SET_2A,
                &the_clock
            );
        let ticket = update_guardian_set::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        // You shall not pass!
        update_guardian_set(&mut worm_state, receipt, &the_clock);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = update_guardian_set::E_NO_GUARDIANS)]
    fun test_cannot_update_guardian_set_with_no_guardians() {
        // Testing this method.
        use wormhole::update_guardian_set::{update_guardian_set};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `update_guardian_set`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);


        // Show that the encoded number of guardians is zero.
        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_UPDATE_GUARDIAN_SET_EMPTY,
                &the_clock
            );
        let payload =
            governance_message::take_decree(vaa::payload(&verified_vaa));
        let cur = cursor::new(payload);

        let new_guardian_set_index = bytes::take_u32_be(&mut cur);
        assert!(new_guardian_set_index == 1, 0);

        let num_guardians = bytes::take_u8(&mut cur);
        assert!(num_guardians == 0, 0);

        cursor::destroy_empty(cur);

        let ticket = update_guardian_set::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        // You shall not pass!
        update_guardian_set(&mut worm_state, receipt, &the_clock);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_set_fee_outdated_version() {
        // Testing this method.
        use wormhole::update_guardian_set::{update_guardian_set};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `update_guardian_set`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut worm_state);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        state::migrate_version_test_only(
            &mut worm_state,
            version_control::previous_version_test_only(),
            version_control::next_version()
        );

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_UPDATE_GUARDIAN_SET_1,
                &the_clock
            );
        let ticket = update_guardian_set::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        // You shall not pass!
        update_guardian_set(&mut worm_state, receipt, &the_clock);

        abort 42
    }
}
