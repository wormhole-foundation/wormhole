module wormhole::wormhole {
    use sui::coin::{Coin};
    use sui::sui::{SUI};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use wormhole::emitter::{EmitterCapability};
    use wormhole::state::{State};

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

    // -----------------------------------------------------------------------------
    // get_new_emitter
    //
    // Honestly, unsure if this should survive once we get into code review but it
    // sure makes writing my test script work quite well
    //
    // This creates a new emitter object and stores it away into the senders context.
    //
    // You can then use this to call publish_message_free and generate a vaa

    public entry fun new_emitter(
        wormhole_state: &mut State,
        ctx: &mut TxContext
    ) {
        use wormhole::state::{new_emitter};

        transfer::transfer(
            new_emitter(wormhole_state, ctx),
            tx_context::sender(ctx)
        );
    }
}
