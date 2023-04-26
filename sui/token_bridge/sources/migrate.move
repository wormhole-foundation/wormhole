// SPDX-License-Identifier: Apache 2

/// This module implements a public method intended to be called after an
/// upgrade has been commited. The purpose is to add one-off migration logic
/// that would alter Wormhole `State`.
///
/// Included in migration is the ability to ensure that breaking changes for
/// any of Wormhole's methods by enforcing the current build version as their
/// required minimum version.
module token_bridge::migrate {
    use wormhole::governance_message::{Self, DecreeReceipt};
    //use wormhole::vaa::{VAA};

    use token_bridge::state::{Self, State};
    use token_bridge::upgrade_contract::{Self};

    /// Execute migration logic. See `wormhole::migrate` description for more
    /// info.
    public fun migrate(token_bridge_state: &mut State, receipt: DecreeReceipt) {
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

    fun handle_migrate(token_bridge_state: &mut State, receipt: DecreeReceipt) {
        // Update the version first.
        //
        // See `version_control` module for hard-coded configuration.
        state::migrate_version(token_bridge_state);

        // This state capability ensures that the current build version is used.
        let cap = state::new_cap(token_bridge_state);

        // Check if build digest is the current one.
        let digest =
            upgrade_contract::take_digest(
                governance_message::payload(&receipt)
            );
        state::assert_current_digest(&cap, token_bridge_state, digest);
        governance_message::destroy(receipt);
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
    use token_bridge::version_control::{Self};
    use token_bridge::token_bridge_scenario::{
        person,
        return_state,
        set_up_wormhole_and_token_bridge,
        take_state,
        upgrade_token_bridge
    };

    const UPGRADE_VAA: vector<u8> =
        x"0100000000010011f0d45907077bf2baf7da1872a48b8fdea161b14b9979d758c0f2bffa9f184d56f0c70fc8e2893d69861552bf074549969a155570313476f70d3ac4570b31860100bc614e0000000000010000000000000000000000000000000000000000000000000000000000000004000000000000000101000000000000000000000000000000000000000000546f6b656e42726964676501001500000000000000000000000000000000000000000000006e6577206275696c64";

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

        // First migrate to V_DUMMY to simulate migrating from this to the
        // existing build version.
        state::migrate_version_test_only(
            &mut token_bridge_state,
            version_control::first(),
            version_control::dummy()
        );

        let verified_vaa = parse_and_verify_vaa(scenario, UPGRADE_VAA);
        let ticket =
            upgrade_contract::authorize_governance(&token_bridge_state);
        let receipt =
            verify_governance_vaa(scenario, verified_vaa, ticket);
        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        migrate(&mut token_bridge_state, receipt);

        // Clean up.
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_INCORRECT_OLD_VERSION)]
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

        // First migrate to V_DUMMY to simulate migrating from this to the
        // existing build version.
        state::migrate_version_test_only(
            &mut token_bridge_state,
            version_control::first(),
            version_control::dummy()
        );

        let verified_vaa = parse_and_verify_vaa(scenario, UPGRADE_VAA);
        let ticket =
            upgrade_contract::authorize_governance(&token_bridge_state);
        let receipt =
            verify_governance_vaa(scenario, verified_vaa, ticket);
        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        migrate(&mut token_bridge_state, receipt);

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        let verified_vaa = parse_and_verify_vaa(scenario, UPGRADE_VAA);
        let ticket =
            upgrade_contract::authorize_governance(&token_bridge_state);
        let receipt =
            verify_governance_vaa(scenario, verified_vaa, ticket);
        // You shall not pass!
        migrate(&mut token_bridge_state, receipt);

        // Clean up.
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }
}
