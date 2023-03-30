module token_bridge::attest_token {
    use sui::balance::{Balance};
    use sui::clock::{Clock};
    use sui::coin::{CoinMetadata};
    use sui::sui::{SUI};
    use wormhole::state::{State as WormholeState};

    use token_bridge::asset_meta::{Self};
    use token_bridge::state::{Self, State};
    use token_bridge::token_registry::{Self};
    use token_bridge::version_control::{AttestToken as AttestTokenControl};

    const E_REGISTERED_WRAPPED_ASSET: u64 = 0;

    public fun attest_token<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        wormhole_fee: Balance<SUI>,
        coin_metadata: &CoinMetadata<CoinType>,
        nonce: u32,
        the_clock: &Clock
    ): u64 {
        state::check_minimum_requirement<AttestTokenControl>(
            token_bridge_state
        );

        let encoded_asset_meta =
            serialize_asset_meta(token_bridge_state, coin_metadata);

        state::publish_wormhole_message(
            token_bridge_state,
            worm_state,
            nonce,
            encoded_asset_meta,
            wormhole_fee,
            the_clock
        )
    }

    fun serialize_asset_meta<CoinType>(
        token_bridge_state: &mut State,
        coin_metadata: &CoinMetadata<CoinType>,
    ): vector<u8> {
        let registry = state::borrow_token_registry_mut(token_bridge_state);

        // Register if it is a new asset.
        //
        // NOTE: We don't want to abort if the asset is already registered
        // because we may want to send asset metadata again after registration
        // (the owner of a particular `CoinType` can change `CoinMetadata` any
        // time after we register the asset).
        if (token_registry::has<CoinType>(registry)) {
            // If this asset is already registered, there should already
            // be canonical info associated with this coin type.
            let verified = token_registry::new_asset_cap<CoinType>(registry);
            token_registry::checked_token_address(&verified, registry)
        } else {
            // Otherwise, register it.
            token_registry::add_new_native(registry, coin_metadata)
        };

        asset_meta::serialize(asset_meta::from_metadata(coin_metadata))
    }

    #[test_only]
    public fun serialize_asset_meta_test_only<CoinType>(
        token_bridge_state: &mut State,
        coin_metadata: &CoinMetadata<CoinType>,
    ): vector<u8> {
        serialize_asset_meta(token_bridge_state, coin_metadata)
    }
}

#[test_only]
module token_bridge::attest_token_tests {
    use std::ascii::{Self};
    use std::string::{Self};
    use sui::balance::{Self};
    use sui::coin::{Self};
    use sui::test_scenario::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::asset_meta::{Self};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::native_asset::{Self};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        person,
        return_clock,
        return_state,
        return_states,
        set_up_wormhole_and_token_bridge,
        take_clock,
        take_state,
        take_states
    };
    use token_bridge::token_registry::{Self};

    #[test]
    fun test_attest_token() {
        use token_bridge::attest_token::{attest_token};

        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Publish coin.
        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);
        let coin_meta = coin_native_10::take_metadata(scenario);

        // Emit `AssetMeta` payload.
        let sequence =
            attest_token(
                &mut token_bridge_state,
                &mut worm_state,
                balance::create_for_testing(
                    wormhole_fee
                ),
                &coin_meta,
                1234, // nonce
                &the_clock
            );
        assert!(sequence == 0, 0);

        // Check that Wormhole message was emitted.
        let effects = test_scenario::next_tx(scenario, user);
        let num_events = test_scenario::num_user_events(&effects);
        assert!(num_events == 1, 0);

        // Check that asset is registered.
        {
            let registry =
                state::borrow_token_registry(&token_bridge_state);
            assert!(token_registry::is_native<COIN_NATIVE_10>(registry), 0);

            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);

            let expected_token_address =
                native_asset::canonical_address(&coin_meta);
            assert!(
                native_asset::token_address(asset) == expected_token_address,
                0
            );
            assert!(native_asset::decimals(asset) == 10, 0);

            let (
                token_chain,
                token_address
            ) = native_asset::canonical_info(asset);
            assert!(token_chain == chain_id(), 0);
            assert!(token_address == expected_token_address, 0);

            assert!(native_asset::balance(asset) == 0, 0);
        };

        // Update metadata.
        let new_symbol = {
            use std::vector::{Self};

            let symbol = coin::get_symbol(&coin_meta);
            let buf = ascii::into_bytes(symbol);
            vector::reverse(&mut buf);

            ascii::string(buf)
        };

        let new_name = coin::get_name(&coin_meta);
        string::append(&mut new_name, string::utf8(b"??? and profit"));

        let treasury_cap = coin_native_10::take_treasury_cap(scenario);
        coin::update_symbol(&treasury_cap, &mut coin_meta, new_symbol);
        coin::update_name(&treasury_cap, &mut coin_meta, new_name);

        // We should be able to call `attest_token` any time after.
        let sequence =
            attest_token(
                &mut token_bridge_state,
                &mut worm_state,
                balance::create_for_testing(wormhole_fee),
                &coin_meta,
                1234, // nonce
                &the_clock
            );
        assert!(sequence == 1, 0);

        // Clean up.
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        coin_native_10::return_globals(treasury_cap, coin_meta);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_serialize_asset_meta() {
        use token_bridge::attest_token::{serialize_asset_meta_test_only};

        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Publish coin.
        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Proceed to next operation.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);
        let coin_meta = coin_native_10::take_metadata(scenario);

        // Emit `AssetMeta` payload.
        let serialized =
            serialize_asset_meta_test_only(&mut token_bridge_state, &coin_meta);
        assert!(
            serialized == asset_meta::serialize(asset_meta::from_metadata(&coin_meta)),
            0
        );

        // Clean up.
        return_state(token_bridge_state);
        coin_native_10::return_metadata(coin_meta);

        // Done.
        test_scenario::end(my_scenario);
    }
}
