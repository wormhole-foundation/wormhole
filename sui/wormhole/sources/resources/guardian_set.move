module wormhole::guardian_set {
    use std::vector::{Self};
    use sui::tx_context::{Self, TxContext};
    use wormhole::guardian::{Guardian};

    // Needs `set_expiration`
    friend wormhole::state;

    struct GuardianSet has store {
        index: u32,
        guardians: vector<Guardian>,
        expiration_time: u32,
    }

    public fun new(index: u32, guardians: vector<Guardian>): GuardianSet {
       GuardianSet { index, guardians, expiration_time: 0 }
    }

    public fun index(self: &GuardianSet): u32 {
        self.index
    }

    public fun guardians(self: &GuardianSet): &vector<Guardian> {
        &self.guardians
    }

    public fun expiration_time(self: &GuardianSet): u32 {
        self.expiration_time
    }

    public fun is_active(self: &GuardianSet, ctx: &TxContext): bool {
        (
            self.expiration_time == 0 ||
            self.expiration_time > (tx_context::epoch(ctx) as u32)
        )
    }

    public fun num_guardians(self: &GuardianSet): u64 {
        vector::length(&self.guardians)
    }

    /// Returns the minimum number of signatures required for a VAA to be valid.
    public fun quorum(self: &GuardianSet): u64 {
        (num_guardians(self) * 2) / 3 + 1
    }

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
