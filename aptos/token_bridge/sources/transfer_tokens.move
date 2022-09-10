module token_bridge::transfer_tokens {
    use aptos_framework::aptos_coin::{AptosCoin};
    use aptos_framework::bcs::to_bytes;
    use aptos_framework::coin::{Self, Coin};

    use wormhole::u16::{Self, U16};
    use wormhole::u256;

    use token_bridge::bridge_state as state;
    use token_bridge::transfer;
    use token_bridge::transfer_result::{Self, TransferResult};
    use token_bridge::transfer_with_payload;
    use token_bridge::utils;

    public entry fun transfer_tokens_with_signer<CoinType>(
        sender: &signer,
        amount: u64,
        recipient_chain: u64,
        recipient: vector<u8>,
        relayer_fee: u64,
        wormhole_fee: u64,
        nonce: u64
        ): u64 {
        let coins = coin::withdraw<CoinType>(sender, amount);
        //let relayer_fee_coins = coin::withdraw<AptosCoin>(sender, relayer_fee);
        let wormhole_fee_coins = coin::withdraw<AptosCoin>(sender, wormhole_fee);
        transfer_tokens<CoinType>(coins, wormhole_fee_coins, u16::from_u64(recipient_chain), recipient, relayer_fee, nonce)
    }

    public fun transfer_tokens<CoinType>(
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<AptosCoin>,
        recipient_chain: U16,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u64
        ): u64 {
        let wormhole_fee = coin::value<AptosCoin>(&wormhole_fee_coins);
        let result = transfer_tokens_internal<CoinType>(coins, relayer_fee, wormhole_fee);
        let transfer = transfer::create(
            1,
            transfer_result::get_normalized_amount(&result),
            transfer_result::get_token_address(&result),
            transfer_result::get_token_chain(&result),
            recipient,
            recipient_chain,
            transfer_result::get_normalized_relayer_fee(&result),
        );
        state::publish_message(
            nonce,
            transfer::encode(transfer),
            wormhole_fee_coins,
        )
    }

    public fun transfer_tokens_with_payload_with_signer<CoinType>(
        sender: &signer,
        amount: u64,
        wormhole_fee: u64,
        recipient_chain: U16,
        recipient: vector<u8>,
        nonce: u64,
        payload: vector<u8>
        ): u64 {
        let coins = coin::withdraw<CoinType>(sender, amount);
        let wormhole_fee_coins = coin::withdraw<AptosCoin>(sender, wormhole_fee);
        transfer_tokens_with_payload(coins, wormhole_fee_coins, recipient_chain, recipient, nonce, payload)
    }

    public fun transfer_tokens_with_payload<CoinType>(
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<AptosCoin>,
        recipient_chain: U16,
        recipient: vector<u8>,
        nonce: u64,
        payload: vector<u8>
        ): u64 {
        let result = transfer_tokens_internal<CoinType>(coins, 0, 0); // TODO: the wormhole fee 0 is sus
        let transfer = transfer_with_payload::create(
            transfer_result::get_normalized_amount(&result),
            transfer_result::get_token_address(&result),
            transfer_result::get_token_chain(&result),
            recipient,
            recipient_chain,
            to_bytes<address>(&@token_bridge), //TODO - is token bridge the only one who will ever call log_transfer_with_payload?
            payload
        );
        let payload = transfer_with_payload::encode(transfer);
        state::publish_message(
            nonce,
            payload,
            wormhole_fee_coins,
        )
    }

    #[test_only]
    public fun transfer_tokens_test<CoinType>(
        coins: Coin<CoinType>,
        relayer_fee: u64,
        wormhole_fee: u64
    ): TransferResult {
        transfer_tokens_internal(coins, relayer_fee, wormhole_fee)
    }

    // transfer a native or wraped token from sender to token_bridge
    fun transfer_tokens_internal<CoinType>(
        coins: Coin<CoinType>,
        relayer_fee: u64,
        wormhole_fee: u64,
        ): TransferResult {

        // transfer coin to token_bridge
        if (!coin::is_account_registered<CoinType>(@token_bridge)){
            coin::register<CoinType>(&state::token_bridge_signer());
        };
        if (!coin::is_account_registered<AptosCoin>(@token_bridge)){
            coin::register<AptosCoin>(&state::token_bridge_signer());
        };
        // TODO: check that fee <= amount
        let amount = coin::value<CoinType>(&coins);
        coin::deposit<CoinType>(@token_bridge, coins);

        if (state::is_wrapped_asset<CoinType>()) {
            // now we burn the wrapped coins to remove them from circulation
            // TODO - wrapped::burn<CoinType>(amount);
            // wrapped::burn<CoinType>(amount);
            // problem here is that wrapped imports state, so state can't import wrapped...
        } else {
            // if we're seeing this native token for the first time, store its
            // type info
            if (!state::is_registered_native_asset<CoinType>()) {
                state::set_native_asset_type_info<CoinType>();
            };
        };

        let origin_info = state::origin_info<CoinType>();
        let token_chain = state::get_origin_info_token_chain(&origin_info);
        let token_address = state::get_origin_info_token_address(&origin_info);

        let decimals_token = coin::decimals<CoinType>();

        let normalized_amount = utils::normalize_amount(u256::from_u64(amount), decimals_token);
        let normalized_relayer_fee = utils::normalize_amount(u256::from_u64(relayer_fee), decimals_token);

        let transfer_result: TransferResult = transfer_result::create(
            token_chain,
            token_address,
            normalized_amount,
            normalized_relayer_fee,
            u256::from_u64(wormhole_fee)
        );
        transfer_result
    }


}

#[test_only]
module token_bridge::transfer_tokens_test {

}
