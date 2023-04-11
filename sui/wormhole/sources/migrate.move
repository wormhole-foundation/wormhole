// SPDX-License-Identifier: Apache 2

/// This module implements a public method intended to be called after an
/// upgrade has been commited. The purpose is to add one-off migration logic
/// that would alter Wormhole `State`.
///
/// Included in migration is the ability to ensure that breaking changes for
/// any of Wormhole's methods by enforcing the current build version as their
/// required minimum version.
module wormhole::migrate {
    use wormhole::state::{Self, State};
    use wormhole::version_control::{Migrate as MigrateControl};

    // This import is only used when `state::require_current_version` is used.
    //use wormhole::version_control::{Self as control};

    /// Execute migration logic. See `wormhole::migrate` description for more
    /// info.
    public fun migrate(wormhole_state: &mut State) {
        state::check_minimum_requirement<MigrateControl>(wormhole_state);

        // Wormhole `State` destroys the `MigrateTicket` as the final step.
        state::consume_migrate_ticket(wormhole_state);

        ////////////////////////////////////////////////////////////////////////
        //
        // If there are any methods that require the current build, we need to
        // explicity require them here.
        //
        // Calls to `require_current_version` are commented out for convenience.
        //
        ////////////////////////////////////////////////////////////////////////

        // state::require_current_version<control::NewEmitter>(wormhole_state);
        // state::require_current_version<control::ParseAndVerify>(wormhole_state);
        // state::require_current_version<control::PublishMessage>(wormhole_state);
        // state::require_current_version<control::SetFee>(wormhole_state);
        // state::require_current_version<control::TransferFee>(wormhole_state);
        // state::require_current_version<control::UpdateGuardianSet>(wormhole_state);

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
        // If the nature of your migration absolutely requires the migration to
        // happen before certain other functionality is available, then guard
        // that functionality with the `assert!` from above.
        //
        ////////////////////////////////////////////////////////////////////////



        ////////////////////////////////////////////////////////////////////////
        // Done.
        ////////////////////////////////////////////////////////////////////////
    }
}
