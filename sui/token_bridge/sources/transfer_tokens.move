module token_bridge::transfer_tokens {
    use sui::sui::SUI;
    use sui::coin::{Self, Coin, CoinMetadata};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::myu16::{Self as u16, U16};
    use wormhole::emitter::{Self, EmitterCapability};

    use token_bridge::bridge_state::{Self, BridgeState};
    use token_bridge::transfer_result::{Self, TransferResult};
    use token_bridge::transfer::{Self};
    use token_bridge::normalized_amount::{Self};
    use token_bridge::transfer_with_payload::{Self};

    const E_TOO_MUCH_RELAYER_FEE: u64 = 0;

    public entry fun transfer_tokens<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        coin_metadata: &CoinMetadata<CoinType>,
        wormhole_fee_coins: Coin<SUI>,
        recipient_chain: u64,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u64,
    ) {
        let result = transfer_tokens_internal<CoinType>(
            bridge_state,
            coins,
            coin_metadata,
            relayer_fee,
        );
        let (token_chain, token_address, normalized_amount, normalized_relayer_fee)
            = transfer_result::destroy(result);
        let transfer = transfer::create(
            normalized_amount,
            token_address,
            token_chain,
            external_address::from_bytes(recipient),
            u16::from_u64(recipient_chain),
            normalized_relayer_fee,
        );
        bridge_state::publish_message(
            wormhole_state,
            bridge_state,
            nonce,
            transfer::encode(transfer),
            wormhole_fee_coins,
        );
    }

    public fun transfer_tokens_with_payload<CoinType>(
        emitter_cap: &EmitterCapability,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        coin_metadata: &CoinMetadata<CoinType>,
        wormhole_fee_coins: Coin<SUI>,
        recipient_chain: U16,
        recipient: ExternalAddress,
        relayer_fee: u64,
        nonce: u64,
        payload: vector<u8>,
    ): u64 {
        let result = transfer_tokens_internal<CoinType>(
            bridge_state,
            coins,
            coin_metadata,
            relayer_fee,
        );
        let (token_chain, token_address, normalized_amount, _)
            = transfer_result::destroy(result);

        let transfer = transfer_with_payload::create(
            normalized_amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            emitter::get_external_address(emitter_cap),
            payload
        );
        let payload = transfer_with_payload::encode(transfer);
        bridge_state::publish_message(
            wormhole_state,
            bridge_state,
            nonce,
            payload,
            wormhole_fee_coins
        )
    }

    fun transfer_tokens_internal<CoinType>(
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        coin_metadata: &CoinMetadata<CoinType>,
        relayer_fee: u64,
    ): TransferResult {
        let amount = coin::value<CoinType>(&coins);
        assert!(relayer_fee <= amount, E_TOO_MUCH_RELAYER_FEE);

        if (bridge_state::is_wrapped_asset<CoinType>(bridge_state)) {
            // now we burn the wrapped coins to remove them from circulation
            bridge_state::burn<CoinType>(bridge_state, coins);
        } else {
            // deposit native assets. this call to deposit requires the native
            // asset to have been attested
            bridge_state::deposit<CoinType>(bridge_state, coins);
        };

        let origin_info = bridge_state::origin_info<CoinType>(bridge_state);
        let token_chain = bridge_state::get_token_chain_from_origin_info(&origin_info);
        let token_address = bridge_state::get_token_address_from_origin_info(&origin_info);

        let decimals = coin::get_decimals(coin_metadata);
        let normalized_amount = normalized_amount::normalize(amount, decimals);
        let normalized_relayer_fee = normalized_amount::normalize(relayer_fee, decimals);

        let transfer_result: TransferResult = transfer_result::create(
            token_chain,
            token_address,
            normalized_amount,
            normalized_relayer_fee,
        );
        transfer_result
    }
}
