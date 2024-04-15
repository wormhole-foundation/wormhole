// SPDX-License-Identifier: Apache 2

/// This module implements two methods: `authorize_transfer` and `redeem_coin`,
/// which are to be executed in a transaction block in this order.
///
/// `authorize_transfer` allows a contract to complete a Token Bridge transfer
/// with arbitrary payload. This deserialized `TransferWithPayload` with the
/// bridged balance and source chain ID are packaged in a `RedeemerReceipt`.
///
/// `redeem_coin` unpacks the `RedeemerReceipt` and checks whether the specified
/// `EmitterCap` is the specified redeemer for this transfer. If he is the
/// correct redeemer, the balance is unpacked and transformed into `Coin` and
/// is returned alongside `TransferWithPayload` and source chain ID.
///
/// The purpose of splitting this transfer redemption into two steps is in case
/// Token Bridge needs to be upgraded and there is a breaking change for this
/// module, an integrator would not be left broken. It is discouraged to put
/// `authorize_transfer` in an integrator's package logic. Otherwise, this
/// integrator needs to be prepared to upgrade his contract to handle the latest
/// version of `complete_transfer_with_payload`.
///
/// Instead, an integrator is encouraged to execute a transaction block, which
/// executes `authorize_transfer` using the latest Token Bridge package ID and
/// to implement `redeem_coin` in his contract to consume this receipt. This is
/// similar to how an integrator with Wormhole is not meant to use
/// `vaa::parse_and_verify` in his contract in case the `vaa` module needs to
/// be upgraded due to a breaking change.
///
/// Like in `complete_transfer`, a VAA with an encoded transfer can be redeemed
/// only once.
///
/// See `transfer_with_payload` module for serialization and deserialization of
/// Wormhole message payload.
module token_bridge::complete_transfer_with_payload {
    use sui::coin::{Self, Coin};
    use sui::object::{Self};
    use sui::tx_context::{TxContext};
    use wormhole::emitter::{EmitterCap};

    use token_bridge::complete_transfer::{Self};
    use token_bridge::state::{Self, State, LatestOnly};
    use token_bridge::transfer_with_payload::{Self, TransferWithPayload};
    use token_bridge::vaa::{Self, TokenBridgeMessage};

    /// `EmitterCap` address does not agree with encoded redeemer.
    const E_INVALID_REDEEMER: u64 = 0;

    #[allow(lint(coin_field))]
    /// This type is only generated from `authorize_transfer` and can only be
    /// redeemed using `redeem_coin`. Integrators are expected to implement
    /// `redeem_coin` within their contracts and call `authorize_transfer` in a
    /// transaction block preceding the method that consumes this receipt. The
    /// only way to destroy this receipt is calling `redeem_coin` with an
    /// `EmitterCap` generated from the `wormhole::emitter` module, whose ID is
    /// the expected redeemer for this token transfer.
    struct RedeemerReceipt<phantom CoinType> {
        /// Which chain ID this transfer originated from.
        source_chain: u16,
        /// Deserialized transfer info.
        parsed: TransferWithPayload,
        /// Coin of bridged asset.
        bridged_out: Coin<CoinType>
    }

    /// `authorize_transfer` deserializes a token transfer VAA payload, which
    /// encodes its own arbitrary payload (which has meaning to the redeemer).
    /// Once the transfer is authorized, an event (`TransferRedeemed`) is
    /// emitted to reflect which Token Bridge this transfer originated from.
    /// The `RedeemerReceipt` returned wraps a balance reflecting the encoded
    /// transfer amount along with the source chain and deserialized
    /// `TransferWithPayload`.
    ///
    /// NOTE: This method is guarded by a minimum build version check. This
    /// method could break backward compatibility on an upgrade.
    ///
    /// It is important for integrators to refrain from calling this method
    /// within their contracts. This method is meant to be called in a
    /// transaction block, passing the `RedeemerReceipt` to a method which calls
    /// `redeem_coin` within a contract. If in a circumstance where this module
    /// has a breaking change in an upgrade, `redeem_coin` will not be affected
    /// by this change.
    ///
    /// See `redeem_coin` for more details.
    public fun authorize_transfer<CoinType>(
        token_bridge_state: &mut State,
        msg: TokenBridgeMessage,
        ctx: &mut TxContext
    ): RedeemerReceipt<CoinType> {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        // Emitting the transfer being redeemed.
        //
        // NOTE: We save the emitter chain ID to save the integrator from
        // having to `parse_and_verify` the same encoded VAA to get this info.
        let source_chain =
            complete_transfer::emit_transfer_redeemed(&msg);

        // Finally deserialize the Wormhole message payload and handle bridging
        // out token of a given coin type.
        handle_authorize_transfer(
            &latest_only,
            token_bridge_state,
            source_chain,
            vaa::take_payload(msg),
            ctx
        )
    }

    /// After a transfer is authorized, only a valid redeemer may unpack the
    /// `RedeemerReceipt`. The specified `EmitterCap` is the only authorized
    /// redeemer of the transfer. Once the redeemer is validated, coin from
    /// this receipt of the specified coin type is returned alongside the
    /// deserialized `TransferWithPayload` and source chain ID.
    ///
    /// NOTE: Integrators of Token Bridge redeeming these token transfers should
    /// be calling only this method from their contracts. This method is not
    /// guarded by version control (thus not requiring a reference to the
    /// Token Bridge `State` object), so it is intended to work for any package
    /// version.
    public fun redeem_coin<CoinType>(
        emitter_cap: &EmitterCap,
        receipt: RedeemerReceipt<CoinType>
    ): (
        Coin<CoinType>,
        TransferWithPayload,
        u16 // `wormhole::vaa::emitter_chain`
    ) {
        let RedeemerReceipt { source_chain, parsed, bridged_out } = receipt;

        // Transfer must be redeemed by the contract's registered Wormhole
        // emitter.
        let redeemer = transfer_with_payload::redeemer_id(&parsed);
        assert!(redeemer == object::id(emitter_cap), E_INVALID_REDEEMER);

        // Create coin from balance and return other unpacked members of receipt.
        (bridged_out, parsed, source_chain)
    }

    fun handle_authorize_transfer<CoinType>(
        latest_only: &LatestOnly,
        token_bridge_state: &mut State,
        source_chain: u16,
        transfer_vaa_payload: vector<u8>,
        ctx: &mut TxContext
    ): RedeemerReceipt<CoinType> {
        // Deserialize for processing.
        let parsed = transfer_with_payload::deserialize(transfer_vaa_payload);

        // Handle bridging assets out to be returned to method caller.
        //
        // See `complete_transfer` module for more info.
        let (
            _,
            bridged_out,
        ) =
            complete_transfer::verify_and_bridge_out(
                latest_only,
                token_bridge_state,
                transfer_with_payload::token_chain(&parsed),
                transfer_with_payload::token_address(&parsed),
                transfer_with_payload::redeemer_chain(&parsed),
                transfer_with_payload::amount(&parsed)
            );

        RedeemerReceipt {
            source_chain,
            parsed,
            bridged_out: coin::from_balance(bridged_out, ctx)
        }
    }

    #[test_only]
    public fun burn<CoinType>(receipt: RedeemerReceipt<CoinType>) {
        let RedeemerReceipt {
            source_chain: _,
            parsed: _,
            bridged_out
        } = receipt;
        coin::burn_for_testing(bridged_out);
    }
}

#[test_only]
module token_bridge::complete_transfer_with_payload_tests {
    use sui::coin::{Self};
    use sui::object::{Self};
    use sui::test_scenario::{Self};
    use wormhole::emitter::{Self};
    use wormhole::state::{chain_id};
    use wormhole::wormhole_scenario::{new_emitter, parse_and_verify_vaa};

    use token_bridge::coin_wrapped_12::{Self, COIN_WRAPPED_12};
    use token_bridge::complete_transfer_with_payload::{Self};
    use token_bridge::complete_transfer::{Self};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::dummy_message::{Self};
    use token_bridge::native_asset::{Self};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        register_dummy_emitter,
        return_state,
        set_up_wormhole_and_token_bridge,
        take_state,
        two_people
    };
    use token_bridge::token_registry::{Self};
    use token_bridge::transfer_with_payload::{Self};
    use token_bridge::vaa::{Self};
    use token_bridge::wrapped_asset::{Self};

    #[test]
    /// Test the public-facing function authorize_transfer.
    /// using a native transfer VAA_ATTESTED_DECIMALS_10.
    fun test_complete_transfer_with_payload_native_asset() {
        use token_bridge::complete_transfer_with_payload::{
            authorize_transfer,
            redeem_coin
        };

        let transfer_vaa =
            dummy_message::encoded_transfer_with_payload_vaa_native();

        let (user, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register Sui as a foreign emitter.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Initialize native token.
        let mint_amount = 1000000;
        coin_native_10::init_register_and_deposit(
            scenario,
            coin_deployer,
            mint_amount
        );

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);

        {
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(
                state::borrow_token_registry(&token_bridge_state)
            );
            assert!(native_asset::custody(asset) == mint_amount, 0);
        };

        // Set up dummy `EmitterCap` as the expected redeemer.
        let emitter_cap = emitter::dummy();

        // Verify that the emitter cap is the expected redeemer.
        let expected_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    parse_and_verify_vaa(scenario, transfer_vaa)
                )
            );
        assert!(
            transfer_with_payload::redeemer_id(&expected_transfer) == object::id(&emitter_cap),
            0
        );

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        // Execute authorize_transfer.
        let receipt =
            authorize_transfer<COIN_NATIVE_10>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );
        let (
            bridged,
            parsed_transfer,
            source_chain
        ) = redeem_coin(&emitter_cap, receipt);

        assert!(source_chain == expected_source_chain, 0);

        // Assert coin value, source chain, and parsed transfer details are correct.
        // We expect the coin value to be 300000, because that's in terms of
        // 10 decimals. The amount specified in the VAA_ATTESTED_DECIMALS_12 is 3000, because that's
        // in terms of 8 decimals.
        let expected_bridged = 300000;
        assert!(coin::value(&bridged) == expected_bridged, 0);

        // Amount left on custody should be whatever is left remaining after
        // the transfer.
        let remaining = mint_amount - expected_bridged;
        {
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(
                state::borrow_token_registry(&token_bridge_state)
            );
            assert!(native_asset::custody(asset) == remaining, 0);
        };

        // Verify token info.
        let registry = state::borrow_token_registry(&token_bridge_state);
        let verified =
            token_registry::verified_asset<COIN_NATIVE_10>(registry);
        let expected_token_chain = token_registry::token_chain(&verified);
        let expected_token_address = token_registry::token_address(&verified);
        assert!(expected_token_chain == chain_id(), 0);
        assert!(
            transfer_with_payload::token_chain(&parsed_transfer) == expected_token_chain,
            0
        );
        assert!(
            transfer_with_payload::token_address(&parsed_transfer) == expected_token_address,
            0
        );

        // Verify transfer by serializing both parsed and expected.
        let serialized = transfer_with_payload::serialize(parsed_transfer);
        let expected_serialized =
            transfer_with_payload::serialize(expected_transfer);
        assert!(serialized == expected_serialized, 0);

        // Clean up.
        return_state(token_bridge_state);
        coin::burn_for_testing(bridged);
        emitter::destroy_test_only(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    /// Test the public-facing functions `authorize_transfer` and `redeem_coin`.
    /// Use an actual devnet Wormhole complete transfer with payload
    /// VAA_ATTESTED_DECIMALS_12.
    ///
    /// This test confirms that:
    ///   - `authorize_transfer` with `redeem_coin` deserializes the encoded
    ///      transfer and recovers the source chain, payload, and additional
    ///      transfer details wrapped in a redeemer receipt.
    ///   - a wrapped coin with the correct value is minted by the bridge
    ///     and returned by authorize_transfer
    ///
    fun test_complete_transfer_with_payload_wrapped_asset() {
        use token_bridge::complete_transfer_with_payload::{
            authorize_transfer,
            redeem_coin
        };

        let transfer_vaa =
            dummy_message::encoded_transfer_with_payload_wrapped_12();

        let (user, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register chain ID 2 as a foreign emitter.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Register wrapped token.
        coin_wrapped_12::init_and_register(scenario, coin_deployer);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);

        // Set up dummy `EmitterCap` as the expected redeemer.
        let emitter_cap = emitter::dummy();

        // Verify that the emitter cap is the expected redeemer.
        let expected_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    parse_and_verify_vaa(scenario, transfer_vaa)
                )
            );
        assert!(
            transfer_with_payload::redeemer_id(&expected_transfer) == object::id(&emitter_cap),
            0
        );

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        // Execute authorize_transfer.
        let receipt =
            authorize_transfer<COIN_WRAPPED_12>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );
        let (
            bridged,
            parsed_transfer,
            source_chain
        ) = redeem_coin(&emitter_cap, receipt);
        assert!(source_chain == expected_source_chain, 0);

        // Assert coin value, source chain, and parsed transfer details are correct.
        let expected_bridged = 3000;
        assert!(coin::value(&bridged) == expected_bridged, 0);

        // Total supply should equal the amount just minted.
        let registry = state::borrow_token_registry(&token_bridge_state);
        {
            let asset =
                token_registry::borrow_wrapped<COIN_WRAPPED_12>(registry);
            assert!(wrapped_asset::total_supply(asset) == expected_bridged, 0);
        };

        // Verify token info.
        let verified =
            token_registry::verified_asset<COIN_WRAPPED_12>(registry);
        let expected_token_chain = token_registry::token_chain(&verified);
        let expected_token_address = token_registry::token_address(&verified);
        assert!(expected_token_chain != chain_id(), 0);
        assert!(
            transfer_with_payload::token_chain(&parsed_transfer) == expected_token_chain,
            0
        );
        assert!(
            transfer_with_payload::token_address(&parsed_transfer) == expected_token_address,
            0
        );

        // Verify transfer by serializing both parsed and expected.
        let serialized = transfer_with_payload::serialize(parsed_transfer);
        let expected_serialized =
            transfer_with_payload::serialize(expected_transfer);
        assert!(serialized == expected_serialized, 0);

        // Clean up.
        return_state(token_bridge_state);
        coin::burn_for_testing(bridged);
        emitter::destroy_test_only(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = complete_transfer_with_payload::E_INVALID_REDEEMER,
    )]
    /// Test the public-facing function authorize_transfer.
    /// This test fails because the ecmitter_cap (recipient) is incorrect (0x2 instead of 0x3).
    ///
    fun test_cannot_complete_transfer_with_payload_invalid_redeemer() {
        use token_bridge::complete_transfer_with_payload::{
            authorize_transfer,
            redeem_coin
        };

        let transfer_vaa =
            dummy_message::encoded_transfer_with_payload_wrapped_12();

        let (user, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register chain ID 2 as a foreign emitter.
        register_dummy_emitter(scenario, 2);

        // Register wrapped asset with 12 decimals.
        coin_wrapped_12::init_and_register(scenario, coin_deployer);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);

        let parsed =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    parse_and_verify_vaa(scenario, transfer_vaa)
                )
            );

        // Because the vaa expects the dummy emitter as the redeemer, we need
        // to generate another emitter.
        let emitter_cap = new_emitter(scenario);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        assert!(
            transfer_with_payload::redeemer_id(&parsed) != object::id(&emitter_cap),
            0
        );

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        let receipt =
            authorize_transfer<COIN_WRAPPED_12>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );
        // You shall not pass!
        let (
            bridged_out,
            _,
            _
        ) = redeem_coin(&emitter_cap, receipt);

        // Clean up.
        coin::burn_for_testing(bridged_out);

        abort 42
    }

    #[test]
    #[expected_failure(
        abort_code = complete_transfer::E_CANONICAL_TOKEN_INFO_MISMATCH
    )]
    /// This test demonstrates that the `CoinType` specified for the token
    /// redemption must agree with the canonical token info encoded in the VAA_ATTESTED_DECIMALS_12,
    /// which is registered with the Token Bridge.
    fun test_cannot_complete_transfer_with_payload_wrong_coin_type() {
        use token_bridge::complete_transfer_with_payload::{
            authorize_transfer
        };

        let transfer_vaa =
            dummy_message::encoded_transfer_with_payload_wrapped_12();

        let (user, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register chain ID 2 as a foreign emitter.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Register wrapped token.
        coin_wrapped_12::init_and_register(scenario, coin_deployer);

        // Also register unexpected token (in this case a native one).
        coin_native_10::init_and_register(scenario, coin_deployer);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);

        let registry = state::borrow_token_registry(&token_bridge_state);

        // Set up dummy `EmitterCap` as the expected redeemer.
        let emitter_cap = emitter::dummy();

        // Verify that the emitter cap is the expected redeemer.
        let expected_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    parse_and_verify_vaa(scenario, transfer_vaa)
                )
            );
        assert!(
            transfer_with_payload::redeemer_id(&expected_transfer) == object::id(&emitter_cap),
            0
        );

        // Also verify that the encoded token info disagrees with the expected
        // token info.
        let verified =
            token_registry::verified_asset<COIN_NATIVE_10>(registry);
        let expected_token_chain = token_registry::token_chain(&verified);
        let expected_token_address = token_registry::token_address(&verified);
        assert!(
            transfer_with_payload::token_chain(&expected_transfer) != expected_token_chain,
            0
        );
        assert!(
            transfer_with_payload::token_address(&expected_transfer) != expected_token_address,
            0
        );

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        // You shall not pass!
        let receipt =
            authorize_transfer<COIN_NATIVE_10>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        return_state(token_bridge_state);
        complete_transfer_with_payload::burn(receipt);
        emitter::destroy_test_only(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = complete_transfer::E_TARGET_NOT_SUI)]
    /// This test verifies that `complete_transfer` reverts when a transfer is
    /// sent to the wrong target blockchain (chain ID != 21).
    fun test_cannot_complete_transfer_with_payload_wrapped_asset_invalid_target_chain() {
        use token_bridge::complete_transfer_with_payload::{
            authorize_transfer
        };

        let transfer_vaa =
            dummy_message::encoded_transfer_with_payload_wrapped_12_invalid_target_chain();

        let (user, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register chain ID 2 as a foreign emitter.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Register wrapped token.
        coin_wrapped_12::init_and_register(scenario, coin_deployer);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);

        // Set up dummy `EmitterCap` as the expected redeemer.
        let emitter_cap = emitter::dummy();

        // Verify that the emitter cap is the expected redeemer.
        let expected_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    parse_and_verify_vaa(scenario, transfer_vaa)
                )
            );
        assert!(
            transfer_with_payload::redeemer_id(&expected_transfer) == object::id(&emitter_cap),
            0
        );

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        // You shall not pass!
        let receipt =
            authorize_transfer<COIN_WRAPPED_12>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        complete_transfer_with_payload::burn(receipt);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_complete_transfer_with_payload_outdated_version() {
        use token_bridge::complete_transfer_with_payload::{authorize_transfer};

        let transfer_vaa =
            dummy_message::encoded_transfer_with_payload_vaa_native();

        let (user, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register Sui as a foreign emitter.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Initialize native token.
        let mint_amount = 1000000;
        coin_native_10::init_register_and_deposit(
            scenario,
            coin_deployer,
            mint_amount
        );

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        let token_bridge_state = take_state(scenario);

        // Set up dummy `EmitterCap` as the expected redeemer.
        let emitter_cap = emitter::dummy();

        // Verify that the emitter cap is the expected redeemer.
        let expected_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    parse_and_verify_vaa(scenario, transfer_vaa)
                )
            );
        assert!(
            transfer_with_payload::redeemer_id(&expected_transfer) == object::id(&emitter_cap),
            0
        );

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

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
        let receipt =
            authorize_transfer<COIN_NATIVE_10>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        complete_transfer_with_payload::burn(receipt);

        abort 42
    }
}
