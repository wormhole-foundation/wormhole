// SPDX-License-Identifier: Apache 2

/// This module implements the `pause` and `unpause` entry functions for the
/// Token Bridge emergency pause mechanism.
///
/// - `pause` can only be called by the configured pauser address.
/// - `unpause` can only be called by the configured unpauser address.
/// - Neither function is guarded by `assert_not_paused` (both must be callable
///   regardless of pause state — `pause` is a no-op when already paused,
///   `unpause` must obviously work when paused).
module token_bridge::pause {
    use sui::tx_context::{Self, TxContext};

    use token_bridge::state::{Self, State};

    /// The configured pauser is the zero address (unassigned).
    const E_PAUSER_NOT_CONFIGURED: u64 = 0;
    /// The configured unpauser is the zero address (unassigned).
    const E_UNPAUSER_NOT_CONFIGURED: u64 = 1;
    /// Caller is not the configured pauser.
    const E_NOT_PAUSER: u64 = 2;
    /// Caller is not the configured unpauser.
    const E_NOT_UNPAUSER: u64 = 3;

    /// Event emitted when the bridge is paused.
    struct Paused has drop, copy {
        sender: address
    }

    /// Event emitted when the bridge is unpaused.
    struct Unpaused has drop, copy {
        sender: address
    }

    /// Pause the token bridge. Only callable by the configured pauser.
    /// Aborts if pauser is unassigned (@0x0).
    public fun pause(
        token_bridge_state: &mut State,
        ctx: &TxContext
    ) {
        // Version check.
        let latest_only = state::assert_latest_only(token_bridge_state);

        let configured_pauser = state::pauser(token_bridge_state);
        assert!(configured_pauser != @0x0, E_PAUSER_NOT_CONFIGURED);

        let sender = tx_context::sender(ctx);
        assert!(sender == configured_pauser, E_NOT_PAUSER);

        state::set_paused(&latest_only, token_bridge_state, true);

        sui::event::emit(Paused { sender });
    }

    /// Unpause the token bridge. Only callable by the configured unpauser.
    /// Aborts if unpauser is unassigned (@0x0).
    /// NOT guarded by assert_not_paused (must be callable when paused).
    public fun unpause(
        token_bridge_state: &mut State,
        ctx: &TxContext
    ) {
        // Version check.
        let latest_only = state::assert_latest_only(token_bridge_state);

        let configured_unpauser = state::unpauser(token_bridge_state);
        assert!(configured_unpauser != @0x0, E_UNPAUSER_NOT_CONFIGURED);

        let sender = tx_context::sender(ctx);
        assert!(sender == configured_unpauser, E_NOT_UNPAUSER);

        state::set_paused(&latest_only, token_bridge_state, false);

        sui::event::emit(Unpaused { sender });
    }
}
