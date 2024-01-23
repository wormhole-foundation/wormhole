// SPDX-License-Identifier: Apache 2

/// This module implements a public method intended to be called after an
/// upgrade has been committed. The purpose is to add one-off migration logic
/// that would alter Wormhole `State`.
///
/// Included in migration is the ability to ensure that breaking changes for
/// any of Wormhole's methods by enforcing the current build version as their
/// required minimum version.
module wormhole::migrate {
    use sui::clock::{Clock};
    use sui::object::{ID};

    use wormhole::governance_message::{Self};
    use wormhole::state::{Self, State};
    use wormhole::upgrade_contract::{Self};
    use wormhole::vaa::{Self};

    /// Event reflecting when `migrate` is successfully executed.
    struct MigrateComplete has drop, copy {
        package: ID
    }

    /// Execute migration logic. See `wormhole::migrate` description for more
    /// info.
    public fun migrate(
        wormhole_state: &mut State,
        upgrade_vaa_buf: vector<u8>,
        the_clock: &Clock
    ) {
        state::migrate__v__0_2_0(wormhole_state);

        // Perform standard migrate.
        handle_migrate(wormhole_state, upgrade_vaa_buf, the_clock);

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
        wormhole_state: &mut State,
        upgrade_vaa_buf: vector<u8>,
        the_clock: &Clock
    ) {
        // Update the version first.
        //
        // See `version_control` module for hard-coded configuration.
        state::migrate_version(wormhole_state);

        // This VAA needs to have been used for upgrading this package.
        //
        // NOTE: All of the following methods have protections to make sure that
        // the current build is used. Given that we officially migrated the
        // version as the first call of `migrate`, these should be successful.

        // First we need to check that `parse_and_verify` still works.
        let verified_vaa =
            vaa::parse_and_verify(wormhole_state, upgrade_vaa_buf, the_clock);

        // And governance methods.
        let ticket = upgrade_contract::authorize_governance(wormhole_state);
        let receipt =
            governance_message::verify_vaa(
                wormhole_state,
                verified_vaa,
                ticket
            );

        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(wormhole_state);

        // Check if build digest is the current one.
        let digest =
            upgrade_contract::take_digest(
                governance_message::payload(&receipt)
            );
        state::assert_authorized_digest(&latest_only, wormhole_state, digest);
        governance_message::destroy(receipt);

        // Finally emit an event reflecting a successful migrate.
        let package = state::current_package(&latest_only, wormhole_state);
        sui::event::emit(MigrateComplete { package });
    }

    #[test_only]
    public fun set_up_migrate(wormhole_state: &mut State) {
        state::reverse_migrate__v__dummy(wormhole_state);
    }
}

#[test_only]
module wormhole::migrate_tests {
    use sui::test_scenario::{Self};

    use wormhole::state::{Self};
    use wormhole::wormhole_scenario::{
        person,
        return_clock,
        return_state,
        set_up_wormhole,
        take_clock,
        take_state,
        upgrade_wormhole
    };

    const UPGRADE_VAA: vector<u8> =
        x"01000000000100db695668c0c91f4df6e4106dcb912d9062898fd976d631ff1c1b4109ccd203b43cd2419c7d9a191f8d42a780419e63307aacc93080d8629c6c03061c52becf1d0100bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f726501001500000000000000000000000000000000000000000000006e6577206275696c64";

    #[test]
    fun test_migrate() {
        use wormhole::migrate::{migrate};

        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole.
        let wormhole_message_fee = 350;
        set_up_wormhole(scenario, wormhole_message_fee);

        // Next transaction should be conducted as an ordinary user.
        test_scenario::next_tx(scenario, user);

        // Upgrade (digest is just b"new build") for testing purposes.
        upgrade_wormhole(scenario);

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Set up migrate (which prepares this package to be the same state as
        // a previous release).
        wormhole::migrate::set_up_migrate(&mut worm_state);

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut worm_state);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        migrate(&mut worm_state, UPGRADE_VAA, &the_clock);

        // Make sure we emitted an event.
        let effects = test_scenario::next_tx(scenario, user);
        assert!(test_scenario::num_user_events(&effects) == 1, 0);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_INCORRECT_OLD_VERSION)]
    /// ^ This expected error may change depending on the migration. In most
    /// cases, this will abort with `wormhole::package_utils::E_INCORRECT_OLD_VERSION`.
    fun test_cannot_migrate_again() {
        use wormhole::migrate::{migrate};

        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole.
        let wormhole_message_fee = 350;
        set_up_wormhole(scenario, wormhole_message_fee);

        // Next transaction should be conducted as an ordinary user.
        test_scenario::next_tx(scenario, user);

        // Upgrade (digest is just b"new build") for testing purposes.
        upgrade_wormhole(scenario);

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Set up migrate (which prepares this package to be the same state as
        // a previous release).
        wormhole::migrate::set_up_migrate(&mut worm_state);

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut worm_state);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        migrate(&mut worm_state, UPGRADE_VAA, &the_clock);

        // Make sure we emitted an event.
        let effects = test_scenario::next_tx(scenario, user);
        assert!(test_scenario::num_user_events(&effects) == 1, 0);

        // You shall not pass!
        migrate(&mut worm_state, UPGRADE_VAA, &the_clock);

        abort 42
    }
}
