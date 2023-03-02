module wormhole::wormhole {
    use sui::coin::{Coin};
    use sui::sui::{SUI};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use wormhole::emitter::{EmitterCap};
    use wormhole::state::{State};

    /// `publish_message` exposes `publish_message::publish_message` as an entry
    /// method to publish Wormhole messages with an emitter cap owned a
    /// wallet.
    ///
    /// See `publish_message` module for more details.
    public entry fun publish_message(
        wormhole_state: &mut State,
        emitter_cap: &mut EmitterCap,
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

    // `new_emitter` exposes `state::new_emitter` as an entry method to create
    // a new emitter cap and transfer it to the transaction sender.
    //
    // See `state` module for more details.
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

    /// Placeholder for upgrade contract logic.
    public entry fun upgrade_contract(
        _wormhole_state: &mut State,
        _vaa_buf: vector<u8>,
        _ctx: &TxContext
    ) {
        abort 0
    }

    /// `update_guardian_set` exposes `update_guardian_set::update_guardian_set`
    /// as an entry method to perform Guardian governance to update the existing
    /// guardian set to a new one, specifying the latest guardian set index and
    /// associated guardian public keys.
    ///
    /// See `update_guardian_set` module for more details.
    public entry fun update_guardian_set(
        wormhole_state: &mut State,
        vaa_buf: vector<u8>,
        ctx: &TxContext
    ) {
        use wormhole::update_guardian_set::{update_guardian_set};

        update_guardian_set(wormhole_state, vaa_buf, ctx);
    }

    /// `set_fee` exposes `set_fee::set_fee` as an entry method to perform
    /// Guardian governance to update the existing Wormhole message fee.
    ///
    /// See `set_fee` module for more details.
    public entry fun set_fee(
        wormhole_state: &mut State,
        vaa_buf: vector<u8>,
        ctx: &TxContext
    ) {
        use wormhole::set_fee::{set_fee};

        set_fee(wormhole_state, vaa_buf, ctx);
    }

    /// `transfer_fee` exposes `transfer_fee::transfer_fee` as an entry method
    /// to perform Guardian governance to transfer fees to a specified
    /// recipient.
    ///
    /// See `transfer_fee` module for more details.
    public entry fun transfer_fee(
        wormhole_state: &mut State,
        vaa_buf: vector<u8>,
        ctx: &mut TxContext
    ) {
        use wormhole::transfer_fee::{transfer_fee};

        transfer_fee(wormhole_state, vaa_buf, ctx);
    }
}
