// SPDX-License-Identifier: Apache 2

/// This module implements entry methods that expose methods from modules found
/// in the Token Bridge contract.
module token_bridge::token_bridge {
    use sui::balance::{Self};
    use sui::coin::{Self, Coin};
    use sui::sui::{SUI};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};
    use wormhole::emitter::{EmitterCap};
    use wormhole::external_address::{Self};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{State};

    entry fun transfer_tokens<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        bridged: Coin<CoinType>,
        wormhole_fee: Coin<SUI>,
        recipient_chain: u16,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u32,
    ) {
        use token_bridge::transfer_tokens::{transfer_tokens};

        transfer_tokens(
            token_bridge_state,
            worm_state,
            coin::into_balance(bridged),
            coin::into_balance(wormhole_fee),
            recipient_chain,
            external_address::from_bytes(recipient),
            relayer_fee,
            nonce
        );
    }

    entry fun transfer_tokens_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        worm_state: &mut WormholeState,
        bridged: Coin<CoinType>,
        wormhole_fee: Coin<SUI>,
        recipient_chain: u16,
        recipient: vector<u8>,
        nonce: u32,
        payload: vector<u8>,
    ) {
        use token_bridge::transfer_tokens_with_payload::{
            transfer_tokens_with_payload
        };

        transfer_tokens_with_payload(
            token_bridge_state,
            emitter_cap,
            worm_state,
            coin::into_balance(bridged),
            coin::into_balance(wormhole_fee),
            recipient_chain,
            external_address::from_bytes(recipient),
            nonce,
            payload
        );
    }

    entry fun complete_transfer<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        ctx: &mut TxContext
    ) {
        use token_bridge::complete_transfer::{complete_transfer};

        // There may be some value to `payout` if the sender of the transaction
        // is not the same as the intended recipient and there was an encoded
        // fee.
        let payout = complete_transfer<CoinType>(
            token_bridge_state,
            worm_state,
            vaa_buf,
            ctx
        );

        if (balance::value(&payout) == 0) {
            balance::destroy_zero(payout);
        } else {
            transfer::transfer(
                coin::from_balance(payout, ctx),
                tx_context::sender(ctx)
            );
        };
    }
}
