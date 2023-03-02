module wormhole::migrate {
    use wormhole::state::{Self, State};

    const E_CANNOT_MIGRATE: u64 = 0;

    public entry fun migrate(
        wormhole_state: &mut State,
    ) {
        assert!(state::can_migrate(wormhole_state), E_CANNOT_MIGRATE);
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

        // TODO: write an example of how a particular method can be disabled
        // with a breaking change.
        //
        ////////////////////////////////////////////////////////////////////////



        ////////////////////////////////////////////////////////////////////////
        // Done.
        state::disable_migration(wormhole_state);
    }
}
