// SPDX-License-Identifier: Apache 2

/// This module implements methods that create a specific coin type reflecting a
/// wrapped (foreign) asset, whose metadata is encoded in a VAA sent from
/// another network.
///
/// Wrapped assets are created in two steps.
///   1. `prepare_registration`: This method creates a new `TreasuryCap` for a
///      given coin type and wraps an encoded asset metadata VAA. We require a
///      one-time witness (OTW) to throw an explicit error (even though it is
///      redundant with what `create_currency` requires). This coin will
///      be published using this method, meaning the `init` method in that
///      untrusted package will have the asset's decimals hard-coded for its
///      coin metadata. A `WrappedAssetSetup` object is transferred to the
///      transaction sender.
///   2. `complete_registration`: This method destroys the `WrappedAssetSetup`
///      object by unpacking its `TreasuryCap`, which will be warehoused in the
///      `TokenRegistry`. The shared coin metadata object will be updated to
///      reflect the contents of the encoded asset metadata payload.
///
/// Wrapped asset metadata can also be updated with a new asset metadata VAA.
/// By calling `update_attestation`, Token Bridge verifies that the specific
/// coin type is registered and agrees with the encoded asset metadata's
/// canonical token info. `ForeignInfo` and the coin's metadata will be updated
/// based on the encoded asset metadata payload.
///
/// See `state` and `wrapped_asset` modules for more details.
///
/// References:
/// https://examples.sui.io/basics/one-time-witness.html
module token_bridge::create_wrapped {
    use std::ascii::{Self};
    use std::option::{Self};
    use std::type_name::{Self};
    use sui::coin::{Self, TreasuryCap, CoinMetadata};
    use sui::object::{Self, UID};
    use sui::package::{UpgradeCap};
    use sui::transfer::{Self};
    use sui::tx_context::{TxContext};

    use token_bridge::asset_meta::{Self};
    use token_bridge::normalized_amount::{max_decimals};
    use token_bridge::state::{Self, State};
    use token_bridge::token_registry::{Self};
    use token_bridge::vaa::{Self, TokenBridgeMessage};
    use token_bridge::wrapped_asset::{Self};

    #[test_only]
    use token_bridge::version_control::{Self, V__0_2_0 as V__CURRENT};

    /// Failed one-time witness verification.
    const E_BAD_WITNESS: u64 = 0;
    /// Coin witness does not equal "COIN".
    const E_INVALID_COIN_MODULE_NAME: u64 = 1;
    /// Decimals value exceeds `MAX_DECIMALS` from `normalized_amount`.
    const E_DECIMALS_EXCEED_WRAPPED_MAX: u64 = 2;

    /// A.K.A. "coin".
    const COIN_MODULE_NAME: vector<u8> = b"coin";

    /// Container holding new coin type's `TreasuryCap` and encoded asset metadata
    /// VAA, which are required to complete this asset's registration.
    struct WrappedAssetSetup<phantom CoinType, phantom Version> has key, store {
        id: UID,
        treasury_cap: TreasuryCap<CoinType>
    }

    /// This method is executed within the `init` method of an untrusted module,
    /// which defines a one-time witness (OTW) type (`CoinType`). OTW is
    /// required to ensure that only one `TreasuryCap` exists for `CoinType`. This
    /// is similar to how a `TreasuryCap` is created in `coin::create_currency`.
    ///
    /// Because this method is stateless (i.e. no dependency on Token Bridge's
    /// `State` object), the contract defers VAA verification to
    /// `complete_registration` after this method has been executed.
    public fun prepare_registration<CoinType: drop, Version>(
        witness: CoinType,
        decimals: u8,
        ctx: &mut TxContext
    ): WrappedAssetSetup<CoinType, Version> {
        let setup = prepare_registration_internal(witness, decimals, ctx);

        // Also make sure that this witness module name is literally "coin".
        let module_name = type_name::get_module(&type_name::get<CoinType>());
        assert!(
            ascii::into_bytes(module_name) == COIN_MODULE_NAME,
            E_INVALID_COIN_MODULE_NAME
        );

        setup
    }

    #[allow(lint(share_owned))]
    /// This function performs the bulk of `prepare_registration`, except
    /// checking the module name. This separation is useful for testing.
    fun prepare_registration_internal<CoinType: drop, Version>(
        witness: CoinType,
        decimals: u8,
        ctx: &mut TxContext
    ): WrappedAssetSetup<CoinType, Version> {
        // Make sure there's only one instance of the type `CoinType`. This
        // resembles the same check for `coin::create_currency`.
        // Technically this check is redundant as it's performed by
        // `coin::create_currency` below, but it doesn't hurt.
        assert!(sui::types::is_one_time_witness(&witness), E_BAD_WITNESS);

        // Ensure that the decimals passed into this method do not exceed max
        // decimals (see `normalized_amount` module).
        assert!(decimals <= max_decimals(), E_DECIMALS_EXCEED_WRAPPED_MAX);

        // We initialise the currency with empty metadata. Later on, in the
        // `complete_registration` call, when `CoinType` gets associated with a
        // VAA, we update these fields.
        let no_symbol = b"";
        let no_name = b"";
        let no_description = b"";
        let no_icon_url = option::none();

        let (treasury_cap, coin_meta) =
            coin::create_currency(
                witness,
                decimals,
                no_symbol,
                no_name,
                no_description,
                no_icon_url,
                ctx
            );

        // The CoinMetadata is turned into a shared object so that other
        // functions (and wallets) can easily grab references to it. This is
        // safe to do, as the metadata setters require a `TreasuryCap` for the
        // coin too, which is held by the token bridge.
        transfer::public_share_object(coin_meta);

        // Create `WrappedAssetSetup` object and transfer to transaction sender.
        // The owner of this object will call `complete_registration` to destroy
        // it.
        WrappedAssetSetup {
            id: object::new(ctx),
            treasury_cap
        }
    }

    /// After executing `prepare_registration`, owner of `WrappedAssetSetup`
    /// executes this method to complete this wrapped asset's registration.
    ///
    /// This method destroys `WrappedAssetSetup`, unpacking the `TreasuryCap` and
    /// encoded asset metadata VAA. The deserialized asset metadata VAA is used
    /// to update the associated `CoinMetadata`.
    public fun complete_registration<CoinType: drop, Version>(
        token_bridge_state: &mut State,
        coin_meta: &mut CoinMetadata<CoinType>,
        setup: WrappedAssetSetup<CoinType, Version>,
        coin_upgrade_cap: UpgradeCap,
        msg: TokenBridgeMessage
    ) {
        // This capability ensures that the current build version is used. This
        // call performs an additional check of whether `WrappedAssetSetup` was
        // created using the current package.
        let latest_only =
            state::assert_latest_only_specified<Version>(token_bridge_state);

        let WrappedAssetSetup {
            id,
            treasury_cap
        } = setup;

        // Finally destroy the object.
        object::delete(id);

        // Deserialize to `AssetMeta`.
        let token_meta = asset_meta::deserialize(vaa::take_payload(msg));

        // `register_wrapped_asset` uses `token_registry::add_new_wrapped`,
        // which will check whether the asset has already been registered and if
        // the token chain ID is not Sui's.
        //
        // If both of these conditions are met, `register_wrapped_asset` will
        // succeed and the new wrapped coin will be registered.
        token_registry::add_new_wrapped(
            state::borrow_mut_token_registry(&latest_only, token_bridge_state),
            token_meta,
            coin_meta,
            treasury_cap,
            coin_upgrade_cap
        );
    }

    /// For registered wrapped assets, we can update `ForeignInfo` for a
    /// given `CoinType` with a new asset meta VAA emitted from another network.
    public fun update_attestation<CoinType>(
        token_bridge_state: &mut State,
        coin_meta: &mut CoinMetadata<CoinType>,
        msg: TokenBridgeMessage
    ) {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        // Deserialize to `AssetMeta`.
        let token_meta = asset_meta::deserialize(vaa::take_payload(msg));

        // This asset must exist in the registry.
        let registry =
            state::borrow_mut_token_registry(&latest_only, token_bridge_state);
        token_registry::assert_has<CoinType>(registry);

        // Now update wrapped.
        wrapped_asset::update_metadata(
            token_registry::borrow_mut_wrapped<CoinType>(registry),
            coin_meta,
            token_meta
        );
    }

    public fun incomplete_metadata<CoinType>(
        coin_meta: &CoinMetadata<CoinType>
    ): bool {
        use std::string::{bytes};
        use std::vector::{is_empty};

        (
            is_empty(ascii::as_bytes(&coin::get_symbol(coin_meta))) &&
            is_empty(bytes(&coin::get_name(coin_meta))) &&
            is_empty(bytes(&coin::get_description(coin_meta))) &&
            std::option::is_none(&coin::get_icon_url(coin_meta))
        )
    }

    #[test_only]
    public fun new_setup_test_only<CoinType: drop, Version: drop>(
        _version: Version,
        witness: CoinType,
        decimals: u8,
        ctx: &mut TxContext
    ): (WrappedAssetSetup<CoinType, Version>, UpgradeCap) {
        let setup =
            prepare_registration_internal(
                witness,
                decimals,
                ctx
            );

        let upgrade_cap =
            sui::package::test_publish(
                object::id_from_address(@token_bridge),
                ctx
            );

        (setup, upgrade_cap)
    }

    #[test_only]
    public fun new_setup_current<CoinType: drop>(
        witness: CoinType,
        decimals: u8,
        ctx: &mut TxContext
    ): (WrappedAssetSetup<CoinType, V__CURRENT>, UpgradeCap) {
        new_setup_test_only(
            version_control::current_version_test_only(),
            witness,
            decimals,
            ctx
        )
    }

    #[test_only]
    public fun take_treasury_cap<CoinType>(
        setup: WrappedAssetSetup<CoinType, V__CURRENT>
    ): TreasuryCap<CoinType> {
        let WrappedAssetSetup {
            id,
            treasury_cap
        } = setup;
        object::delete(id);

        treasury_cap
    }
}

#[test_only]
module token_bridge::create_wrapped_tests {
    use sui::coin::{Self};
    use sui::test_scenario::{Self};
    use sui::test_utils::{Self};
    use sui::tx_context::{Self};
    use wormhole::wormhole_scenario::{parse_and_verify_vaa};

    use token_bridge::asset_meta::{Self};
    use token_bridge::coin_wrapped_12::{Self};
    use token_bridge::coin_wrapped_7::{Self};
    use token_bridge::create_wrapped::{Self};
    use token_bridge::state::{Self};
    use token_bridge::string_utils::{Self};
    use token_bridge::token_bridge_scenario::{
        register_dummy_emitter,
        return_state,
        set_up_wormhole_and_token_bridge,
        take_state,
        two_people
    };
    use token_bridge::token_registry::{Self};
    use token_bridge::vaa::{Self};
    use token_bridge::version_control::{V__0_2_0 as V__CURRENT};
    use token_bridge::wrapped_asset::{Self};

    struct NOT_A_WITNESS has drop {}

    struct CREATE_WRAPPED_TESTS has drop {}

    #[test]
    #[expected_failure(abort_code = create_wrapped::E_BAD_WITNESS)]
    fun test_cannot_prepare_registration_bad_witness() {
        let ctx = &mut tx_context::dummy();

        // You shall not pass!
        let wrapped_asset_setup =
            create_wrapped::prepare_registration<NOT_A_WITNESS, V__CURRENT>(
                NOT_A_WITNESS {},
                3,
                ctx
            );

        // Clean up.
        test_utils::destroy(wrapped_asset_setup);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = create_wrapped::E_INVALID_COIN_MODULE_NAME)]
    fun test_cannot_prepare_registration_invalid_coin_module_name() {
        let ctx = &mut tx_context::dummy();

        // You shall not pass!
        let wrapped_asset_setup =
            create_wrapped::prepare_registration<
                CREATE_WRAPPED_TESTS,
                V__CURRENT
            >(
                CREATE_WRAPPED_TESTS {},
                3,
                ctx
            );

        // Clean up.
        test_utils::destroy(wrapped_asset_setup);

        abort 42
    }

    #[test]
    fun test_complete_and_update_attestation() {
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
        let (
            wrapped_asset_setup,
            upgrade_cap
        ) =
            create_wrapped::new_setup_current(
                CREATE_WRAPPED_TESTS {},
                8,
                test_scenario::ctx(scenario)
            );

        let token_bridge_state = take_state(scenario);

        let verified_vaa =
            parse_and_verify_vaa(scenario, coin_wrapped_12::encoded_vaa());
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        let coin_meta = test_scenario::take_shared(scenario);

        create_wrapped::complete_registration(
            &mut token_bridge_state,
            &mut coin_meta,
            wrapped_asset_setup,
            upgrade_cap,
            msg
        );

        let (
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        ) = asset_meta::unpack_test_only(coin_wrapped_12::token_meta());

        // Check registry.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let verified =
                token_registry::verified_asset<CREATE_WRAPPED_TESTS>(registry);
            assert!(token_registry::is_wrapped(&verified), 0);

            let asset =
                token_registry::borrow_wrapped<CREATE_WRAPPED_TESTS>(registry);
            assert!(wrapped_asset::total_supply(asset) == 0, 0);

            // Decimals are capped for this wrapped asset.
            assert!(coin::get_decimals(&coin_meta) == 8, 0);

            // Check metadata against asset metadata.
            let info = wrapped_asset::info(asset);
            assert!(wrapped_asset::token_chain(info) == token_chain, 0);
            assert!(wrapped_asset::token_address(info) == token_address, 0);
            assert!(
                wrapped_asset::native_decimals(info) == native_decimals,
                0
            );
            assert!(coin::get_symbol(&coin_meta) == string_utils::to_ascii(&symbol), 0);
            assert!(coin::get_name(&coin_meta) == name, 0);
        };


        // Now update metadata.
        let verified_vaa =
            parse_and_verify_vaa(
                scenario,
                coin_wrapped_12::encoded_updated_vaa()
            );
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);
        create_wrapped::update_attestation<CREATE_WRAPPED_TESTS>(
            &mut token_bridge_state,
            &mut coin_meta,
            msg
        );

        // Check updated name and symbol.
        let (
            _,
            _,
            _,
            new_symbol,
            new_name
        ) = asset_meta::unpack_test_only(coin_wrapped_12::updated_token_meta());

        assert!(symbol != new_symbol, 0);

        assert!(coin::get_symbol(&coin_meta) == string_utils::to_ascii(&new_symbol), 0);

        assert!(name != new_name, 0);
        assert!(coin::get_name(&coin_meta) == new_name, 0);

        test_scenario::return_shared(coin_meta);

        // Clean up.
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wrapped_asset::E_ASSET_META_MISMATCH)]
    fun test_cannot_update_attestation_wrong_canonical_info() {
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
        let (
            wrapped_asset_setup,
            upgrade_cap
        ) =
            create_wrapped::new_setup_current(
                CREATE_WRAPPED_TESTS {},
                8,
                test_scenario::ctx(scenario)
            );

        let token_bridge_state = take_state(scenario);

        let verified_vaa =
            parse_and_verify_vaa(scenario, coin_wrapped_12::encoded_vaa());
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        let coin_meta = test_scenario::take_shared(scenario);

        create_wrapped::complete_registration(
            &mut token_bridge_state,
            &mut coin_meta,
            wrapped_asset_setup,
            upgrade_cap,
            msg
        );
        // This VAA is for COIN_WRAPPED_7 metadata, which disagrees with
        // COIN_WRAPPED_12.
        let invalid_asset_meta_vaa = coin_wrapped_7::encoded_vaa();

        let verified_vaa =
            parse_and_verify_vaa(scenario, invalid_asset_meta_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);
        // You shall not pass!
        create_wrapped::update_attestation<CREATE_WRAPPED_TESTS>(
            &mut token_bridge_state,
            &mut coin_meta,
            msg
        );

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = state::E_VERSION_MISMATCH)]
    fun test_cannot_complete_registration_version_mismatch() {
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
        let (
            wrapped_asset_setup,
            upgrade_cap
        ) =
            create_wrapped::new_setup_test_only(
                token_bridge::version_control::dummy(),
                CREATE_WRAPPED_TESTS {},
                8,
                test_scenario::ctx(scenario)
            );

        let token_bridge_state = take_state(scenario);

        let verified_vaa =
            parse_and_verify_vaa(scenario, coin_wrapped_12::encoded_vaa());
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        let coin_meta = test_scenario::take_shared(scenario);

        create_wrapped::complete_registration(
            &mut token_bridge_state,
            &mut coin_meta,
            wrapped_asset_setup,
            upgrade_cap,
            msg
        );

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_complete_registration_outdated_version() {
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
        let (
            wrapped_asset_setup,
            upgrade_cap
        ) =
            create_wrapped::new_setup_current(
                CREATE_WRAPPED_TESTS {},
                8,
                test_scenario::ctx(scenario)
            );

        let token_bridge_state = take_state(scenario);

        let verified_vaa =
            parse_and_verify_vaa(scenario, coin_wrapped_12::encoded_vaa());
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

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
        create_wrapped::complete_registration(
            &mut token_bridge_state,
            &mut coin_meta,
            wrapped_asset_setup,
            upgrade_cap,
            msg
        );

        abort 42
    }
}
