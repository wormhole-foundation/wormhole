module token_bridge::attest_token {
    use aptos_framework::aptos_coin::{AptosCoin};
    use aptos_framework::coin::{Self, Coin};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::state;
    use token_bridge::token_hash;
    use token_bridge::string32;

    const E_COIN_IS_NOT_INITIALIZED: u64 = 0;
    /// Wrapped assets can't be attested
    const E_WRAPPED_ASSET: u64 = 1;

    public entry fun attest_token_entry<CoinType>(user: &signer) {
        let message_fee = wormhole::state::get_message_fee();
        let fee_coins = coin::withdraw<AptosCoin>(user, message_fee);
        attest_token<CoinType>(fee_coins);
    }

    public fun attest_token<CoinType>(fee_coins: Coin<AptosCoin>): u64 {
        let asset_meta: AssetMeta = attest_token_internal<CoinType>();
        let payload: vector<u8> = asset_meta::encode(asset_meta);
        let nonce = 0;
        state::publish_message(
            nonce,
            payload,
            fee_coins
        )
    }

    #[test_only]
    public fun attest_token_test<CoinType>(): AssetMeta {
        attest_token_internal<CoinType>()
    }

    fun attest_token_internal<CoinType>(): AssetMeta {
        // wrapped assets and uninitialised type can't be attested.
        assert!(!state::is_wrapped_asset<CoinType>(), E_WRAPPED_ASSET);
        assert!(coin::is_coin_initialized<CoinType>(), E_COIN_IS_NOT_INITIALIZED); // not tested

        let token_address = token_hash::derive<CoinType>();
        if (!state::is_registered_native_asset<CoinType>()) {
            // if native asset is not registered, register it in the reverse look-up map
            state::set_native_asset_type_info<CoinType>();
        };
        let token_chain = wormhole::state::get_chain_id();
        let decimals = coin::decimals<CoinType>();
        let symbol = string32::from_string(&coin::symbol<CoinType>());
        let name = string32::from_string(&coin::name<CoinType>());
        asset_meta::create(
            token_hash::get_external_address(&token_address),
            token_chain,
            decimals,
            symbol,
            name
        )
    }
}

#[test_only]
module token_bridge::attest_token_test {
    use aptos_framework::coin;
    use aptos_framework::string::utf8;
    use aptos_framework::type_info::type_of;

    use token_bridge::token_bridge::{Self as bridge};
    use token_bridge::state;
    use token_bridge::attest_token;
    use token_bridge::token_hash;
    use token_bridge::asset_meta;
    use token_bridge::string32;
    use token_bridge::wrapped_test;

    struct MyCoin has key {}

    fun setup(
        token_bridge: &signer,
        deployer: &signer,
    ) {
        // we initialise the bridge with zero fees to avoid having to mint fee
        // tokens in these tests. The wormolhe fee handling is already tested
        // in wormhole.move, so it's unnecessary here.
        wormhole::wormhole_test::setup(0);
        bridge::init_test(deployer);

        init_my_token(token_bridge);
    }

    fun init_my_token(admin: &signer) {
        let name = utf8(b"Some test coin");
        let symbol = utf8(b"TEST");
        let decimals = 10;
        let monitor_supply = true;
        let (burn_cap, freeze_cap, mint_cap) = coin::initialize<MyCoin>(admin, name, symbol, decimals, monitor_supply);
        coin::destroy_burn_cap(burn_cap);
        coin::destroy_freeze_cap(freeze_cap);
        coin::destroy_mint_cap(mint_cap);
    }

    #[test(token_bridge=@token_bridge, deployer=@deployer)]
    fun test_attest_token(token_bridge: &signer, deployer: &signer) {
        use std::string;

        setup(token_bridge, deployer);
        let asset_meta = attest_token::attest_token_test<MyCoin>();

        let token_address = asset_meta::get_token_address(&asset_meta);
        let token_chain = asset_meta::get_token_chain(&asset_meta);
        let decimals = asset_meta::get_decimals(&asset_meta);
        let symbol = string32::to_string(&asset_meta::get_symbol(&asset_meta));
        let name = string32::to_string(&asset_meta::get_name(&asset_meta));

        assert!(token_address == token_hash::get_external_address(&token_hash::derive<MyCoin>()), 0);
        assert!(token_chain == wormhole::u16::from_u64(22), 0);
        assert!(decimals == 10, 0);
        assert!(name == string::utf8(b"Some test coin"), 0);
        assert!(symbol == string::utf8(b"TEST"), 0);
    }

    #[test(token_bridge=@token_bridge, deployer=@deployer)]
    #[expected_failure(abort_code = 1, location=token_bridge::attest_token)]
    fun test_attest_wrapped_token(token_bridge: &signer, deployer: &signer) {
        setup(token_bridge, deployer);
        wrapped_test::init_wrapped_token();
        // this should fail because T is a wrapped asset
        let _asset_meta = attest_token::attest_token_test<wrapped_coin::coin::T>();
    }

    #[test(token_bridge=@token_bridge, deployer=@deployer)]
    fun test_attest_token_with_signer(token_bridge: &signer, deployer: &signer) {
        setup(token_bridge, deployer);
        let asset_meta1 = attest_token::attest_token_test<MyCoin>();

        // check that native asset is registered with State
        let token_address = token_hash::derive<MyCoin>();
        assert!(state::native_asset_info(token_address) == type_of<MyCoin>(), 0);

        // attest same token a second time, should have no change in behavior
        let asset_meta2 = attest_token::attest_token_test<MyCoin>();
        assert!(asset_meta1 == asset_meta2, 0);
        assert!(state::native_asset_info(token_address) == type_of<MyCoin>(), 0);
    }
}
