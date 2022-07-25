// Thoughts:
//
// - There's a desire when first coding to want to just mark everything `drop`.
// - Simply freezing everything as a starting model seems to work.
// - This implementation does not store VAA verify results, callers can decide this instead.

module Wormhole::Wormhole {
    use sui::tx_context::{Self, TxContext};
    use Wormhole::Governance;

    fun init(ctx: &mut TxContext) {
        Governance::init_guardian_set(ctx);
    }
}

