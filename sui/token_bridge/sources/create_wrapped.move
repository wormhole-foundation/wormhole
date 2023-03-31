// SPDX-License-Identifier: Apache 2

/// This module implements methods that create a specific coin type reflecting a
/// wrapped (foreign) asset, whose metadata is encoded in a VAA sent from
/// another network.
///
/// Wrapped assets are created in two steps.
///   1. `prepare_registration`: This method creates a new `Supply` for a given
///      coin type and wraps an encoded asset metadata VAA. We require a one-
///      time witness (OTW) because we only want one `Supply` for a given coin
///      type. This coin will be published using this method, meaning the `init`
///      method in that package will have the asset metadata VAA hard-coded
///      (which is passed into `prepare_registration` as one of its arguments).
///      A `WrappedAssetSetup` object is transferred to the transaction sender.
///   2. `complete_registration`: This method destroys the `WrappedAssetSetup`
///      object by unpacking its members. The encoded asset metadata VAA is
///      deserialized and moved (along with the `Supply`) to the state module
///      to create `ForeignMetadata`.
///
/// Wrapped asset metadata can also be updated with a new asset metadata VAA.
/// By calling `update_attestation`, Token Bridge verifies that the specific
/// coin type is registered and agrees with the encoded asset metadata's
/// canonical token info. `ForeignMetadata` will be updated based on the encoded
/// asset metadata payload.
///
/// See `state` and `wrapped_asset` modules for more details.
///
/// References:
/// https://examples.sui.io/basics/one-time-witness.html
module token_bridge::create_wrapped {
    use std::ascii::{Self};
    use std::type_name::{Self};
    use sui::balance::{Self, Supply};
    use sui::clock::{Clock};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::state::{State as WormholeState};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::state::{Self, State};
    use token_bridge::token_registry::{Self};
    use token_bridge::vaa::{Self};
    use token_bridge::version_control::{
        Self as control,
        CreateWrapped as CreateWrappedControl
    };

    /// Asset metadata is for native Sui coin type.
    const E_NATIVE_ASSET: u64 = 0;
    /// Asset metadata has not been registered yet.
    const E_UNREGISTERED_FOREIGN_ASSET: u64 = 1;
    /// Failed one-time witness verification.
    const E_BAD_WITNESS: u64 = 2;
    /// Coin witness does not equal "COIN".
    const E_INVALID_COIN_MODULE_NAME: u64 = 3;

    /// A.K.A. "coin".
    const COIN_MODULE_NAME: vector<u8> = b"coin";

    /// Container holding new coin type's `Supply` and encoded asset metadata
    /// VAA, which are required to complete this asset's registration.
    struct WrappedAssetSetup<phantom CoinType> has key, store {
        id: UID,
        vaa_buf: vector<u8>,
        supply: Supply<CoinType>,
        build_version: u64
    }

    /// This method is executed within the `init` method of an untrusted module,
    /// which defines a one-time witness (OTW) type (`CoinType`). OTW is
    /// required to ensure that only one `Supply` exists for `CoinType`. This
    /// is similar to how a `TreasuryCap` is created in `coin::create_currency`.
    ///
    /// Because this method is stateless (i.e. no dependency on Token Bridge's
    /// `State` object), the contract defers VAA verification to
    /// `complete_registration` after this method has been executed.
    public fun prepare_registration<CoinType: drop>(
        witness: CoinType,
        vaa_buf: vector<u8>,
        ctx: &mut TxContext
    ): WrappedAssetSetup<CoinType> {
        // Make sure there's only one instance of the type `CoinType`. This
        // resembles the same check for `coin::create_currency`.
        assert!(sui::types::is_one_time_witness(&witness), E_BAD_WITNESS);

        // Also make sure that this witness module name is literally "coin".
        let module_name = type_name::get_module(&type_name::get<CoinType>());
        assert!(
            ascii::into_bytes(module_name) == COIN_MODULE_NAME,
            E_INVALID_COIN_MODULE_NAME
        );

        // Create `WrappedAssetSetup` object and transfer to transaction sender.
        // The owner of this object will call `complete_registration` to destroy
        // it.
        new_setup(witness, vaa_buf, ctx)
    }

    /// After executing `prepare_registration`, owner of `WrappedAssetSetup`
    /// executes this method to complete this wrapped asset's registration.
    ///
    /// This method destroys `WrappedAssetSetup`, unpacking the `Supply` and
    /// encoded asset metadata VAA. The deserialized asset metadata VAA is used
    /// to create `ForeignMetadata`.
    ///
    /// TODO: Maybe add `UpgradeCap` argument (which would come from the
    /// `CoinType` package so we can either destroy it or warehouse it in
    /// `WrappedAsset`).
    public fun complete_registration<CoinType: drop>(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        setup: WrappedAssetSetup<CoinType>,
        the_clock: &Clock,
        ctx: &mut TxContext,
    ) {
        state::check_minimum_requirement<CreateWrappedControl>(
            token_bridge_state
        );

        let WrappedAssetSetup {
            id,
            vaa_buf,
            supply,
            build_version
        } = setup;

        // Do an additional check of whether `WrappedAssetSetup` was created
        // using the minimum required version for this module.
        state::check_minimum_requirement_specified<CreateWrappedControl>(
            token_bridge_state,
            build_version
        );

        // Finally destroy the object.
        object::delete(id);

        // Deserialize to `AssetMeta`.
        let token_meta =
            parse_and_verify_asset_meta(
                token_bridge_state,
                worm_state,
                vaa_buf,
                the_clock
            );

        // `register_wrapped_asset` uses `token_registry::add_new_wrapped`,
        // which will check whether the asset has already been registered and if
        // the token chain ID is not Sui's.
        //
        // If both of these conditions are met, `register_wrapped_asset` will
        // succeed and the new wrapped coin will be registered.
        token_registry::add_new_wrapped(
            state::borrow_token_registry_mut(token_bridge_state),
            token_meta,
            supply,
            ctx
        );
    }

    /// For registered wrapped assets, we can update `ForeignMetadata` for a
    /// given `CoinType` with a new asset meta VAA emitted from another network.
    public fun update_attestation<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        the_clock: &Clock
    ) {
        state::check_minimum_requirement<CreateWrappedControl>(
            token_bridge_state
        );

        // Deserialize to `AssetMeta`.
        let token_meta =
            parse_and_verify_asset_meta(
                token_bridge_state,
                worm_state,
                vaa_buf,
                the_clock
            );

        // When a wrapped asset is updated, the encoded token info is checked
        // against what exists in the registry.
        token_registry::update_wrapped<CoinType>(
            state::borrow_token_registry_mut(token_bridge_state),
            token_meta
        );
    }

    fun parse_and_verify_asset_meta(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        the_clock: &Clock
    ): AssetMeta {
        let parsed =
            vaa::parse_verify_and_consume(
                token_bridge_state,
                worm_state,
                vaa_buf,
                the_clock
            );

        // Finally deserialize the VAA payload.
        asset_meta::deserialize(wormhole::vaa::take_payload(parsed))
    }

    fun new_setup<CoinType: drop>(
        witness: CoinType,
        vaa_buf: vector<u8>,
        ctx: &mut TxContext
    ): WrappedAssetSetup<CoinType> {
       WrappedAssetSetup {
            id: object::new(ctx),
            vaa_buf,
            supply: balance::create_supply(witness),
            build_version: control::version()
        }
    }

    #[test_only]
    public fun new_setup_test_only<CoinType: drop>(
        witness: CoinType,
        vaa_buf: vector<u8>,
        ctx: &mut TxContext
    ): WrappedAssetSetup<CoinType> {
        new_setup(witness, vaa_buf, ctx)
    }

    #[test_only]
    public fun take_supply<CoinType>(
        setup: WrappedAssetSetup<CoinType>
    ): Supply<CoinType> {
        let WrappedAssetSetup {
            id,
            vaa_buf: _,
            supply,
            build_version: _
        } = setup;
        object::delete(id);

        supply
    }
}

#[test_only]
module token_bridge::create_wrapped_tests {
    use sui::test_scenario::{Self};
    use sui::test_utils::{Self};
    use sui::tx_context::{Self};

    use token_bridge::asset_meta::{Self};
    use token_bridge::coin_wrapped_12::{Self};
    use token_bridge::coin_wrapped_7::{Self};
    use token_bridge::create_wrapped::{Self};
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

    struct NOT_A_WITNESS has drop {}

    struct CREATE_WRAPPED_TESTS has drop {}

    #[test]
    #[expected_failure(abort_code = create_wrapped::E_BAD_WITNESS)]
    public fun test_cannot_prepare_registration_bad_witness() {
        let ctx = &mut tx_context::dummy();

        // You shall not pass!
        let wrapped_asset_setup =
            create_wrapped::prepare_registration(
                NOT_A_WITNESS {},
                coin_wrapped_12::encoded_vaa(),
                ctx
            );

        // Clean up.
        test_utils::destroy(wrapped_asset_setup);
    }

    #[test]
    #[expected_failure(abort_code = create_wrapped::E_INVALID_COIN_MODULE_NAME)]
    public fun test_cannot_prepare_registration_invalid_coin_module_name() {
        let ctx = &mut tx_context::dummy();

        // You shall not pass!
        let wrapped_asset_setup =
            create_wrapped::prepare_registration(
                CREATE_WRAPPED_TESTS {},
                coin_wrapped_12::encoded_vaa(),
                ctx
            );

        // Clean up.
        test_utils::destroy(wrapped_asset_setup);
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
        let wrapped_asset_setup =
            create_wrapped::new_setup_test_only(
                CREATE_WRAPPED_TESTS {},
                coin_wrapped_12::encoded_vaa(),
                test_scenario::ctx(scenario)
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
            let registry = state::borrow_token_registry(&token_bridge_state);
            assert!(token_registry::is_wrapped<CREATE_WRAPPED_TESTS>(registry), 0);

            let asset =
                token_registry::borrow_wrapped<CREATE_WRAPPED_TESTS>(registry);
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
        create_wrapped::update_attestation<CREATE_WRAPPED_TESTS>(
            &mut token_bridge_state,
            &worm_state,
            coin_wrapped_12::encoded_updated_vaa(),
            &the_clock
        );

        // Check updated name and symbol.
        let registry = state::borrow_token_registry(&token_bridge_state);
        let asset = token_registry::borrow_wrapped<CREATE_WRAPPED_TESTS>(registry);
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

    #[test]
    #[expected_failure(abort_code = wrapped_asset::E_ASSET_META_MISMATCH)]
    public fun test_cannot_update_attestation_wrong_canonical_info() {
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
        let wrapped_asset_setup =
            create_wrapped::new_setup_test_only(
                CREATE_WRAPPED_TESTS {},
                coin_wrapped_12::encoded_vaa(),
                test_scenario::ctx(scenario)
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
        // This VAA is for COIN_WRAPPED_7 metadata, which disagrees with
        // COIN_WRAPPED_12.
        let invalid_asset_meta_vaa = coin_wrapped_7::encoded_vaa();

        // You shall not pass!
        create_wrapped::update_attestation<CREATE_WRAPPED_TESTS>(
            &mut token_bridge_state,
            &worm_state,
            invalid_asset_meta_vaa,
            &the_clock
        );

        // Clean up.
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }
}
