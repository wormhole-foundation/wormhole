// SPDX-License-Identifier: Apache 2

/// This module implements the method `attest_token` which allows someone
/// to send asset metadata of a coin type native to Sui. Part of this process
/// is registering this asset in the `TokenRegistry`.
///
/// NOTE: If an asset has not been attested for, it cannot be bridged using
/// `transfer_tokens` or `transfer_tokens_with_payload`.
///
/// See `asset_meta` module for serialization and deserialization of Wormhole
/// message payload.
module token_bridge::attest_token {
    use sui::coin::{CoinMetadata};
    use wormhole::publish_message::{MessageTicket};

    use token_bridge::asset_meta::{Self};
    use token_bridge::create_wrapped::{Self};
    use token_bridge::state::{Self, State, LatestOnly};
    use token_bridge::token_registry::{Self};

    /// Coin type belongs to a wrapped asset.
    const E_WRAPPED_ASSET: u64 = 0;
    /// Coin type belongs to an untrusted contract from `create_wrapped` which
    /// has not completed registration.
    const E_FROM_CREATE_WRAPPED: u64 = 1;

    /// `attest_token` takes `CoinMetadata` of a coin type and generates a
    /// `MessageTicket` with encoded asset metadata for a foreign Token Bridge
    /// contract to consume and create a wrapped asset reflecting this Sui
    /// asset. Asset metadata is encoded using `AssetMeta`.
    ///
    /// See `token_registry` and `asset_meta` module for more info.
    public fun attest_token<CoinType>(
        token_bridge_state: &mut State,
        coin_meta: &CoinMetadata<CoinType>,
        nonce: u32
    ): MessageTicket {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        // Encode Wormhole message payload.
        let encoded_asset_meta =
            serialize_asset_meta(&latest_only, token_bridge_state, coin_meta);

        // Prepare Wormhole message.
        state::prepare_wormhole_message(
            &latest_only,
            token_bridge_state,
            nonce,
            encoded_asset_meta
        )
    }

    fun serialize_asset_meta<CoinType>(
        latest_only: &LatestOnly,
        token_bridge_state: &mut State,
        coin_meta: &CoinMetadata<CoinType>,
    ): vector<u8> {
        let registry = state::borrow_token_registry(token_bridge_state);

        // Register if it is a new asset.
        //
        // NOTE: We don't want to abort if the asset is already registered
        // because we may want to send asset metadata again after registration
        // (the owner of a particular `CoinType` can change `CoinMetadata` any
        // time after we register the asset).
        if (token_registry::has<CoinType>(registry)) {
            let asset_info = token_registry::verified_asset<CoinType>(registry);
            // If this asset is already registered, there should already
            // be canonical info associated with this coin type.
            assert!(
                !token_registry::is_wrapped(&asset_info),
                E_WRAPPED_ASSET
            );
        } else {
            // Before we consider registering, we should not accidentally
            // perform this registration that may be the `CoinMetadata` from
            // `create_wrapped::prepare_registration`, which has empty fields.
            assert!(
                !create_wrapped::incomplete_metadata(coin_meta),
                E_FROM_CREATE_WRAPPED
            );

            // Now register it.
            token_registry::add_new_native(
                state::borrow_mut_token_registry(
                    latest_only,
                    token_bridge_state
                ),
                coin_meta
            );
        };

        asset_meta::serialize(asset_meta::from_metadata(coin_meta))
    }

    #[test_only]
    public fun serialize_asset_meta_test_only<CoinType>(
        token_bridge_state: &mut State,
        coin_metadata: &CoinMetadata<CoinType>,
    ): vector<u8> {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        serialize_asset_meta(&latest_only, token_bridge_state, coin_metadata)
    }
}

#[test_only]
module token_bridge::attest_token_tests {
    use std::ascii::{Self};
    use std::string::{Self};
    use sui::coin::{Self};
    use sui::test_scenario::{Self};
    use wormhole::publish_message::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::asset_meta::{Self};
    use token_bridge::attest_token::{Self};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::coin_wrapped_7::{Self, COIN_WRAPPED_7};
    use token_bridge::native_asset::{Self};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        person,
        return_state,
        set_up_wormhole_and_token_bridge,
        take_state,
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

        let token_bridge_state = take_state(scenario);
        let coin_meta = coin_native_10::take_metadata(scenario);

        // Emit `AssetMeta` payload.
        let prepared_msg =
            attest_token(
                &mut token_bridge_state,
                &coin_meta,
                1234, // nonce
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        // Check that asset is registered.
        {
            let registry =
                state::borrow_token_registry(&token_bridge_state);
            let verified =
                token_registry::verified_asset<COIN_NATIVE_10>(registry);
            assert!(!token_registry::is_wrapped(&verified), 0);

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

            assert!(native_asset::custody(asset) == 0, 0);
        };

        // Clean up for next call.
        publish_message::destroy(prepared_msg);

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
        let prepared_msg =
            attest_token(
                &mut token_bridge_state,
                &coin_meta,
                1234, // nonce
            );

        // Clean up.
        publish_message::destroy(prepared_msg);
        return_state(token_bridge_state);
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
        let expected_serialized =
            asset_meta::serialize_test_only(
                asset_meta::from_metadata_test_only(&coin_meta)
            );
        assert!(serialized == expected_serialized, 0);

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

        // Check that the new serialization reflects updated metadata.
        let expected_serialized =
            asset_meta::serialize_test_only(
                asset_meta::from_metadata_test_only(&coin_meta)
            );
        assert!(serialized != expected_serialized, 0);
        let updated_serialized =
            serialize_asset_meta_test_only(&mut token_bridge_state, &coin_meta);
        assert!(updated_serialized == expected_serialized, 0);

        // Clean up.
        return_state(token_bridge_state);
        coin_native_10::return_globals(treasury_cap, coin_meta);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = attest_token::E_FROM_CREATE_WRAPPED)]
    fun test_cannot_attest_token_from_create_wrapped() {
        use token_bridge::attest_token::{attest_token};

        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Publish coin.
        coin_wrapped_7::init_test_only(test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);
        let coin_meta = test_scenario::take_shared(scenario);

        // You shall not pass!
        let prepared_msg =
            attest_token<COIN_WRAPPED_7>(
                &mut token_bridge_state,
                &coin_meta,
                1234 // nonce
            );

        // Clean up.
        publish_message::destroy(prepared_msg);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_attest_token_outdated_version() {
        use token_bridge::attest_token::{attest_token};

        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Publish coin.
        coin_wrapped_7::init_test_only(test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Ignore effects.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);
        let coin_meta = test_scenario::take_shared(scenario);

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut token_bridge_state);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        state::migrate_version_test_only(
            &mut token_bridge_state,
            token_bridge::version_control::previous_version_test_only(),
            token_bridge::version_control::next_version()
        );

        // You shall not pass!
        let prepared_msg =
            attest_token<COIN_WRAPPED_7>(
                &mut token_bridge_state,
                &coin_meta,
                1234 // nonce
            );

        // Clean up.
        publish_message::destroy(prepared_msg);

        abort 42
    }
}
