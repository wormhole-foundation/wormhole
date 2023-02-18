module wormhole::wormhole {
    use sui::coin::{Coin};
    use sui::sui::{SUI};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use wormhole::emitter::{EmitterCapability};
    use wormhole::setup::{Self};
    use wormhole::state::{State};

    /// Called automatically when module is first published. Transfers
    /// `DeployerCapability` to sender.
    ///
    /// Only `setup::init_and_share_state` requires `DeployerCapability`.
    fun init(ctx: &mut TxContext) {
        transfer::transfer(
            setup::new_capability(ctx),
            tx_context::sender(ctx)
        );
    }

    #[test_only]
    public fun init_test_only(ctx: &mut TxContext) {
        init(ctx)
    }

    /// `publish_message` exposes `wormhole::publish_message` as an entry method
    /// to publish Wormhole messages via RPC transaction.
    ///
    /// See `wormhole::publish_message` for more details.
    public entry fun publish_message(
        wormhole_state: &mut State,
        emitter_cap: &mut EmitterCapability,
        nonce: u32,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
    ) {
        use wormhole::publish_message::{publish_message};

        publish_message(
            wormhole_state,
            emitter_cap,
            nonce,
            payload,
            message_fee
        );
    }
}
