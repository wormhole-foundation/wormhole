// SPDX-License-Identifier: Apache 2

/// This module implements entry methods that expose methods from modules found
/// in the Wormhole contract.
module wormhole::wormhole {
    use sui::clock::{Clock};
    use sui::tx_context::{TxContext};

    use wormhole::state::{State};

    /// `update_guardian_set` exposes `update_guardian_set::update_guardian_set`
    /// as an entry method to perform Guardian governance to update the existing
    /// guardian set to a new one, specifying the latest guardian set index and
    /// associated guardian public keys.
    ///
    /// See `update_guardian_set` module for more details.
    entry fun update_guardian_set(
        wormhole_state: &mut State,
        vaa_buf: vector<u8>,
        the_clock: &Clock
    ) {
        use wormhole::update_guardian_set::{update_guardian_set};

        update_guardian_set(wormhole_state, vaa_buf, the_clock);
    }

    /// `set_fee` exposes `set_fee::set_fee` as an entry method to perform
    /// Guardian governance to update the existing Wormhole message fee.
    ///
    /// See `set_fee` module for more details.
    entry fun set_fee(
        wormhole_state: &mut State,
        vaa_buf: vector<u8>,
        the_clock: &Clock
    ) {
        use wormhole::set_fee::{set_fee};

        set_fee(wormhole_state, vaa_buf, the_clock);
    }

    /// `transfer_fee` exposes `transfer_fee::transfer_fee` as an entry method
    /// to perform Guardian governance to transfer fees to a specified
    /// recipient.
    ///
    /// See `transfer_fee` module for more details.
    entry fun transfer_fee(
        wormhole_state: &mut State,
        vaa_buf: vector<u8>,
        the_clock: &Clock,
        ctx: &mut TxContext
    ) {
        use wormhole::transfer_fee::{transfer_fee};

        transfer_fee(wormhole_state, vaa_buf, the_clock, ctx);
    }
}
