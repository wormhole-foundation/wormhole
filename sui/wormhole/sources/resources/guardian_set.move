// SPDX-License-Identifier: Apache 2

/// This module implements a container that keeps track of a list of Guardian
/// public keys and which Guardian set index this list of Guardians represents.
/// Each guardian set is unique and there should be no two sets that have the
/// same Guardian set index (which requirement is handled in `wormhole::state`).
///
/// If the current Guardian set is not the latest one, its `expiration_time` is
/// configured, which defines how long the past Guardian set can be active.
module wormhole::guardian_set {
    use sui::clock::{Self, Clock};

    use wormhole::guardian::{Self, Guardian};


    /// Found duplicate public key.
    const E_DUPLICATE_GUARDIAN: u64 = 0;

    /// Container for the list of Guardian public keys, its index value and at
    /// what point in time the Guardian set is configured to expire.
    public struct GuardianSet has store {
        /// A.K.A. Guardian set index.
        index: u32,

        /// List of Guardians. This order should not change.
        guardians: vector<Guardian>,

        /// At what point in time the Guardian set is no longer active (in ms).
        expiration_timestamp_ms: u64,
    }

    /// Create new `GuardianSet`.
    public fun new(index: u32, guardians: vector<Guardian>): GuardianSet {
        // Ensure that there are no duplicate guardians.
        let (mut i, n) = (0, guardians.length());
        while (i < n - 1) {
            let left = guardian::pubkey(guardians.borrow(i));
            let mut j = i + 1;
            while (j < n) {
                let right = guardian::pubkey(guardians.borrow(j));
                assert!(left != right, E_DUPLICATE_GUARDIAN);
                j = j + 1;
            };
            i = i + 1;
        };

        GuardianSet { index, guardians, expiration_timestamp_ms: 0 }
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
    public fun expiration_timestamp_ms(self: &GuardianSet): u64 {
        self.expiration_timestamp_ms
    }

    /// Retrieve whether this Guardian set is still active by checking the
    /// current time.
    public fun is_active(self: &GuardianSet, clock: &Clock): bool {
        (
            self.expiration_timestamp_ms == 0 ||
            self.expiration_timestamp_ms > clock::timestamp_ms(clock)
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
    ///
    /// NOTE: `time_to_live` is in units of seconds while `Clock` uses
    /// milliseconds.
    public(package) fun set_expiration(
        self: &mut GuardianSet,
        seconds_to_live: u32,
        the_clock: &Clock
    ) {
        let ttl_ms = (seconds_to_live as u64) * 1000;
        self.expiration_timestamp_ms = clock::timestamp_ms(the_clock) + ttl_ms;
    }

    #[test_only]
    public fun destroy(set: GuardianSet) {
        let GuardianSet {
            index: _,
            mut guardians,
            expiration_timestamp_ms: _
        } = set;
        while (!guardians.is_empty()) {
            guardians.pop_back().destroy();
        };

        guardians.destroy_empty();
    }
}

#[test_only]
module wormhole::guardian_set_tests {
    use wormhole::guardian::{Self};
    use wormhole::guardian_set::{Self};

    #[test]
    fun test_new() {
        let mut guardians = vector::empty();

        let mut pubkeys = vector[
            x"8888888888888888888888888888888888888888",
            x"9999999999999999999999999999999999999999",
            x"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            x"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
            x"cccccccccccccccccccccccccccccccccccccccc",
            x"dddddddddddddddddddddddddddddddddddddddd",
            x"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
            x"ffffffffffffffffffffffffffffffffffffffff"
        ];
        while (!pubkeys.is_empty()) {
            guardians.push_back(
                guardian::new(pubkeys.pop_back())
            );
        };

        let set = guardian_set::new(69, guardians);

        // Clean up.
        set.destroy();
    }

    #[test]
    #[expected_failure(abort_code = guardian_set::E_DUPLICATE_GUARDIAN)]
    fun test_cannot_new_duplicate_guardian() {
        let mut guardians = vector::empty();

        let mut pubkeys = vector[
            x"8888888888888888888888888888888888888888",
            x"9999999999999999999999999999999999999999",
            x"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            x"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
            x"cccccccccccccccccccccccccccccccccccccccc",
            x"dddddddddddddddddddddddddddddddddddddddd",
            x"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
            x"ffffffffffffffffffffffffffffffffffffffff",
            x"cccccccccccccccccccccccccccccccccccccccc",
        ];
        while (!pubkeys.is_empty()) {
            guardians.push_back(
                guardian::new(pubkeys.pop_back())
            );
        };

        let set = guardian_set::new(69, guardians);

        // Clean up.
        set.destroy();

        abort 42
    }
}
