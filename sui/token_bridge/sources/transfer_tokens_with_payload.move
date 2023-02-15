module token_bridge::transfer_tokens_with_payload {
    use sui::sui::{SUI};
    use sui::coin::{Coin};
    use wormhole::emitter::{Self, EmitterCapability};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{Self, State};
    use token_bridge::transfer_result::{Self};
    use token_bridge::transfer_tokens::{handle_transfer_tokens};
    use token_bridge::transfer_with_payload::{Self};

    public fun transfer_tokens_with_payload<CoinType>(
        emitter_cap: &EmitterCapability,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut State,
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<SUI>,
        recipient_chain: u16,
        recipient: ExternalAddress,
        nonce: u32,
        payload: vector<u8>,
    ): u64 {
        let result = handle_transfer_tokens<CoinType>(
            bridge_state,
            coins,
            0,
        );
        let (token_chain, token_address, normalized_amount, _)
            = transfer_result::destroy(result);

        let transfer = transfer_with_payload::new(
            normalized_amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            emitter::get_external_address(emitter_cap),
            payload
        );

        state::publish_wormhole_message(
            bridge_state,
            wormhole_state,
            nonce,
            transfer_with_payload::serialize(transfer),
            wormhole_fee_coins
        )
    }
}

// TODO: write specific tests for `transfer_tokens_with_payload`
