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
    /// TODO: the above behaviour has been remedied in the Aptos VM, so we could
    /// use `init_module` now. Let's reconsider before the mainnet launch.
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
    use aptos_framework::signer::{Self};

    use token_bridge::token_bridge::{Self as bridge};
    use token_bridge::bridge_state::{Self as state};
    use token_bridge::transfer_tokens;
    use token_bridge::wrapped;
    use token_bridge::attest_token;
    use token_bridge::utils::{pad_left_32};
    use token_bridge::token_hash;

    use token_bridge::register_chain;

    use wormhole::u16::{Self};

    use wrapped_coin::coin::T;

    /// Registration VAA for the etheruem token bridge 0xdeadbeef
    const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    /// Attestation VAA sent from the ethereum token bridge 0xdeadbeef
    const ATTESTATION_VAA: vector<u8> = x"01000000000100102d399190fa61daccb11c2ea4f7a3db3a9365e5936bcda4cded87c1b9eeb095173514f226256d5579af71d4089eb89496befb998075ba94cd1d4460c5c57b84000000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000002634973000200000000000000000000000000000000000000000000000000000000beefface00020c0000000000000000000000000000000000000000000000000000000042454546000000000000000000000000000000000042656566206661636520546f6b656e";

    struct MyCoin has key {}

    struct MyCoinCaps<phantom CoinType> has key, store {
        burn_cap: BurnCapability<CoinType>,
        freeze_cap: FreezeCapability<CoinType>,
        mint_cap: MintCapability<CoinType>,
    }

    fun init_my_token(admin: &signer) {
        let name = utf8(b"mycoindd");
        let symbol = utf8(b"MCdd");
        let decimals = 10;
        let monitor_supply = true;
        let (burn_cap, freeze_cap, mint_cap) = coin::initialize<MyCoin>(admin, name, symbol, decimals, monitor_supply);
        move_to(admin, MyCoinCaps {burn_cap, freeze_cap, mint_cap});
    }

    fun setup(
        aptos_framework: &signer,
        token_bridge: &signer,
        deployer: &signer,
    ) {
        // we initialise the bridge with zero fees to avoid having to mint fee
        // tokens in these tests. The wormolhe fee handling is already tested
        // in wormhole.move, so it's unnecessary here.
        let (burn_cap, mint_cap) = aptos_coin::initialize_for_test(aptos_framework);
        wormhole::wormhole_test::setup(0);
        bridge::init_test(deployer);

        coin::register<AptosCoin>(deployer);
        coin::register<AptosCoin>(token_bridge); //how important is this registration step and where to check it?
        coin::destroy_burn_cap(burn_cap);
        coin::destroy_mint_cap(mint_cap);
    }

    #[test(aptos_framework=@aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_init_token_bridge(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, token_bridge, deployer);
        let _governance_chain_id = state::governance_chain_id();
    }

    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_attest_token_no_signer(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, token_bridge, deployer);
        init_my_token(token_bridge);
        let _sequence = attest_token::attest_token<MyCoin>(coin::zero());
        assert!(_sequence==0, 1);
    }

    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_attest_token_with_signer(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, token_bridge, deployer);
        init_my_token(token_bridge);
        let _sequence = attest_token::attest_token_with_signer<MyCoin>(deployer);
        assert!(_sequence==0, 1);

        // check that native asset is registered with State
        let token_address = token_hash::derive<MyCoin>();
        assert!(state::asset_type_info(token_address)==type_of<MyCoin>(), 0);

        // attest same token a second time, should have no change in behavior
        let _sequence = attest_token::attest_token_with_signer<MyCoin>(deployer);
        assert!(_sequence==1, 2);
    }

    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    #[expected_failure(abort_code = 0)]
    fun test_create_wrapped_coin_unregistered(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, token_bridge, deployer);

        let _addr = wrapped::create_wrapped_coin_type(ATTESTATION_VAA);
    }


    // test create_wrapped_coin_type and create_wrapped_coin
    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_create_wrapped_coin(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, token_bridge, deployer);
        register_chain::submit_vaa(ETHEREUM_TOKEN_REG);

        let _addr = wrapped::create_wrapped_coin_type(ATTESTATION_VAA);

        // assert coin is NOT initialized
        let is_initialized = coin::is_coin_initialized<T>();
        assert!(is_initialized==false, 0);

        // initialize coin using type T, move caps to token_bridge, sets bridge state variables
        wrapped::create_wrapped_coin<T>(ATTESTATION_VAA);

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
        let origin_token_address = state::get_origin_info_token_address(&origin_info);
        let origin_token_chain = state::get_origin_info_token_chain(&origin_info);
        let wrapped_asset_type_info = state::asset_type_info(token_address);
        let is_wrapped_asset = state::is_wrapped_asset<T>();
        assert!(type_of<T>() == wrapped_asset_type_info, 0); //utf8(b"0xb54071ea68bc35759a17e9ddff91a8394a36a4790055e5bd225fae087a4a875b::coin::T"), 0);
        assert!(origin_token_chain == u16::from_u64(2), 0);
        assert!(origin_token_address == x"00000000000000000000000000000000000000000000000000000000beefface", 0);
        assert!(is_wrapped_asset, 0);

        // load beef face token cap and mint some beef face coins to token_bridge, then burn
        let beef_coins = wrapped::mint<T>(10000);
        assert!(coin::value(&beef_coins)==10000, 0);
        coin::register<T>(token_bridge);
        coin::deposit<T>(@token_bridge, beef_coins);
        let supply_before = coin::supply<T>();
        let e = option::borrow(&supply_before);
        assert!(*e==10000, 0);
        wrapped::burn<T>(5000);
        let supply_after = coin::supply<T>();
        let e = option::borrow(&supply_after);
        assert!(*e==5000, 0);
    }

    // test transfer wrapped coin (with and without payload)
    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_transfer_wrapped_token(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, token_bridge, deployer);
        register_chain::submit_vaa(ETHEREUM_TOKEN_REG);
        // TODO(csongor): create a better error message when attestation is missing
        let _addr = wrapped::create_wrapped_coin_type(ATTESTATION_VAA);
        // TODO(csongor): write a blurb about why this test works (something
        // something static linking)
        // initialize coin using type T, move caps to token_bridge, sets bridge state variables
        wrapped::create_wrapped_coin<T>(ATTESTATION_VAA);

        // test transfer wrapped tokens
        let beef_coins = wrapped::mint<T>(10000);
        let _sequence = transfer_tokens::transfer_tokens<T>(
            beef_coins,
            coin::zero(),
            u16::from_u64(2),
            x"C973E38e87A0571446dC6Ad17C28217F079583C2",
            0,
            0
        );

        //test transfer wrapped tokens with payload
        let beef_coins = wrapped::mint<T>(10000);
        let _sequence = transfer_tokens::transfer_tokens_with_payload<T>(
            beef_coins,
            coin::zero(),
            u16::from_u64(2),
            x"C973E38e87A0571446dC6Ad17C28217F079583C2",
            0,
            x"beeeff",
        );
    }

    // test transfer native coin (with and without payload)
    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_transfer_native_token(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) acquires MyCoinCaps{
        setup(aptos_framework, token_bridge, deployer);
        init_my_token(token_bridge);
        let MyCoinCaps {burn_cap, freeze_cap, mint_cap} = move_from<MyCoinCaps<MyCoin>>(signer::address_of(token_bridge));

        // test transfer native coins
        let my_coins = coin::mint<MyCoin>(10000, &mint_cap);
        let _sequence = transfer_tokens::transfer_tokens<MyCoin>(
            my_coins,
            coin::zero(),
            u16::from_u64(2),
            x"C973E38e87A0571446dC6Ad17C28217F079583C2",
            0,
            0
        );

         // test transfer native coins with payload
        let my_coins = coin::mint<MyCoin>(10000, &mint_cap);
        let _sequence = transfer_tokens::transfer_tokens_with_payload<MyCoin>(
            my_coins,
            coin::zero(),
            u16::from_u64(2),
            x"C973E38e87A0571446dC6Ad17C28217F079583C2",
            0,
            x"beeeff",
        );

        // destroy coin caps
        coin::destroy_mint_cap<MyCoin>(mint_cap);
        coin::destroy_burn_cap<MyCoin>(burn_cap);
        coin::destroy_freeze_cap<MyCoin>(freeze_cap);
    }

}
