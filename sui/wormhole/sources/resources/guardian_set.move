module wormhole::guardian_set {
    use std::vector::{Self};
    use sui::tx_context::{Self, TxContext};

    use wormhole::cursor::{Self};
    use wormhole::guardian::{Self, Guardian};
    use wormhole::guardian_signature::{Self, GuardianSignature};

    // Needs `set_expiration`
    friend wormhole::state;

    const E_NO_QUORUM: u64 = 0;
    const E_INVALID_SIGNATURE: u64 = 1;
    const E_GUARDIAN_SET_EXPIRED: u64 = 2;
    const E_NON_INCREASING_SIGNERS: u64 = 3;

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

    public fun index_as_u64(self: &GuardianSet): u64 {
        (self.index as u64)
    }

    public fun guardians(self: &GuardianSet): &vector<Guardian> {
        &self.guardians
    }

    public fun guardian_at(self: &GuardianSet, index: u64): &Guardian {
        vector::borrow(&self.guardians, index)
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

    public fun verify_signatures(
        self: &GuardianSet,
        signatures: vector<GuardianSignature>,
        message: vector<u8>,
        ctx: &TxContext
    ) {
        // Guardian set must be active (not expired).
        assert!(is_active(self, ctx), E_GUARDIAN_SET_EXPIRED);

        // Number of signatures must be at least quorum.
        assert!(vector::length(&signatures) >= quorum(self), E_NO_QUORUM);

        // Drain `Cursor` by checking each signature.
        let cur = cursor::new(signatures);
        let (i, last_guardian_index) = (0, 0);
        while (!cursor::is_empty(&cur)) {
            let signature = cursor::poke(&mut cur);
            let guardian_index = guardian_signature::index_as_u64(&signature);

            // Ensure that the provided signatures are strictly increasing.
            // This check makes sure that no duplicate signers occur. The
            // increasing order is guaranteed by the guardians, or can always be
            // reordered by the client.
            assert!(
                i == 0 || guardian_index > last_guardian_index,
                E_NON_INCREASING_SIGNERS
            );

            // If the guardian pubkey cannot be recovered using the signature
            // and message hash, revert.
            assert!(
                guardian::verify(
                    guardian_at(self, guardian_index),
                    signature,
                    message
                ),
                E_INVALID_SIGNATURE
            );

            // Continue.
            i = i + 1;
            last_guardian_index = guardian_index;
        };

        // Done.
        cursor::destroy_empty(cur);
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
