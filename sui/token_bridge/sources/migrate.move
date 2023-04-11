// SPDX-License-Identifier: Apache 2

/// This module implements a public method intended to be called after an
/// upgrade has been commited. The purpose is to add one-off migration logic
/// that would alter Token Bridge `State`.
///
/// Included in migration is the ability to ensure that breaking changes for
/// any of Token Bridge's methods by enforcing the current build version as
/// their required minimum version.
module token_bridge::migrate {
    use token_bridge::state::{Self, State};
    use token_bridge::version_control::{Migrate as MigrateControl};

    // This import is only used when `state::require_current_version` is used.
    //use token_bridge::version_control::{Self as control};

    /// Execute migration logic. See `token_bridge::migrate` description for
    /// more info.
    public fun migrate(token_bridge_state: &mut State) {
        state::check_minimum_requirement<MigrateControl>(token_bridge_state);

        // Token Bridge `State` destroys the `MigrateTicket` as the final step.
        state::consume_migrate_ticket(token_bridge_state);

        ////////////////////////////////////////////////////////////////////////
        //
        // If there are any methods that require the current build, we need to
        // explicity require them here.
        //
        // Calls to `require_current_version` are commented out for convenience.
        //
        ////////////////////////////////////////////////////////////////////////

        // state::require_current_version<control::AttestToken>(token_bridge_state);
        // state::require_current_version<control::CompleteTransfer>(token_bridge_state);
        // state::require_current_version<control::CompleteTransferWithPayload>(token_bridge_state);
        // state::require_current_version<control::CreateWrapped>(token_bridge_state);
        // state::require_current_version<control::RegisterChain>(token_bridge_state);
        // state::require_current_version<control::TransferTokens>(token_bridge_state);
        // state::require_current_version<control::TransferTokensWithPayload>(token_bridge_state);
        // state::require_current_version<control::Vaa>(token_bridge_state);

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
