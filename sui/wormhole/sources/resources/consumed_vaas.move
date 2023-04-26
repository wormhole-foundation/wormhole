module wormhole::consumed_vaas {
    use sui::tx_context::{TxContext};

    use wormhole::bytes32::{Bytes32};
    use wormhole::set::{Self, Set};

    /// Container storing VAA hashes (digests). This will be checked against in
    /// `parse_verify_and_consume` so a particular VAA cannot be replayed. It
    /// is up to the integrator to have this container live in his contract
    /// in order to take advantage of this no-replay protection. Or an
    /// integrator can implement his own method to prevent replay.
    struct ConsumedVAAs has store {
        hashes: Set<Bytes32>
    }

    public fun new(ctx: &mut TxContext): ConsumedVAAs {
        ConsumedVAAs { hashes: set::new(ctx) }
    }

    public fun consume(self: &mut ConsumedVAAs, digest: Bytes32) {
        set::add(&mut self.hashes, digest);
    }

    #[test_only]
    public fun destroy(consumed: ConsumedVAAs) {
        let ConsumedVAAs { hashes } = consumed;
        set::destroy(hashes);
    }
}
