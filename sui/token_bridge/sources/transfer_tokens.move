module token_bridge::transfer_tokens {
    use sui::sui::SUI;
    use sui::tx_context::{TxContext};
    use sui::coin::{Self, Coin};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::ExternalAddress;
    use wormhole::myu16::{U16};
    use wormhole::emitter::{Self, EmitterCapability};

    use token_bridge::bridge_state::{Self, BridgeState};
    use token_bridge::transfer_result::{Self, TransferResult};
    use token_bridge::transfer::{Self};
    use token_bridge::normalized_amount::{Self};
    use token_bridge::transfer_with_payload::{Self};

    // In our Sui token bridge implementation, we require that a Sui-native token
    // be registered (AKA attested) before it can be transferred
    const E_NATIVE_ASSET_NOT_REGISTERED: u64 = 0;

    public entry fun transfer_tokens<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<SUI>,
        recipient_chain: U16,
        recipient: ExternalAddress,
        relayer_fee: u64,
        nonce: u64,
        ctx: &mut TxContext
    ): u64 {
        let result = transfer_tokens_internal<CoinType>(
            bridge_state,
            coins,
            relayer_fee,
            ctx
        );
        let (token_chain, token_address, normalized_amount, normalized_relayer_fee)
            = transfer_result::destroy(result);
        let transfer = transfer::create(
            normalized_amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            normalized_relayer_fee,
        );
        bridge_state::publish_message(
            wormhole_state,
            bridge_state,
            nonce,
            transfer::encode(transfer),
            wormhole_fee_coins,
        )
    }

    public fun transfer_tokens_with_payload<CoinType>(
        emitter_cap: &EmitterCapability,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<SUI>,
        recipient_chain: U16,
        recipient: ExternalAddress,
        relayer_fee: u64,
        nonce: u64,
        payload: vector<u8>,
        ctx: &mut TxContext
    ): u64 {
        let result = transfer_tokens_internal<CoinType>(
            bridge_state,
            coins,
            relayer_fee,
            ctx
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
        //wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        _relayer_fee: u64,
        ctx: &mut TxContext
    ): TransferResult {
        let is_wrapped_asset = bridge_state::is_wrapped_asset<CoinType>(bridge_state);
        let is_registered_native_asset = bridge_state::is_registered_native_asset<CoinType>(bridge_state);

        // if CoinType is neither wrapped or registered native, then it is unregistered native
        assert!( is_wrapped_asset || is_registered_native_asset, E_NATIVE_ASSET_NOT_REGISTERED);

        //let this_chain = wormhole::state::get_chain_id(wormhole_state);
        let amount = coin::value<CoinType>(&coins);

        //let origin_info = bridge_state::create_origin_info(token_chain, token_address);
        let origin_info;
        if (is_registered_native_asset) {
            // token is native, so store token in bridge
            bridge_state::deposit<CoinType>(
                bridge_state,
                coins,
                ctx,
            );
            origin_info = bridge_state::get_registered_native_asset_origin_info<CoinType>(bridge_state);
        } else { // is_wrapped_asset
            // token is wrapped, so burn it
            bridge_state::burn<CoinType>(
                bridge_state,
                coins,
            );
            origin_info = bridge_state::get_wrapped_asset_origin_info<CoinType>(bridge_state);
        };
        // TODO - pending Mysten uniform token standard - figure out how to get normalization decimals for token and relayer fee amounts
        //        this is harder to do for native assets. For wrapped assets, we control the treasury cap, so we can get the decimals from there
        //        for now don't do normalization?
        let token_chain = bridge_state::get_token_chain_from_origin_info(&origin_info);
        let token_address = bridge_state::get_token_address_from_origin_info(&origin_info);
        let normalized_amount = normalized_amount::normalize(amount, 0);
        let normalized_relayer_fee = normalized_amount::normalize(_relayer_fee, 0);

        let transfer_result: TransferResult = transfer_result::create(
            token_chain,
            token_address,
            normalized_amount,
            normalized_relayer_fee,
        );
        transfer_result
    }
}
