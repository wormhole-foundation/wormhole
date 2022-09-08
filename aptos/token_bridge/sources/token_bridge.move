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

    use token_bridge::token_bridge::{Self as bridge};
    use token_bridge::bridge_state::{Self as state};
    use token_bridge::bridge_implementation::{attest_token, attest_token_with_signer};
    use token_bridge::utils::{hash_type_info};

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
        let monitor_supply = false;
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
        let token_address = hash_type_info<MyCoin>();
        assert!(state::native_asset(token_address)==type_of<MyCoin>(), 0);

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
}
