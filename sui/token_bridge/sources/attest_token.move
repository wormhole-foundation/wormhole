module token_bridge::attest_token {
    use std::string::{Self};
    use sui::balance::{Balance};
    use sui::coin::{Self, CoinMetadata};
    use sui::sui::{SUI};
    use wormhole::state::{State as WormholeState};

    use token_bridge::asset_meta::{Self};
    use token_bridge::state::{Self, State};

    const E_REGISTERED_WRAPPED_ASSET: u64 = 0;

    public fun attest_token<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        wormhole_fee: Balance<SUI>,
        coin_metadata: &CoinMetadata<CoinType>,
        nonce: u32,
    ): u64 {
        // Register if it is a new asset.
        //
        // NOTE: We don't want to abort if the asset is already registered
        // because we may want to send asset metadata again after registration
        // (the owner of a particular `CoinType` can change `CoinMetadata` any
        // time after we register the asset).
        if (state::is_registered_asset<CoinType>(token_bridge_state)) {
            // If this asset is already registered, make sure it is not a
            // Token Bridge wrapped asset.
            assert!(
                state::is_native_asset<CoinType>(token_bridge_state),
                E_REGISTERED_WRAPPED_ASSET
            )
        } else {
            state::register_native_asset(token_bridge_state, coin_metadata);
        };

        let payload = serialize_asset_meta(token_bridge_state, coin_metadata);
        state::publish_wormhole_message(
            token_bridge_state,
            worm_state,
            nonce,
            payload,
            wormhole_fee
        )
    }

    fun serialize_asset_meta<CoinType>(
        token_bridge_state: &State,
        metadata: &CoinMetadata<CoinType>,
    ): vector<u8> {
        // Get canonical token info.
        let (
            token_chain,
            token_address
        ) = state::token_info<CoinType>(token_bridge_state);

        asset_meta::serialize(
            asset_meta::new(
                token_address,
                token_chain,
                state::coin_decimals<CoinType>(token_bridge_state),
                string::from_ascii(coin::get_symbol(metadata)),
                coin::get_name(metadata)
            )
        )
    }

    #[test_only]
    public fun serialize_asset_meta_test_only<CoinType>(
        token_bridge_state: &State,
        coin_metadata: &CoinMetadata<CoinType>,
    ): vector<u8> {
        serialize_asset_meta(token_bridge_state, coin_metadata)
    }
}

#[test_only]
module token_bridge::attest_token_tests {
    use sui::balance::{Self};
    use sui::test_scenario::{Self};

    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        person,
        return_state,
        return_states,
        set_up_wormhole_and_token_bridge,
        take_state,
        take_states
    };

    #[test]
    fun test_attest_token() {
        use token_bridge::attest_token::{attest_token};

        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Init the native coin
        test_scenario::next_tx(scenario, user); {
            coin_native_10::init_test_only(
                test_scenario::ctx(scenario)
            );
        };

        // Proceed to next operation.
        test_scenario::next_tx(scenario, user);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let coin_meta = coin_native_10::take_metadata(scenario);

        // Emit `AssetMeta` payload.
        let sequence = attest_token(
            &mut token_bridge_state,
            &mut worm_state,
            balance::create_for_testing(
                wormhole_fee
            ),
            &coin_meta,
            1234 // nonce
        );
        assert!(sequence == 0, 0);

        // Check that Wormhole message was emitted.
        let effects = test_scenario::next_tx(scenario, user);
        let num_events = test_scenario::num_user_events(&effects);
        assert!(num_events == 1, 0);

        // Check that asset is registered.
        assert!(
            state::is_registered_asset<COIN_NATIVE_10>(
                &token_bridge_state
            ),
            0
        );

        // TODO: check token info.

        // We should be able to call `attest_token` any time after.
        let sequence = attest_token(
            &mut token_bridge_state,
            &mut worm_state,
            balance::create_for_testing(wormhole_fee),
            &coin_meta,
            1234 // nonce
        );
        assert!(sequence == 1, 0);

        // TODO: verify token info has not changed.

        // Clean up.
        return_states(token_bridge_state, worm_state);
        test_scenario::return_shared(coin_meta);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_serialize_asset_meta_test_only() {
        use token_bridge::attest_token::{serialize_asset_meta_test_only};

        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        let wormhole_fee = 0;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Init the native coin
        test_scenario::next_tx(scenario, user); {
            coin_native_10::init_test_only(
                test_scenario::ctx(scenario)
            );
        };

        // Proceed to next operation.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);
        let coin_meta = coin_native_10::take_metadata(scenario);

        state::register_native_asset_test_only(&mut token_bridge_state, &coin_meta);

        let serialized =
            serialize_asset_meta_test_only(
                &token_bridge_state,
                &coin_meta,
            );
        let expected =
            x"02000000000000000000000000000000000000000000000000000000000000000100150a4445433130000000000000000000000000000000000000000000000000000000446563696d616c73203130000000000000000000000000000000000000000000";
        assert!(serialized == expected, 0);

        // Clean up.
        return_state(token_bridge_state);
        coin_native_10::return_metadata(coin_meta);

        // Done.
        test_scenario::end(my_scenario);
    }
}
