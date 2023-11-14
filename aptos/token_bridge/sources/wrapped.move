module token_bridge::wrapped {
    use aptos_framework::account;
    use aptos_framework::coin::{Self, Coin, MintCapability, BurnCapability, FreezeCapability};

    use wormhole::vaa;

    use token_bridge::state;
    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::deploy_coin::{deploy_coin};
    use token_bridge::vaa as token_bridge_vaa;
    use token_bridge::string32;

    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::transfer_tokens;

    #[test_only]
    friend token_bridge::transfer_tokens_test;
    #[test_only]
    friend token_bridge::wrapped_test;
    #[test_only]
    friend token_bridge::complete_transfer_test;

    const E_IS_NOT_WRAPPED_ASSET: u64 = 0;
    const E_COIN_CAP_DOES_NOT_EXIST: u64 = 1;

    struct CoinCapabilities<phantom CoinType> has key, store {
        mint_cap: MintCapability<CoinType>,
        freeze_cap: FreezeCapability<CoinType>,
        burn_cap: BurnCapability<CoinType>,
    }

    // this function is called before create_wrapped_coin
    // TODO(csongor): document why these two are in separate transactions
    public entry fun create_wrapped_coin_type(vaa: vector<u8>) {
        // NOTE: we do not do replay protection here, only verify that the VAA
        // comes from a known emitter. This is because `create_wrapped_coin`
        // itself will need to verify the VAA again in a separate transaction,
        // and it itself will perform the replay protection.
        // This function cannot be called twice with the same VAA because it
        // creates a resource account, which will fail the second time if the
        // account already exists.
        // TODO(csongor): should we implement a more explicit replay protection
        // for this function?
        let vaa = token_bridge_vaa::parse_and_verify(vaa);
        let asset_meta = asset_meta::parse(vaa::destroy(vaa));
        let seed = asset_meta::create_seed(&asset_meta);

        //create resource account
        let token_bridge_signer = state::token_bridge_signer();
        let (new_signer, new_cap) = account::create_resource_account(&token_bridge_signer, seed);

        let token_address = asset_meta::get_token_address(&asset_meta);
        let token_chain = asset_meta::get_token_chain(&asset_meta);
        let origin_info = state::create_origin_info(token_chain, token_address);

        deploy_coin(&new_signer);
        state::set_wrapped_asset_signer_capability(origin_info, new_cap);
    }

    // this function is called in tandem with bridge_implementation::create_wrapped_coin_type
    // initializes a coin for CoinType, updates mappings in State
    public entry fun create_wrapped_coin<CoinType>(vaa: vector<u8>) {
        let vaa = token_bridge_vaa::parse_verify_and_replay_protect(vaa);
        let asset_meta: AssetMeta = asset_meta::parse(vaa::destroy(vaa));

        let native_token_address = asset_meta::get_token_address(&asset_meta);
        let native_token_chain = asset_meta::get_token_chain(&asset_meta);
        let origin_info = state::create_origin_info(native_token_chain, native_token_address);

        // The CoinType type variable is instantiated by the caller of the
        // function, so a malicious actor could try and pass in something other
        // than what we're expecting based on the VAA. So how do we protect
        // against this? The signer capability is keyed by the origin info of
        // the token, and a coin can only be initialised by the signer that owns
        // the module that defines the CoinType.
        // See the `test_create_wrapped_coin_bad_type` negative test below.
        let coin_signer = state::get_wrapped_asset_signer(origin_info);
        init_wrapped_coin<CoinType>(&coin_signer, &asset_meta);
    }

    public(friend) fun init_wrapped_coin<CoinType>(
        coin_signer: &signer,
        asset_meta: &AssetMeta,
    ) {
        // initialize new coin using CoinType
        let name = asset_meta::get_name(asset_meta);
        let symbol = asset_meta::get_symbol(asset_meta);

        // The amounts in the token bridge payload are truncated to 8 decimals
        // in each of the contracts when sending tokens out, so there's no
        // precision beyond 10^-8. We could preserve the original number of
        // decimals when creating wrapped assets, and "untruncate" the amounts
        // on the way out by scaling back appropriately. This is what most other
        // chains do, but untruncating from 8 decimals to 18 decimals loses
        // log2(10^10) ~ 33 bits of precision, which we cannot afford on Aptos
        // (and Solana), as the coin type only has 64bits to begin with.
        // Contrast with Ethereum, where amounts are 256 bits.
        // So we cap the maximum decimals at 8 when creating a wrapped token.
        let max_decimals: u8 = 8;
        let parsed_decimals = asset_meta::get_decimals(asset_meta);
        let decimals = if (max_decimals < parsed_decimals) max_decimals else parsed_decimals;

        let monitor_supply = true;
        let (burn_cap, freeze_cap, mint_cap)
            = coin::initialize<CoinType>(
                coin_signer,
                string32::to_string(&name),
                // take the first 10 characters of the symbol (maximum in aptos)
                string32::take_utf8(string32::to_string(&symbol), 10),
                decimals,
                monitor_supply
            );

        let token_address = asset_meta::get_token_address(asset_meta);
        let token_chain = asset_meta::get_token_chain(asset_meta);
        let origin_info = state::create_origin_info(token_chain, token_address);

        // update the following two mappings in State
        // 1. (native chain, native address) => wrapped address
        // 2. wrapped address => (native chain, native address)
        state::setup_wrapped<CoinType>(origin_info);

        // store coin capabilities
        let token_bridge = state::token_bridge_signer();
        move_to(&token_bridge, CoinCapabilities { mint_cap, freeze_cap, burn_cap });
    }

    public(friend) fun mint<CoinType>(amount: u64): Coin<CoinType> acquires CoinCapabilities {
        assert!(state::is_wrapped_asset<CoinType>(), E_IS_NOT_WRAPPED_ASSET);
        assert!(exists<CoinCapabilities<CoinType>>(@token_bridge), E_COIN_CAP_DOES_NOT_EXIST);
        let caps = borrow_global<CoinCapabilities<CoinType>>(@token_bridge);
        let mint_cap = &caps.mint_cap;
        let coins = coin::mint<CoinType>(amount, mint_cap);
        coins
    }

    public(friend) fun burn<CoinType>(coins: Coin<CoinType>) acquires CoinCapabilities {
        assert!(state::is_wrapped_asset<CoinType>(), E_IS_NOT_WRAPPED_ASSET);
        assert!(exists<CoinCapabilities<CoinType>>(@token_bridge), E_COIN_CAP_DOES_NOT_EXIST);
        let caps = borrow_global<CoinCapabilities<CoinType>>(@token_bridge);
        let burn_cap = &caps.burn_cap;
        coin::burn<CoinType>(coins, burn_cap);
    }

}

#[test_only]
module token_bridge::wrapped_test {
    use aptos_framework::account;
    use aptos_framework::coin;
    use aptos_framework::string::{utf8};
    use aptos_framework::type_info::{type_of};
    use aptos_framework::option;

    use token_bridge::token_bridge::{Self as bridge};
    use token_bridge::state;
    use token_bridge::wrapped;
    use token_bridge::asset_meta;
    use token_bridge::string32;

    use token_bridge::register_chain;

    use wormhole::u16::{Self};
    use wrapped_coin::coin::T;
    use wormhole::external_address::{Self};

    /// Registration VAA for the ethereum token bridge 0xdeadbeef
    const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    /// Attestation VAA sent from the ethereum token bridge 0xdeadbeef
    const ATTESTATION_VAA: vector<u8> = x"0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";

    fun setup(
        deployer: &signer,
    ) {
        wormhole::wormhole_test::setup(0);
        bridge::init_test(deployer);
    }

    public fun init_wrapped_token() {
        let chain = wormhole::u16::from_u64(2);
        let token_address = external_address::from_bytes(x"deadbeef");
        let asset_meta = asset_meta::create(
            token_address,
            chain,
            9, // this will get truncated to 8
            string32::from_bytes(b"foo"),
            string32::from_bytes(b"Foo bar token")
        );
        let wrapped_coin = account::create_account_for_test(@wrapped_coin);

        // set up the signer capability first
        let signer_cap = account::create_test_signer_cap(@wrapped_coin);
        let origin_info = state::create_origin_info(chain, token_address);
        state::set_wrapped_asset_signer_capability(origin_info, signer_cap);

        wrapped::init_wrapped_coin<wrapped_coin::coin::T>(&wrapped_coin, &asset_meta);
    }


    #[test(deployer=@deployer)]
    #[expected_failure(abort_code = 0, location = token_bridge::vaa)]
    fun test_create_wrapped_coin_unregistered(deployer: &signer) {
        setup(deployer);

        wrapped::create_wrapped_coin_type(ATTESTATION_VAA);
    }

    struct YourCoin {}

    // This test ensures that I can't take a valid attestation VAA and trick the
    // token bridge to register my own type. I think what that could lead to is
    // a denial of service in case the 3rd party type belongs to a module
    // with an 'arbitrary' upgrade policy which can be deleted in the future.
    // This upgrade policy is not enabled in the VM as of writing, but that
    // might well change in the future, so we future proof ourselves here.
    #[test(deployer=@deployer)]
    #[expected_failure(abort_code = 65537, location = 0000000000000000000000000000000000000000000000000000000000000001::coin)] // ECOIN_INFO_ADDRESS_MISMATCH
    fun test_create_wrapped_coin_bad_type(deployer: &signer) {
        setup(deployer);
        register_chain::submit_vaa(ETHEREUM_TOKEN_REG);
        wrapped::create_wrapped_coin_type(ATTESTATION_VAA);

        // initialize coin using type T, move caps to token_bridge, sets bridge state variables
        wrapped::create_wrapped_coin<YourCoin>(ATTESTATION_VAA);
    }

    // test create_wrapped_coin_type and create_wrapped_coin
    #[test(deployer=@deployer)]
    fun test_create_wrapped_coin(deployer: &signer) {
        setup(deployer);
        register_chain::submit_vaa(ETHEREUM_TOKEN_REG);

        wrapped::create_wrapped_coin_type(ATTESTATION_VAA);

        // assert coin is NOT initialized
        assert!(!coin::is_coin_initialized<T>(), 0);

        // initialize coin using type T, move caps to token_bridge, sets bridge state variables
        wrapped::create_wrapped_coin<T>(ATTESTATION_VAA);

        // assert that coin IS initialized
        assert!(coin::is_coin_initialized<T>(), 0);

        // assert coin info is correct
        assert!(coin::name<T>() == utf8(b"Beef face Token"), 0);
        assert!(coin::symbol<T>() == utf8(b"BEEF"), 0);
        assert!(coin::decimals<T>() == 8, 0); // truncated correctly to 8 from 12

        // assert origin address, chain, type_info, is_wrapped are correct
        let origin_info = state::origin_info<T>();
        let origin_token_address = state::get_origin_info_token_address(&origin_info);
        let origin_token_chain = state::get_origin_info_token_chain(&origin_info);
        let wrapped_asset_type_info = state::wrapped_asset_info(origin_info);
        let is_wrapped_asset = state::is_wrapped_asset<T>();
        assert!(type_of<T>() == wrapped_asset_type_info, 0); //utf8(b"0xf4f53cc591e5190eddbc43940746e2b5deea6e0e1562b2bba765d488504842c7::coin::T"), 0);
        assert!(origin_token_chain == u16::from_u64(2), 0);
        assert!(external_address::get_bytes(&origin_token_address) == x"00000000000000000000000000000000000000000000000000000000beefface", 0);
        assert!(is_wrapped_asset, 0);

        // load beef face token cap and mint some beef face coins, then burn
        let beef_coins = wrapped::mint<T>(10000);
        assert!(coin::value(&beef_coins)==10000, 0);
        assert!(coin::supply<T>() == option::some(10000), 0);
        wrapped::burn<T>(beef_coins);
        assert!(coin::supply<T>() == option::some(0), 0);
    }
}
