// SPDX-License-Identifier: Apache 2

/// This module implements a container that keeps track of a list of Guardian
/// public keys and which Guardian set index this list of Guardians represents.
/// Each guardian set is unique and there should be no two sets that have the
/// same Guardian set index (which requirement is handled in `wormhole::state`).
///
/// If the current Guardian set is not the latest one, its `expiration_time` is
/// configured, which defines how long the past Guardian set can be active.
module wormhole::guardian_set {
    use std::vector::{Self};
    use sui::tx_context::{Self, TxContext};

    use wormhole::guardian::{Guardian};

    // Needs `set_expiration`.
    friend wormhole::state;

    /// Container for the list of Guardian public keys, its index value and at
    /// what point in time the Guardian set is configured to expire.
    struct GuardianSet has store {
        /// A.K.A. Guardian set index.
        index: u32,

        /// List of Guardians. This order should not change.
        guardians: vector<Guardian>,

        /// At what point in time the Guardian set is no longer active.
        expiration_time: u32,
    }

    /// Create new `GuardianSet`.
    public fun new(index: u32, guardians: vector<Guardian>): GuardianSet {
       GuardianSet { index, guardians, expiration_time: 0 }
    }

    /// Retrieve the Guardian set index.
    public fun index(self: &GuardianSet): u32 {
        self.index
    }

    /// Retrieve the Guardian set index as `u64` (for convenience when used to
    /// compare to indices for iterations, which are natively `u64`).
    public fun index_as_u64(self: &GuardianSet): u64 {
        (self.index as u64)
    }

    /// Retrieve list of Guardians.
    public fun guardians(self: &GuardianSet): &vector<Guardian> {
        &self.guardians
    }

    /// Retrieve specific Guardian by index (in the array representing the set).
    public fun guardian_at(self: &GuardianSet, index: u64): &Guardian {
        vector::borrow(&self.guardians, index)
    }

    /// Retrieve when the Guardian set is no longer active.
    public fun expiration_time(self: &GuardianSet): u32 {
        self.expiration_time
    }

    /// Retrieve whether this Guardian set is still active by checking the
    /// current time.
    /// TODO: change `ctx` to `clock` reference.
    public fun is_active(self: &GuardianSet, ctx: &TxContext): bool {
        (
            self.expiration_time == 0 ||
            self.expiration_time > (tx_context::epoch(ctx) as u32)
        )
    }

    /// Retrieve how many guardians exist in the Guardian set.
    public fun num_guardians(self: &GuardianSet): u64 {
        vector::length(&self.guardians)
    }

    /// Returns the minimum number of signatures required for a VAA to be valid.
    public fun quorum(self: &GuardianSet): u64 {
        (num_guardians(self) * 2) / 3 + 1
    }

    /// Configure this Guardian set to expire from some amount of time based on
    /// what time it is right now.
    /// TODO: change `ctx` to `clock` reference.
    public(friend) fun set_expiration(
        self: &mut GuardianSet,
        epochs_to_live: u32,
        ctx: &TxContext
    ) {
        self.expiration_time = (tx_context::epoch(ctx) as u32) + epochs_to_live;
    }

    #[test_only]
    public fun destroy(set: GuardianSet) {
        use wormhole::guardian::{Self};

        let GuardianSet { index: _, guardians, expiration_time: _ } = set;
        while (!vector::is_empty(&guardians)) {
            guardian::destroy(vector::pop_back(&mut guardians));
        };

        vector::destroy_empty(guardians);
    }
}
