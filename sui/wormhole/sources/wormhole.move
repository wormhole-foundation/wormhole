// SPDX-License-Identifier: Apache 2

/// This module implements entry methods that expose methods from modules found
/// in the Wormhole contract.
module wormhole::wormhole {
    use sui::coin::{Coin};
    use sui::sui::{SUI};
    use sui::tx_context::{TxContext};

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
