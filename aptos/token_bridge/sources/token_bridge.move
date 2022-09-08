module token_bridge::token_bridge {
    #[test_only]
    use aptos_framework::account::{Self};
    use 0x1::account::{SignerCapability};
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
    //use aptos_framework::signer::{address_of};
    //use aptos_framework::aptos_coin::{Self};//, AptosCoin};
    //use aptos_framework::debug::{print};

    use token_bridge::token_bridge;
    use token_bridge::bridge_state::{Self};
    //use token_bridge::bridge_state::{Self, State, token_bridge_signer};
    //use token_bridge::bridge_implementation::{attest_token};

    struct MyCoin has key {}

    struct MyCoinCaps<phantom CoinType> has key, store {
        burn_cap: BurnCapability<CoinType>,
        freeze_cap: FreezeCapability<CoinType>,
        mint_cap: MintCapability<CoinType>,
    }

    fun init_my_token(admin: &signer) {
        let name = utf8(b"my coin");
        let symbol = utf8(b"MC");
        let decimals = 10;
        let monitor_supply = false;
        let (burn_cap, freeze_cap, mint_cap) = coin::initialize<MyCoin>(admin, name, symbol, decimals, monitor_supply);
        move_to(admin, MyCoinCaps {burn_cap, freeze_cap, mint_cap});
    }

    #[test(aptos_framework = @aptos_framework, deployer=@deployer)]
    fun setup(aptos_framework: &signer, deployer: &signer) {
        wormhole::wormhole_test::setup(aptos_framework);
        token_bridge::init_test(deployer);
    }

    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    public fun test_init_token_bridge(aptos_framework: &signer, deployer: &signer) {
        setup(aptos_framework, deployer);
        let _governance_chain_id = bridge_state::governance_chain_id();
        //assert!(exists<State>(address_of(&bridge_signer)), 0);
        //print()
    }

    // #[test(aptos_framework = @aptos_framework)]
    // public fun test_attest_token(aptos_framework: &signer) {
    //     setup(aptos_framework);
    //     init_my_token(&token_bridge_signer()); //initialize native token to ne attested
    //     let (burn_cap, mint_cap) = aptos_coin::initialize_for_test(aptos_framework);
    //     let fees_coins = coin::mint(100, &mint_cap);
    //     coin::destroy_burn_cap<AptosCoin>(burn_cap);
    //     coin::destroy_mint_cap<AptosCoin>(mint_cap);
    //     let _sequence = attest_token<MyCoin>(fees_coins);
    // }
    // expect fail if not enough message fee
}
