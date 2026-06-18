// SPDX-License-Identifier: Apache 2

/// This module implements the emergency pause mechanism for the Token Bridge.
///
/// Authority is held via capability objects rather than EOA addresses. Three
/// capabilities exist: `PauserCap` (gates `pause`), `FreezerCap` (gates
/// `freeze`), and `UnpauserCap` (gates `unpause`). The Token Bridge `State`
/// stores the object id of the single ACTIVE capability for each role; the
/// entry points take the capability and check `object::id(cap)` against the
/// stored active id.
///
/// Why capabilities instead of `tx_context::sender`:
/// - `tx_context::sender` is always the transaction signer, i.e. an EOA. A Sui
///   smart contract (governance program, multisig module) can never be the
///   sender, so it could never hold the role. A capability is an object, which
///   a contract *can* own — so contracts can hold these roles. This mirrors
///   `wormhole::emitter::EmitterCap`, whose identity is also its object id.
///
/// Pause model (whitepaper 0003):
/// - State is a boolean `paused` (authoritative) plus a `pauseExpiry` timestamp
///   (ms). `paused` is the only thing the hot-path `assert_not_paused` reads; a
///   pause is NEVER lifted silently by the passage of time — only an explicit
///   `unpause` or `unpause_expired` call clears it.
/// - `pause` (PauserCap): set paused, push `pauseExpiry` to `now + PAUSE_DURATION`
///   (5 days). Repeatable; each call extends the window. NEVER reduces a
///   `pauseExpiry` already further in the future (a lower-trust pauser cannot
///   curtail a freeze). Not idempotent.
/// - `freeze_bridge` (FreezerCap): set paused, set `pauseExpiry` to the maximum
///   timestamp. The higher-trust counterpart; a frozen bridge is not
///   permissionlessly unpausable in practice and can only be lifted by the
///   unpauser. Idempotent. (Named `freeze_bridge` because `freeze` is reserved
///   in Move; it is the spec's `freeze`.)
/// - `unpause` (UnpauserCap): clear paused, set `pauseExpiry` to `now`. The
///   privileged path to lift any pause (including a freeze) early. Reverts if
///   not currently paused. Recording `now` (rather than 0) leaves on-chain
///   evidence and brings a stale freeze expiry down so a later `pause` works.
/// - `unpause_expired` (PERMISSIONLESS, no cap): clear paused once
///   `now >= pauseExpiry`. Bounds a pauser-initiated pause to PAUSE_DURATION
///   without requiring the unpauser to act.
///
/// Time comes from the shared `sui::clock::Clock` object (there is no
/// `block.timestamp` in Move), so the entry points take a `&Clock`.
///
/// Lifecycle (governance-driven mint):
/// - Capabilities are minted ONLY by `token_bridge::set_pauser_addresses` while
///   handling a `SetPauserAddresses` governance VAA. The VAA encodes the OWNER
///   address that should receive each cap. The handler mints the cap, transfers
///   it to that owner, and records the new cap's id as active in `State`.
/// - Because the handler mints and transfers, the active cap is ALWAYS an owned
///   object — it can never be a shared object, so the entry points (which take
///   `&cap`) cannot be invoked by anyone other than the cap's owner.
/// - Rotation, two ways: (1) governance mints a NEW cap for a new owner,
///   deprecating the old one (its id no longer matches); (2) the holder
///   transfers the active cap directly (it has `store`), authority moving with
///   the object. A compromised cap is always revocable via path (1).
/// - Unassign: a `none` owner records `none` as active and mints nothing; the
///   corresponding entry point then reverts as not-configured.
module token_bridge::pause {
    use std::option::{Self};
    use sui::clock::{Self, Clock};
    use sui::object::{Self, ID, UID};
    use sui::tx_context::{Self, TxContext};

    use token_bridge::state::{Self, State};

    friend token_bridge::set_pauser_addresses;

    /// The pauser role is unassigned (active id is `none`).
    const E_PAUSER_NOT_CONFIGURED: u64 = 0;
    /// The unpauser role is unassigned (active id is `none`).
    const E_UNPAUSER_NOT_CONFIGURED: u64 = 1;
    /// Provided `PauserCap` is not the active pauser.
    const E_NOT_PAUSER: u64 = 2;
    /// Provided `UnpauserCap` is not the active unpauser.
    const E_NOT_UNPAUSER: u64 = 3;
    /// The freezer role is unassigned (active id is `none`).
    const E_FREEZER_NOT_CONFIGURED: u64 = 4;
    /// Provided `FreezerCap` is not the active freezer.
    const E_NOT_FREEZER: u64 = 5;
    /// `unpause`/`unpause_expired` called while the bridge is not paused.
    const E_NOT_PAUSED: u64 = 6;
    /// `unpause_expired` called before `pauseExpiry`.
    const E_NOT_EXPIRED: u64 = 7;

    /// Temporary-pause duration: 5 days, in milliseconds (Sui Clock is ms).
    const PAUSE_DURATION_MS: u64 = 432_000_000;
    /// Maximum representable timestamp; used by `freeze_bridge`.
    const MAX_TIMESTAMP_MS: u64 = 0xFFFFFFFFFFFFFFFF;

    /// Capability whose holder may `pause` the bridge, while its id is the
    /// active pauser. Minted and transferred by governance.
    struct PauserCap has key, store {
        id: UID
    }

    /// Capability whose holder may `freeze` the bridge, while its id is the
    /// active freezer. Minted and transferred by governance.
    struct FreezerCap has key, store {
        id: UID
    }

    /// Capability whose holder may `unpause` the bridge, while its id is the
    /// active unpauser. Minted and transferred by governance.
    struct UnpauserCap has key, store {
        id: UID
    }

    /// Event emitted when the bridge is paused via `pause`.
    struct Paused has drop, copy {
        /// Object id of the cap used to pause.
        cap: ID,
        /// Transaction signer (for audit; not the authority).
        sender: address,
        /// Timestamp (ms) at which the pause becomes permissionlessly liftable.
        pause_expiry: u64
    }

    /// Event emitted when the bridge is frozen via `freeze_bridge`.
    struct Frozen has drop, copy {
        /// Object id of the cap used to freeze.
        cap: ID,
        /// Transaction signer (for audit; not the authority).
        sender: address,
        /// Timestamp (ms) the pause runs until — the maximum timestamp for a
        /// freeze. Included so "paused until X" events are uniform across
        /// `Paused`/`Frozen`.
        pause_expiry: u64
    }

    /// Event emitted when the bridge is unpaused via `unpause`.
    struct Unpaused has drop, copy {
        /// Object id of the cap used to unpause.
        cap: ID,
        /// Transaction signer (for audit; not the authority).
        sender: address
    }

    /// Event emitted when the bridge is unpaused via the permissionless
    /// `unpause_expired`.
    struct UnpauseExpired has drop, copy {
        /// Transaction signer that lifted the expired pause.
        sender: address
    }

    /// Mint a new `PauserCap`. Only callable by `set_pauser_addresses` while
    /// handling a governance VAA. The handler is responsible for transferring
    /// the cap to its owner and recording its id as active.
    public(friend) fun new_pauser_cap(ctx: &mut TxContext): PauserCap {
        PauserCap { id: object::new(ctx) }
    }

    /// Mint a new `FreezerCap`. Only callable by `set_pauser_addresses`.
    public(friend) fun new_freezer_cap(ctx: &mut TxContext): FreezerCap {
        FreezerCap { id: object::new(ctx) }
    }

    /// Mint a new `UnpauserCap`. Only callable by `set_pauser_addresses`.
    public(friend) fun new_unpauser_cap(ctx: &mut TxContext): UnpauserCap {
        UnpauserCap { id: object::new(ctx) }
    }

    /// The object id of a `PauserCap` (the value recorded as the active pauser
    /// in `State`).
    public fun pauser_cap_id(cap: &PauserCap): ID {
        object::id(cap)
    }

    /// The object id of a `FreezerCap`.
    public fun freezer_cap_id(cap: &FreezerCap): ID {
        object::id(cap)
    }

    /// The object id of an `UnpauserCap`.
    public fun unpauser_cap_id(cap: &UnpauserCap): ID {
        object::id(cap)
    }

    /// Destroy a `PauserCap`. The active pauser is tracked by id in `State`, so
    /// destroying the active cap leaves the role uncallable until governance
    /// mints and records a new one.
    public fun destroy_pauser_cap(cap: PauserCap) {
        let PauserCap { id } = cap;
        object::delete(id);
    }

    /// Destroy a `FreezerCap`.
    public fun destroy_freezer_cap(cap: FreezerCap) {
        let FreezerCap { id } = cap;
        object::delete(id);
    }

    /// Destroy an `UnpauserCap`.
    public fun destroy_unpauser_cap(cap: UnpauserCap) {
        let UnpauserCap { id } = cap;
        object::delete(id);
    }

    /// Temporarily pause the token bridge. Requires the active `PauserCap`.
    /// Sets `paused` and pushes `pauseExpiry` to `now + PAUSE_DURATION` (5 days),
    /// never reducing an expiry already further in the future (so a lower-trust
    /// pauser cannot curtail a freeze). Not idempotent — each call extends the
    /// window. Aborts if the pauser role is unassigned.
    public fun pause(
        token_bridge_state: &mut State,
        cap: &PauserCap,
        clock: &Clock,
        ctx: &TxContext
    ) {
        let latest_only = state::assert_latest_only(token_bridge_state);

        let configured = state::pauser(token_bridge_state);
        assert!(option::is_some(&configured), E_PAUSER_NOT_CONFIGURED);

        let cap_id = object::id(cap);
        assert!(cap_id == option::destroy_some(configured), E_NOT_PAUSER);

        let new_expiry = clock::timestamp_ms(clock) + PAUSE_DURATION_MS;
        // Never reduce an expiry already further out (e.g. one set by `freeze`).
        if (new_expiry > state::pause_expiry(token_bridge_state)) {
            state::set_pause_expiry(&latest_only, token_bridge_state, new_expiry);
        };
        state::set_paused(&latest_only, token_bridge_state, true);

        sui::event::emit(Paused {
            cap: cap_id,
            sender: tx_context::sender(ctx),
            pause_expiry: state::pause_expiry(token_bridge_state)
        });
    }

    /// Freeze the token bridge for the maximum duration. Requires the active
    /// `FreezerCap`. Sets `paused` and `pauseExpiry` to the maximum timestamp.
    /// Idempotent. Aborts if the freezer role is unassigned.
    ///
    /// Named `freeze_bridge` because `freeze` is a reserved word in Move; it is
    /// the spec's `freeze`.
    public fun freeze_bridge(
        token_bridge_state: &mut State,
        cap: &FreezerCap,
        ctx: &TxContext
    ) {
        let latest_only = state::assert_latest_only(token_bridge_state);

        let configured = state::freezer(token_bridge_state);
        assert!(option::is_some(&configured), E_FREEZER_NOT_CONFIGURED);

        let cap_id = object::id(cap);
        assert!(cap_id == option::destroy_some(configured), E_NOT_FREEZER);

        state::set_pause_expiry(&latest_only, token_bridge_state, MAX_TIMESTAMP_MS);
        state::set_paused(&latest_only, token_bridge_state, true);

        sui::event::emit(Frozen {
            cap: cap_id,
            sender: tx_context::sender(ctx),
            pause_expiry: MAX_TIMESTAMP_MS
        });
    }

    /// Unpause the token bridge. Requires the active `UnpauserCap`. Clears
    /// `paused` and sets `pauseExpiry` to `now`. The privileged path to lift any
    /// pause (including a freeze) early. Aborts if the unpauser role is
    /// unassigned or the bridge is not currently paused.
    public fun unpause(
        token_bridge_state: &mut State,
        cap: &UnpauserCap,
        clock: &Clock,
        ctx: &TxContext
    ) {
        let latest_only = state::assert_latest_only(token_bridge_state);

        let configured = state::unpauser(token_bridge_state);
        assert!(option::is_some(&configured), E_UNPAUSER_NOT_CONFIGURED);

        let cap_id = object::id(cap);
        assert!(cap_id == option::destroy_some(configured), E_NOT_UNPAUSER);

        assert!(state::is_paused(token_bridge_state), E_NOT_PAUSED);

        state::set_pause_expiry(
            &latest_only,
            token_bridge_state,
            clock::timestamp_ms(clock)
        );
        state::set_paused(&latest_only, token_bridge_state, false);

        sui::event::emit(Unpaused { cap: cap_id, sender: tx_context::sender(ctx) });
    }

    /// Permissionlessly unpause the token bridge once its pause has expired.
    /// Clears `paused` and sets `pauseExpiry` to `now`. No capability required.
    /// Aborts if the bridge is not currently paused or `now < pauseExpiry`.
    public fun unpause_expired(
        token_bridge_state: &mut State,
        clock: &Clock,
        ctx: &TxContext
    ) {
        let latest_only = state::assert_latest_only(token_bridge_state);

        assert!(state::is_paused(token_bridge_state), E_NOT_PAUSED);

        let now = clock::timestamp_ms(clock);
        assert!(now >= state::pause_expiry(token_bridge_state), E_NOT_EXPIRED);

        state::set_pause_expiry(&latest_only, token_bridge_state, now);
        state::set_paused(&latest_only, token_bridge_state, false);

        sui::event::emit(UnpauseExpired { sender: tx_context::sender(ctx) });
    }

    #[test_only]
    public fun new_pauser_cap_test_only(ctx: &mut TxContext): PauserCap {
        new_pauser_cap(ctx)
    }

    #[test_only]
    public fun new_freezer_cap_test_only(ctx: &mut TxContext): FreezerCap {
        new_freezer_cap(ctx)
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
    public fun destroy_freezer_cap_test_only(cap: FreezerCap) {
        destroy_freezer_cap(cap);
    }

    #[test_only]
    public fun destroy_unpauser_cap_test_only(cap: UnpauserCap) {
        destroy_unpauser_cap(cap);
    }

    #[test_only]
    public fun pause_duration_ms(): u64 {
        PAUSE_DURATION_MS
    }

    #[test_only]
    public fun max_timestamp_ms(): u64 {
        MAX_TIMESTAMP_MS
    }
}
