// SPDX-License-Identifier: Apache 2

/// This module implements the emergency pause mechanism for the Token Bridge.
///
/// Authority is held via capability objects rather than EOA addresses. Two
/// capabilities exist: `PauserCap` (gates `pause`) and `UnpauserCap` (gates
/// `unpause`). The Token Bridge `State` stores the object id of the single
/// ACTIVE capability for each role; `pause`/`unpause` take the capability and
/// check `object::id(cap)` against the stored active id.
///
/// Why capabilities instead of `tx_context::sender`:
/// - `tx_context::sender` is always the transaction signer, i.e. an EOA. A Sui
///   smart contract (governance program, multisig module) can never be the
///   sender, so it could never hold the role. A capability is an object, which
///   a contract *can* own — so contracts can be pausers. This mirrors
///   `wormhole::emitter::EmitterCap`, whose identity is also its object id.
///
/// Lifecycle (governance-driven mint):
/// - Capabilities are minted ONLY by `token_bridge::set_pauser_addresses` while
///   handling a `SetPauserAddresses` governance VAA. The VAA encodes the OWNER
///   address that should receive the cap. The handler mints the cap, transfers
///   it to that owner, and records the new cap's id as active in `State`.
/// - Because the handler mints and transfers, the active cap is ALWAYS an owned
///   object — it can never be a shared object, so `pause`/`unpause` (which take
///   `&cap`) cannot be invoked by anyone other than the cap's owner.
/// - Rotation, two ways:
///   1. Governance mints a NEW cap for a new owner via `SetPauserAddresses`;
///      the previously active cap becomes inert (its id no longer matches the
///      recorded active id) — no clawback required.
///   2. The current holder transfers the active cap directly (the cap has
///      `store`, so this needs no governance). The cap keeps its id, so it
///      stays active — authority simply moves with the object. If a cap is
///      compromised, governance can always revoke it via path 1.
/// - Unassign: a zero owner records `@0x0` as active and mints nothing; the
///   corresponding entry point then reverts as not-configured.
///
/// Neither `pause` nor `unpause` is guarded by `assert_not_paused` (both must be
/// callable regardless of pause state — `pause` is a no-op when already paused,
/// `unpause` must obviously work when paused).
module token_bridge::pause {
    use sui::object::{Self, ID, UID};
    use sui::tx_context::{Self, TxContext};

    use token_bridge::state::{Self, State};

    friend token_bridge::set_pauser_addresses;

    /// The pauser role is unassigned (active id is @0x0).
    const E_PAUSER_NOT_CONFIGURED: u64 = 0;
    /// The unpauser role is unassigned (active id is @0x0).
    const E_UNPAUSER_NOT_CONFIGURED: u64 = 1;
    /// Provided `PauserCap` is not the active pauser.
    const E_NOT_PAUSER: u64 = 2;
    /// Provided `UnpauserCap` is not the active unpauser.
    const E_NOT_UNPAUSER: u64 = 3;

    /// Capability whose holder may `pause` the bridge, while its id is the
    /// active pauser. Minted and transferred by governance.
    struct PauserCap has key, store {
        id: UID
    }

    /// Capability whose holder may `unpause` the bridge, while its id is the
    /// active unpauser. Minted and transferred by governance.
    struct UnpauserCap has key, store {
        id: UID
    }

    /// Event emitted when the bridge is paused.
    struct Paused has drop, copy {
        /// Object id of the cap used to pause.
        cap: ID,
        /// Transaction signer (for audit; not the authority).
        sender: address
    }

    /// Event emitted when the bridge is unpaused.
    struct Unpaused has drop, copy {
        /// Object id of the cap used to unpause.
        cap: ID,
        /// Transaction signer (for audit; not the authority).
        sender: address
    }

    /// Mint a new `PauserCap`. Only callable by `set_pauser_addresses` while
    /// handling a governance VAA. The handler is responsible for transferring
    /// the cap to its owner and recording its id as active.
    public(friend) fun new_pauser_cap(ctx: &mut TxContext): PauserCap {
        PauserCap { id: object::new(ctx) }
    }

    /// Mint a new `UnpauserCap`. Only callable by `set_pauser_addresses`.
    public(friend) fun new_unpauser_cap(ctx: &mut TxContext): UnpauserCap {
        UnpauserCap { id: object::new(ctx) }
    }

    /// The object id of a `PauserCap`, as an `address` (the value recorded as
    /// the active pauser in `State`).
    public fun pauser_cap_id(cap: &PauserCap): address {
        object::id_to_address(&object::id(cap))
    }

    /// The object id of an `UnpauserCap`, as an `address`.
    public fun unpauser_cap_id(cap: &UnpauserCap): address {
        object::id_to_address(&object::id(cap))
    }

    /// Destroy a `PauserCap`. The active pauser is tracked by id in `State`, so
    /// destroying the active cap leaves the role uncallable until governance
    /// mints and records a new one.
    public fun destroy_pauser_cap(cap: PauserCap) {
        let PauserCap { id } = cap;
        object::delete(id);
    }

    /// Destroy an `UnpauserCap`.
    public fun destroy_unpauser_cap(cap: UnpauserCap) {
        let UnpauserCap { id } = cap;
        object::delete(id);
    }

    /// Pause the token bridge. Requires the active `PauserCap`.
    /// Aborts if the pauser role is unassigned (@0x0).
    public fun pause(
        token_bridge_state: &mut State,
        cap: &PauserCap,
        ctx: &TxContext
    ) {
        // Version check.
        let latest_only = state::assert_latest_only(token_bridge_state);

        let configured = state::pauser(token_bridge_state);
        assert!(configured != @0x0, E_PAUSER_NOT_CONFIGURED);

        let cap_id = object::id(cap);
        assert!(object::id_to_address(&cap_id) == configured, E_NOT_PAUSER);

        state::set_paused(&latest_only, token_bridge_state, true);

        sui::event::emit(Paused { cap: cap_id, sender: tx_context::sender(ctx) });
    }

    /// Unpause the token bridge. Requires the active `UnpauserCap`.
    /// Aborts if the unpauser role is unassigned (@0x0).
    /// NOT guarded by assert_not_paused (must be callable when paused).
    public fun unpause(
        token_bridge_state: &mut State,
        cap: &UnpauserCap,
        ctx: &TxContext
    ) {
        // Version check.
        let latest_only = state::assert_latest_only(token_bridge_state);

        let configured = state::unpauser(token_bridge_state);
        assert!(configured != @0x0, E_UNPAUSER_NOT_CONFIGURED);

        let cap_id = object::id(cap);
        assert!(object::id_to_address(&cap_id) == configured, E_NOT_UNPAUSER);

        state::set_paused(&latest_only, token_bridge_state, false);

        sui::event::emit(Unpaused { cap: cap_id, sender: tx_context::sender(ctx) });
    }

    #[test_only]
    public fun new_pauser_cap_test_only(ctx: &mut TxContext): PauserCap {
        new_pauser_cap(ctx)
    }

    #[test_only]
    public fun new_unpauser_cap_test_only(ctx: &mut TxContext): UnpauserCap {
        new_unpauser_cap(ctx)
    }

    #[test_only]
    public fun destroy_pauser_cap_test_only(cap: PauserCap) {
        destroy_pauser_cap(cap);
    }

    #[test_only]
    public fun destroy_unpauser_cap_test_only(cap: UnpauserCap) {
        destroy_unpauser_cap(cap);
    }
}
