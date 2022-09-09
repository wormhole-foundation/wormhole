module token_bridge::wrapped {
    use aptos_framework::account::{create_resource_account};
    use aptos_framework::signer::{address_of};
    use aptos_framework::coin::{Self, Coin, MintCapability, BurnCapability, FreezeCapability};
    use aptos_framework::string;

    use wormhole::vaa;

    use token_bridge::bridge_state as state;
    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::deploy_coin::{deploy_coin};
    use token_bridge::vaa as token_bridge_vaa;

    //friend token_bridge::token_bridge;

    #[test_only]
    friend token_bridge::token_bridge_test;
    friend token_bridge::complete_transfer_test;

    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;

    const E_IS_NOT_WRAPPED_ASSET: u64 = 0;
    const E_COIN_CAP_DOES_NOT_EXIST: u64 = 1;

    struct CoinCapabilities<phantom CoinType> has key, store {
        mint_cap: MintCapability<CoinType>,
        freeze_cap: FreezeCapability<CoinType>,
        burn_cap: BurnCapability<CoinType>,
    }

    // this function is called before create_wrapped_coin
    public entry fun create_wrapped_coin_type(vaa: vector<u8>): address {
        // NOTE: we do not do replay protection here, only verify that the VAA
        // comes from a known emitter. This is because `create_wrapped_coin`
        // itself will need to verify the VAA again in a separate transaction,
        // and it itself will perform the replay protection.
        // This function cannot be called twice with the same VAA because it
        // creates a resource account, which will fail the second time if the
        // account already exists.
        // TODO(csongor): should we implement a more explicit replay protection
        // for this function?
        // TODO(csongor): we could break this function up a little so it's
        // better testable. In particular, resource accounts are little hard to
        // test.
        let vaa = token_bridge_vaa::parse_and_verify(vaa);
        let asset_meta:AssetMeta = asset_meta::parse(vaa::destroy(vaa));
        let seed = asset_meta::create_seed(&asset_meta);

        //create resource account
        let token_bridge_signer = state::token_bridge_signer();
        let (new_signer, new_cap) = create_resource_account(&token_bridge_signer, seed);

        let token_address = asset_meta::get_token_address(&asset_meta);
        let token_chain = asset_meta::get_token_chain(&asset_meta);
        let origin_info = state::create_origin_info(token_address, token_chain);

        deploy_coin(&new_signer);
        state::set_wrapped_asset_signer_capability(origin_info, new_cap);

        // return address of the new signer
        address_of(&new_signer)
    }

    // this function is called in tandem with bridge_implementation::create_wrapped_coin_type
    // initializes a coin for CoinType, updates mappings in State
    public entry fun create_wrapped_coin<CoinType>(vaa: vector<u8>) {
        let vaa = token_bridge_vaa::parse_verify_and_replay_protect(vaa);
        let asset_meta: AssetMeta = asset_meta::parse(vaa::destroy(vaa));

        let native_token_address = asset_meta::get_token_address(&asset_meta);
        let native_token_chain = asset_meta::get_token_chain(&asset_meta);
        let native_info = state::create_origin_info(native_token_address, native_token_chain);

        // TODO: where do we check that CoinType corresponds to the thing in the VAA?
        // I think it's fine because only the correct signer can initialise the
        // coin, so it would fail, but we should have a test for this.
        let coin_signer = state::get_wrapped_asset_signer(native_info);
        init_wrapped_coin<CoinType>(&coin_signer, &asset_meta)
    }

    public(friend) fun init_wrapped_coin<CoinType>(
        coin_signer: &signer,
        asset_meta: &AssetMeta,
    ) {
        // initialize new coin using CoinType
        let name = asset_meta::get_name(asset_meta);
        let symbol = asset_meta::get_symbol(asset_meta);
        let decimals = asset_meta::get_decimals(asset_meta);
        let monitor_supply = true;
        let (burn_cap, freeze_cap, mint_cap)
            = coin::initialize<CoinType>(
                coin_signer,
                string::utf8(name),
                string::utf8(symbol),
                decimals,
                monitor_supply
            );

        let token_address = asset_meta::get_token_address(asset_meta);
        let token_chain = asset_meta::get_token_chain(asset_meta);
        let origin_info = state::create_origin_info(token_address, token_chain);

        // update the following two mappings in State
        // 1. (native chain, native address) => wrapped address
        // 2. wrapped address => (native chain, native address)
        state::setup_wrapped<CoinType>(coin_signer, origin_info);

        // store coin capabilities
        let token_bridge = state::token_bridge_signer();
        move_to(&token_bridge, CoinCapabilities { mint_cap, freeze_cap, burn_cap });
    }

    public(friend) fun mint<CoinType>(amount:u64): Coin<CoinType> acquires CoinCapabilities {
        assert!(state::is_wrapped_asset<CoinType>(), E_IS_NOT_WRAPPED_ASSET);
        assert!(exists<CoinCapabilities<CoinType>>(@token_bridge), E_COIN_CAP_DOES_NOT_EXIST);
        let caps = borrow_global<CoinCapabilities<CoinType>>(@token_bridge);
        let mint_cap = &caps.mint_cap;
        let coins = coin::mint<CoinType>(amount, mint_cap);
        coins
    }

    public(friend) fun burn<CoinType>(amount:u64) acquires CoinCapabilities {
        assert!(state::is_wrapped_asset<CoinType>(), E_IS_NOT_WRAPPED_ASSET);
        assert!(exists<CoinCapabilities<CoinType>>(@token_bridge), E_COIN_CAP_DOES_NOT_EXIST);
        let caps = borrow_global<CoinCapabilities<CoinType>>(@token_bridge);
        let burn_cap = &caps.burn_cap;
        let token_bridge = state::token_bridge_signer();
        // TODO: this looks wrong to me. burn should just take the coins to burn.
        let coins = coin::withdraw<CoinType>(&token_bridge, amount);
        coin::burn<CoinType>(coins, burn_cap);
    }

}

#[test_only]
module token_bridge::wrapped_test {

}
