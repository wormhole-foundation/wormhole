// SPDX-License-Identifier: Apache 2

/// This module implements entry methods that expose methods from modules found
/// in the Token Bridge contract.
module token_bridge::token_bridge {
    use sui::clock::{Clock};
    use sui::coin::{Coin, CoinMetadata};
    use sui::sui::{SUI};
    use sui::tx_context::{TxContext};
    use wormhole::bytes32::{Self};
    use wormhole::emitter::{EmitterCap};
    use wormhole::external_address::{Self};
    use wormhole::state::{State as WormholeState};

    use token_bridge::coin_utils::{Self};
    use token_bridge::state::{State};

    entry fun attest_token<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        wormhole_fee: Coin<SUI>,
        coin_metadata: &CoinMetadata<CoinType>,
        nonce: u32,
        the_clock: &Clock
    ) {
        use token_bridge::attest_token::{attest_token};

        attest_token<CoinType>(
            token_bridge_state,
            worm_state,
            wormhole_fee,
            coin_metadata,
            nonce,
            the_clock
        );
    }

    entry fun transfer_tokens<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        bridged_in: Coin<CoinType>,
        wormhole_fee: Coin<SUI>,
        recipient_chain: u16,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u32,
        the_clock: &Clock,
        ctx: &TxContext
    ) {
        use token_bridge::transfer_tokens::{transfer_tokens};

        let (
            _,
            dust
        ) =
            transfer_tokens(
                token_bridge_state,
                worm_state,
                bridged_in,
                wormhole_fee,
                recipient_chain,
                external_address::new(bytes32::new(recipient)),
                relayer_fee,
                nonce,
                the_clock
            );

        coin_utils::return_nonzero(dust, ctx);
    }

    entry fun transfer_tokens_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        worm_state: &mut WormholeState,
        bridged_in: Coin<CoinType>,
        wormhole_fee: Coin<SUI>,
        redeemer_chain: u16,
        redeemer: vector<u8>,
        payload: vector<u8>,
        nonce: u32,
        the_clock: &Clock,
        ctx: &TxContext
    ) {
        use token_bridge::transfer_tokens_with_payload::{
            transfer_tokens_with_payload
        };

        let (
            _,
            dust
        ) =
            transfer_tokens_with_payload(
                token_bridge_state,
                emitter_cap,
                worm_state,
                bridged_in,
                wormhole_fee,
                redeemer_chain,
                external_address::new(bytes32::new(redeemer)),
                payload,
                nonce,
                the_clock
            );

        coin_utils::return_nonzero(dust, ctx);
    }

    entry fun complete_transfer<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        the_clock: &Clock,
        ctx: &mut TxContext
    ) {
        use token_bridge::complete_transfer::{complete_transfer};

        // There may be some value to `payout` if the sender of the transaction
        // is not the same as the intended recipient and there was an encoded
        // fee.
        coin_utils::return_nonzero(
            complete_transfer<CoinType>(
                token_bridge_state,
                worm_state,
                vaa_buf,
                the_clock,
                ctx
            ),
            ctx
        )
    }
}
