// Thoughts:
//
// - There's a desire when first coding to want to just mark everything `drop`.
// - Simply freezing everything as a starting model seems to work.
// - This implementation does not store VAA verify results, callers can decide this instead.

module Wormhole::Wormhole {
    use Sui::Transfer;
    use Sui::TxContext;
    use Wormhole::VAA;
    use Wormhole::GuardianSet::Guardian;
    use Wormhole::GuardianSet::GuardianSet;

    fun init(ctx: &mut TxContext) {
        Transfer::freeze_object(GuardianSet {
            id:        TxContext::new_id(ctx),
            index:     0,
            guardians: vector[
                x"0000000000000000000000000000000000000000",
            ],
        });
    }
}
