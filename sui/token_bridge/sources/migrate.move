// SPDX-License-Identifier: Apache 2

/// This module implements a public method intended to be called after an
/// upgrade has been committed. The purpose is to add one-off migration logic
/// that would alter Token Bridge `State`.
///
/// Included in migration is the ability to ensure that breaking changes for
/// any of Token Bridge's methods by enforcing the current build version as
/// their required minimum version.
module token_bridge::migrate {
    use sui::object::{ID};
    use wormhole::governance_message::{Self, DecreeReceipt};

    use token_bridge::state::{Self, State};
    use token_bridge::upgrade_contract::{Self};

    /// Event reflecting when `migrate` is successfully executed.
    struct MigrateComplete has drop, copy {
        package: ID
    }

    /// Execute migration logic. See `token_bridge::migrate` description for
    /// more info.
    public fun migrate(
        token_bridge_state: &mut State,
        receipt: DecreeReceipt<upgrade_contract::GovernanceWitness>
    ) {
        state::migrate__v__0_2_0(token_bridge_state);

        // Perform standard migrate.
        handle_migrate(token_bridge_state, receipt);

        ////////////////////////////////////////////////////////////////////////
        //
        // NOTE: Put any one-off migration logic here.
        //
        // Most upgrades likely won't need to do anything, in which case the
        // rest of this function's body may be empty. Make sure to delete it
        // after the migration has gone through successfully.
        //
        // WARNING: The migration does *not* proceed atomically with the
        // upgrade (as they are done in separate transactions).
        // If the nature of this migration absolutely requires the migration to
        // happen before certain other functionality is available, then guard
        // that functionality with the `assert!` from above.
        //
        ////////////////////////////////////////////////////////////////////////

        ////////////////////////////////////////////////////////////////////////
    }

    fun handle_migrate(
        token_bridge_state: &mut State,
        receipt: DecreeReceipt<upgrade_contract::GovernanceWitness>
    ) {
        // Update the version first.
        //
        // See `version_control` module for hard-coded configuration.
        state::migrate_version(token_bridge_state);

        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        // Check if build digest is the current one.
        let digest =
            upgrade_contract::take_digest(
                governance_message::payload(&receipt)
            );
        state::assert_authorized_digest(
            &latest_only,
            token_bridge_state,
            digest
        );
        governance_message::destroy(receipt);

        // Finally emit an event reflecting a successful migrate.
        let package = state::current_package(&latest_only, token_bridge_state);
        sui::event::emit(MigrateComplete { package });
    }

    #[test_only]
    public fun set_up_migrate(token_bridge_state: &mut State) {
        state::reverse_migrate__v__dummy(token_bridge_state);
    }
}

#[test_only]
module token_bridge::migrate_tests {
    use sui::test_scenario::{Self};
    use wormhole::wormhole_scenario::{
        parse_and_verify_vaa,
        verify_governance_vaa
    };

    use token_bridge::state::{Self};
    use token_bridge::upgrade_contract::{Self};
    use token_bridge::token_bridge_scenario::{
        person,
        return_state,
        set_up_wormhole_and_token_bridge,
        take_state,
        upgrade_token_bridge
    };

    const UPGRADE_VAA: vector<u8> =
        x"010000000001005b18d7710c442414435162dc2b46a421c3018a7ff03290eff112a828b7927e4a6a624174cb8385210f4684ac2dbde6e01e4046218f7f245af53e85c97a48e21a0100bc614e0000000000010000000000000000000000000000000000000000000000000000000000000004000000000000000101000000000000000000000000000000000000000000546f6b656e42726964676502001500000000000000000000000000000000000000000000006e6577206275696c64";

    #[test]
    fun test_migrate() {
        use token_bridge::migrate::{migrate};

        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole.
        let wormhole_message_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_message_fee);

        // Next transaction should be conducted as an ordinary user.
        test_scenario::next_tx(scenario, user);

        // Upgrade (digest is just b"new build") for testing purposes.
        upgrade_token_bridge(scenario);

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);

        // Set up migrate (which prepares this package to be the same state as
        // a previous release).
        token_bridge::migrate::set_up_migrate(&mut token_bridge_state);

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut token_bridge_state);

        let verified_vaa = parse_and_verify_vaa(scenario, UPGRADE_VAA);
        let ticket =
            upgrade_contract::authorize_governance(&token_bridge_state);
        let receipt =
            verify_governance_vaa(scenario, verified_vaa, ticket);
        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        migrate(&mut token_bridge_state, receipt);

        // Make sure we emitted an event.
        let effects = test_scenario::next_tx(scenario, user);
        assert!(test_scenario::num_user_events(&effects) == 1, 0);

        // Clean up.
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_INCORRECT_OLD_VERSION)]
    /// ^ This expected error may change depending on the migration. In most
    /// cases, this will abort with `wormhole::package_utils::E_INCORRECT_OLD_VERSION`.
    fun test_cannot_migrate_again() {
        use token_bridge::migrate::{migrate};

        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole.
        let wormhole_message_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_message_fee);

        // Next transaction should be conducted as an ordinary user.
        test_scenario::next_tx(scenario, user);

        // Upgrade (digest is just b"new build") for testing purposes.
        upgrade_token_bridge(scenario);

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);

        // Set up migrate (which prepares this package to be the same state as
        // a previous release).
        token_bridge::migrate::set_up_migrate(&mut token_bridge_state);

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut token_bridge_state);

        let verified_vaa = parse_and_verify_vaa(scenario, UPGRADE_VAA);
        let ticket =
            upgrade_contract::authorize_governance(&token_bridge_state);
        let receipt =
            verify_governance_vaa(scenario, verified_vaa, ticket);
        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        migrate(&mut token_bridge_state, receipt);

        // Make sure we emitted an event.
        let effects = test_scenario::next_tx(scenario, user);
        assert!(test_scenario::num_user_events(&effects) == 1, 0);

        let verified_vaa = parse_and_verify_vaa(scenario, UPGRADE_VAA);
        let ticket =
            upgrade_contract::authorize_governance(&token_bridge_state);
        let receipt =
            verify_governance_vaa(scenario, verified_vaa, ticket);
        // You shall not pass!
        migrate(&mut token_bridge_state, receipt);

        abort 42
    }
}
