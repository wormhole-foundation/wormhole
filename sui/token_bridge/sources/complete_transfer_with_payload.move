module token_bridge::complete_transfer_with_payload {
    use sui::tx_context::{TxContext};
    use sui::coin::{Self, Coin, CoinMetadata};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self};

    use token_bridge::bridge_state::{Self, BridgeState, VerifiedCoinType};
    use token_bridge::vaa::{Self};
    //use token_bridge::transfer::{Self, Transfer};
    use token_bridge::transfer_with_payload::{Self, TransferWithPayload};
    use token_bridge::normalized_amount::{denormalize};

    const E_INVALID_TARGET: u64 = 0;

    public fun submit_vaa<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        vaa: vector<u8>,
        ctx: &mut TxContext
    ): (Coin<CoinType>, TransferWithPayload) {
        let vaa = vaa::parse_verify_and_replay_protect(
            wormhole_state,
            bridge_state,
            vaa,
            ctx
        );

        let transfer = transfer_with_payload::parse(wormhole::myvaa::destroy(vaa));

        let token_chain = transfer_with_payload::get_token_chain(&transfer);
        let token_address = transfer_with_payload::get_token_address(&transfer);
        let verified_coin_witness = bridge_state::verify_coin_type<CoinType>(
            bridge_state,
            token_chain,
            token_address
        );

        complete_transfer_with_payload<CoinType>(
            verified_coin_witness,
            transfer,
            wormhole_state,
            bridge_state,
            coin_meta,
            ctx
        )
    }

    // complete transfer with arbitrary Transfer request and without the VAA
    // for native tokens
    #[test_only]
    public fun test_complete_transfer_with_payload<CoinType>(
        transfer: TransferWithPayload,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        ctx: &mut TxContext
    ): (Coin<CoinType>, TransferWithPayload) {
        let token_chain = transfer_with_payload::get_token_chain(&transfer);
        let token_address = transfer_with_payload::get_token_address(&transfer);
        let verified_coin_witness = bridge_state::verify_coin_type<CoinType>(
            bridge_state,
            token_chain,
            token_address
        );
        complete_transfer_with_payload<CoinType>(
            verified_coin_witness,
            transfer,
            wormhole_state,
            bridge_state,
            coin_meta,
            ctx
        )
    }

    fun complete_transfer_with_payload<CoinType>(
        verified_coin_witness: VerifiedCoinType<CoinType>,
        transfer: TransferWithPayload,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        ctx: &mut TxContext
    ): (Coin<CoinType>, TransferWithPayload) {
        let to_chain = transfer_with_payload::get_to_chain(&transfer);
        let this_chain = wormhole::state::get_chain_id(wormhole_state);
        assert!(to_chain == this_chain, E_INVALID_TARGET);

        let _recipient = external_address::to_address(&transfer_with_payload::get_to(&transfer));
        // TODO - pass emitter cap into this function and assert that recipient==address(emitter_cap?)
        // https://github.com/wormhole-foundation/wormhole/blob/main/aptos/token_bridge/sources/complete_transfer_with_payload.move#L37
        let decimals = coin::get_decimals(coin_meta);
        let amount = denormalize(transfer_with_payload::get_amount(&transfer), decimals);

        let recipient_coins;
        if (bridge_state::is_wrapped_asset<CoinType>(bridge_state)) {
            recipient_coins = bridge_state::mint<CoinType>(
                verified_coin_witness,
                bridge_state,
                amount,
                ctx
            );
        } else {
            recipient_coins = bridge_state::withdraw<CoinType>(
                verified_coin_witness,
                bridge_state,
                amount,
                ctx
            );
        };
        (recipient_coins, transfer)
    }
}
