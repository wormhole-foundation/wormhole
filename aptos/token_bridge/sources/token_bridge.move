module token_bridge::token_bridge {
    #[test_only]
    use aptos_framework::account::{Self};
    use aptos_framework::account::{SignerCapability};
    use deployer::deployer::{claim_signer_capability};
    use token_bridge::bridge_state::{init_token_bridge_state};
    use wormhole::wormhole;

    /// Initializes the contract.
    /// The native `init_module` cannot be used, because it runs on each upgrade
    /// (oddly).
    /// Can only be called by the deployer (checked by the
    /// `deployer::claim_signer_capability` function).
    entry fun init(deployer: &signer) {
        let signer_cap = claim_signer_capability(deployer, @token_bridge);
        init_internal(signer_cap);
    }

    fun init_internal(signer_cap: SignerCapability){
        let emitter_cap = wormhole::register_emitter();
        init_token_bridge_state(signer_cap, emitter_cap);
    }

    #[test_only]
    /// Initialise contracts for testing
    /// Returns the token_bridge signer and wormhole signer
    public fun init_test(deployer: &signer) {
        let (_token_bridge, signer_cap) = account::create_resource_account(deployer, b"token_bridge");
        init_internal(signer_cap);
    }
}

#[test_only]
module token_bridge::token_bridge_test {
    use aptos_framework::coin::{Self, MintCapability, FreezeCapability, BurnCapability};
    use aptos_framework::string::{utf8};
    use aptos_framework::type_info::{type_of};
    use aptos_framework::aptos_coin::{Self, AptosCoin};
    use aptos_framework::option::{Self};

    use token_bridge::token_bridge::{Self as bridge};
    use token_bridge::bridge_state::{Self as state};
    use token_bridge::bridge_implementation::{attest_token, attest_token_with_signer, create_wrapped_coin_type};
    use token_bridge::utils::{pad_left_32};
    use token_bridge::token_hash;

    use wormhole::u16::{Self};

    use wrapped_coin::coin::T;

    struct MyCoin has key {}

    struct MyCoinCaps<phantom CoinType> has key, store {
        burn_cap: BurnCapability<CoinType>,
        freeze_cap: FreezeCapability<CoinType>,
        mint_cap: MintCapability<CoinType>,
    }

    struct AptosCoinCaps has key, store {
        burn_cap: BurnCapability<AptosCoin>,
        mint_cap: MintCapability<AptosCoin>,
    }

    fun init_my_token(admin: &signer) {
        let name = utf8(b"mycoindd");
        let symbol = utf8(b"MCdd");
        let decimals = 10;
        let monitor_supply = true;
        let (burn_cap, freeze_cap, mint_cap) = coin::initialize<MyCoin>(admin, name, symbol, decimals, monitor_supply);
        move_to(admin, MyCoinCaps {burn_cap, freeze_cap, mint_cap});
    }

    #[test(aptos_framework=@aptos_framework, deployer=@deployer)]
    fun setup(aptos_framework: &signer, deployer: &signer) {
        wormhole::wormhole_test::setup(aptos_framework);
        bridge::init_test(deployer);
    }

    #[test(aptos_framework=@aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_init_token_bridge(aptos_framework: &signer, deployer: &signer) {
        setup(aptos_framework, deployer);
        let _governance_chain_id = state::governance_chain_id();
    }

    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_attest_token_no_signer(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, deployer);
        init_my_token(token_bridge);
        let (burn_cap, mint_cap) = aptos_coin::initialize_for_test(aptos_framework);
        let fee_coins = coin::mint(100, &mint_cap);
        move_to(token_bridge, AptosCoinCaps {mint_cap: mint_cap, burn_cap: burn_cap});
        coin::register<AptosCoin>(token_bridge); //how important is this registration step and where to check it?
        let _sequence = attest_token<MyCoin>(fee_coins);
        assert!(_sequence==0, 1);
    }

    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_attest_token_with_signer(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, deployer);
        init_my_token(token_bridge);
        let (burn_cap, mint_cap) = aptos_coin::initialize_for_test(aptos_framework);
        let fee_coins = coin::mint(200, &mint_cap);
        coin::register<AptosCoin>(deployer);
        coin::deposit<AptosCoin>(@deployer, fee_coins);
        move_to(token_bridge, AptosCoinCaps {mint_cap: mint_cap, burn_cap: burn_cap});
        coin::register<AptosCoin>(token_bridge); // where else to check registration?
        let _sequence = attest_token_with_signer<MyCoin>(deployer);
        assert!(_sequence==0, 1);

        // check that native asset is registered with State
        let token_address = token_hash::derive<MyCoin>();
        assert!(state::asset_type_info(token_address)==type_of<MyCoin>(), 0);

        // attest same token a second time, should have no change in behavior
        let _sequence = attest_token_with_signer<MyCoin>(deployer);
        assert!(_sequence==1, 2);
    }

    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    #[expected_failure(abort_code = 0x0)]
    fun test_attest_token_no_signer_insufficient_fee(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, deployer);
        init_my_token(token_bridge);
        let (burn_cap, mint_cap) = aptos_coin::initialize_for_test(aptos_framework);
        let fee_coins = coin::mint(0, &mint_cap);
        move_to(token_bridge, AptosCoinCaps {mint_cap: mint_cap, burn_cap: burn_cap});
        coin::register<AptosCoin>(token_bridge); // where else to check registration?
        let _sequence = attest_token<MyCoin>(fee_coins);
        assert!(_sequence==0, 1);
    }

    // test create_wrapped_coin_type and create_wrapped_coin
    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_create_wrapped_coin(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, deployer);
        let vaa = x"010000000001002952fb15d2178bdacbcf05ac5b0e7536d9f0fa60b01e39df468f1ac38cf861306fe0da22948a401fcb85746250cd2ca4d9d32728d0b5955df77eb3ac56dd2dbe010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000002a8c233000200000000000000000000000000000000000000000000000000000000beefface00020c0000000000000000000000000000000000000000000000000000000042454546000000000000000000000000000000000042656566206661636520546f6b656e";
        let _addr = create_wrapped_coin_type(vaa);

        // assert coin is NOT initialized
        let is_initialized = coin::is_coin_initialized<T>();
        assert!(is_initialized==false, 0);

        // initialize coin using type T, move caps to token_bridge, sets bridge state variables
        state::create_wrapped_coin<T>(vaa);

        // assert that coin IS initialized
        let is_initialized = coin::is_coin_initialized<T>();
        assert!(is_initialized==true, 0);

        // assert coin info is correct
        let name = coin::name<T>();
        let symbol = coin::symbol<T>();
        let decimals = coin::decimals<T>();
        assert!(name==utf8(pad_left_32(&b"Beef face Token")), 0);
        assert!(symbol==utf8(pad_left_32(&b"BEEF")), 0);
        assert!(decimals==12, 0);

        // assert origin address, chain, type_info, is_wrapped are correct
        let token_address = token_hash::derive<T>();
        let origin_info = state::origin_info<T>();
        let origin_token_address = state::get_origin_info_token_address(origin_info);
        let origin_token_chain = state::get_origin_info_token_chain(origin_info);
        let wrapped_asset_type_info = state::asset_type_info(token_address);
        let is_wrapped_asset = state::is_wrapped_asset<T>();
        assert!(type_of<T>() == wrapped_asset_type_info, 0); //utf8(b"0xb54071ea68bc35759a17e9ddff91a8394a36a4790055e5bd225fae087a4a875b::coin::T"), 0);
        assert!(origin_token_chain==u16::from_u64(2), 0);
        assert!(origin_token_address==x"00000000000000000000000000000000000000000000000000000000beefface", 0);
        assert!(is_wrapped_asset, 0);

        // load beef face token cap and mint some beef face coins to token_bridge, then burn
        let beef_coins = state::mint_wrapped<T>(10000);
        assert!(coin::value(&beef_coins)==10000, 0);
        coin::register<T>(token_bridge);
        coin::deposit<T>(@token_bridge, beef_coins);
        let supply_before = coin::supply<T>();
        let e = option::borrow(&supply_before);
        assert!(*e==10000, 0);
        state::burn_wrapped<T>(5000);
        let supply_after = coin::supply<T>();
        let e = option::borrow(&supply_after);
        assert!(*e==5000, 0);
    }
}
