#[test_only]
module coin_wrapped_12::coin {
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::create_wrapped::{Self};

    struct COIN has drop {}

    const VAA: vector<u8> =
        x"0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";

    const UPDATED_VAA: vector<u8> =
        x"01000000000100b0571650590e147fce4eb60105e0463522c1244a97bd5dcb365d3e7bc7f32e4071e18c31bd8240bff6451991c86cb9176003379ba470a5124245b60547516ecc010000000000000045000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f0200000000000000000000000000000000000000000000000000000000beefface00020c0000000000000000000000000000424545463f3f3f20616e642070726f66697400000042656566206661636520546f6b656e3f3f3f20616e642070726f666974";

    fun init(witness: COIN, ctx: &mut TxContext) {
        transfer::public_transfer(
            create_wrapped::prepare_registration(
                witness,
                VAA,
                ctx
            ),
            tx_context::sender(ctx)
        );
    }

    public fun encoded_vaa(): vector<u8> {
        VAA
    }

    public fun encoded_updated_vaa(): vector<u8> {
        UPDATED_VAA
    }

    public fun token_meta(): AssetMeta {
        asset_meta::deserialize(
            wormhole::vaa::peel_payload_from_vaa(&VAA)
        )
    }

    public fun updated_token_meta(): AssetMeta {
        asset_meta::deserialize(
            wormhole::vaa::peel_payload_from_vaa(&UPDATED_VAA)
        )
    }

    #[test_only]
    /// NOTE: Even though this module is `#[test_only]`, this method is tagged
    /// with the same macro  as a trick to allow another method within this
    /// module to call `init` using OTW.
    public fun init_test_only(ctx: &mut TxContext) {
        init(COIN {}, ctx)
    }
}

#[test_only]
module coin_wrapped_12::coin_tests {
    use sui::test_scenario::{Self};
    use token_bridge::asset_meta::{Self};
    use token_bridge::create_wrapped::{Self, WrappedAssetSetup};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        register_dummy_emitter,
        return_clock,
        return_states,
        set_up_wormhole_and_token_bridge,
        take_clock,
        take_states,
        two_people
    };
    use token_bridge::token_registry::{Self};
    use token_bridge::wrapped_asset::{Self};

    use coin_wrapped_12::coin::{Self as coin_wrapped_12, COIN};

    #[test]
    public fun test_native_decimals() {
        let meta = coin_wrapped_12::token_meta();
        assert!(asset_meta::native_decimals(&meta) == 12, 0);
        asset_meta::destroy(meta);
    }

    #[test]
    public fun test_complete_and_update_attestation() {
        let (caller, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Ignore effects. Make sure `coin_deployer` receives
        // `WrappedAssetSetup`.
        test_scenario::next_tx(scenario, coin_deployer);

        // Publish coin.
        coin_wrapped_12::init_test_only(test_scenario::ctx(scenario));

        test_scenario::next_tx(scenario, coin_deployer);

        let wrapped_asset_setup =
            test_scenario::take_from_address<WrappedAssetSetup<COIN>>(
                scenario,
                coin_deployer
            );

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        create_wrapped::complete_registration(
            &mut token_bridge_state,
            &worm_state,
            wrapped_asset_setup,
            &the_clock,
            test_scenario::ctx(scenario)
        );

        let (
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        ) = asset_meta::unpack(coin_wrapped_12::token_meta());

        // Check registry.
        {
            let verified = state::verified_asset<COIN>(&token_bridge_state);
            assert!(token_registry::is_wrapped<COIN>(&verified), 0);

            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset =
                token_registry::borrow_wrapped<COIN>(registry);
            assert!(wrapped_asset::total_supply(asset) == 0, 0);

            // Decimals are capped for this wrapped asset.
            assert!(wrapped_asset::decimals(asset) == 8, 0);

            // Check metadata against asset metadata.
            let metadata = wrapped_asset::metadata(asset);
            assert!(wrapped_asset::token_chain(metadata) == token_chain, 0);
            assert!(wrapped_asset::token_address(metadata) == token_address, 0);
            assert!(
                wrapped_asset::native_decimals(metadata) == native_decimals,
                0
            );
            assert!(wrapped_asset::symbol(metadata) == symbol, 0);
            assert!(wrapped_asset::name(metadata) == name, 0);
        };

        // Now update metadata.
        create_wrapped::update_attestation<COIN>(
            &mut token_bridge_state,
            &worm_state,
            coin_wrapped_12::encoded_updated_vaa(),
            &the_clock
        );

        // Check updated name and symbol.
        let registry = state::borrow_token_registry(&token_bridge_state);
        let asset = token_registry::borrow_wrapped<COIN>(registry);
        let metadata = wrapped_asset::metadata(asset);
        let (
            _,
            _,
            _,
            new_symbol,
            new_name
        ) = asset_meta::unpack(coin_wrapped_12::updated_token_meta());

        assert!(symbol != new_symbol, 0);
        assert!(wrapped_asset::symbol(metadata) == new_symbol, 0);

        assert!(name != new_name, 0);
        assert!(wrapped_asset::name(metadata) == new_name, 0);

        // Clean up.
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }
}
