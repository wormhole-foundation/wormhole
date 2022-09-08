module token_bridge::attest_token {
    use aptos_framework::aptos_coin::{AptosCoin};
    use aptos_framework::coin::{Self, Coin};
    use std::string;

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::bridge_state as state;
    use token_bridge::token_hash;

    const E_COIN_IS_NOT_INITIALIZED: u64 = 0;

    public fun attest_token_with_signer<CoinType>(user: &signer): u64 {
        let message_fee = wormhole::state::get_message_fee();
        let fee_coins = coin::withdraw<AptosCoin>(user, message_fee);
        attest_token<CoinType>(fee_coins)
    }

    public fun attest_token<CoinType>(fee_coins: Coin<AptosCoin>): u64 {
        // you've can't attest an uninitialized token
        // TODO - throw error if attempt to attest wrapped token?
        assert!(coin::is_coin_initialized<CoinType>(), E_COIN_IS_NOT_INITIALIZED);
        let payload_id = 0;
        let token_address = token_hash::derive<CoinType>();
        if (!state::is_registered_native_asset<CoinType>() && !state::is_wrapped_asset<CoinType>()) {
            // if native asset is not registered, register it in the reverse look-up map
            state::set_native_asset_type_info<CoinType>();
        };
        let token_chain = wormhole::state::get_chain_id();
        let decimals = coin::decimals<CoinType>();
        let symbol = *string::bytes(&coin::symbol<CoinType>());
        // TODO - left pad to be 32 bytes?
        let name = *string::bytes(&coin::name<CoinType>());
        let asset_meta: AssetMeta = asset_meta::create(
            payload_id,
            token_hash::get_bytes(&token_address),
            token_chain,
            decimals,
            symbol,
            name
        );
        let payload:vector<u8> = asset_meta::encode(asset_meta);
        let nonce = 0;
        state::publish_message(
            nonce,
            payload,
            fee_coins
        )
    }
}

module token_bridge::attest_token_test {

}
