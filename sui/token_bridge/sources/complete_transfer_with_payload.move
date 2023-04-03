// SPDX-License-Identifier: Apache 2

/// This module implements the method `complete_transfer_with_payload` which
/// allows a contract to redeem a Token Bridge transfer with arbitrary payload.
/// Like in `complete_transfer`, a VAA with an encoded transfer can be redeemed
/// only once.
///
/// See `transfer_with_payload` module for serialization and deserialization of
/// Wormhole message payload.
module token_bridge::complete_transfer_with_payload {
    use sui::balance::{Balance};
    use sui::clock::{Clock};
    use sui::object::{Self};
    use wormhole::emitter::{EmitterCap};
    use wormhole::state::{State as WormholeState};

    use token_bridge::complete_transfer::{Self};
    use token_bridge::state::{Self, State};
    use token_bridge::transfer_with_payload::{Self, TransferWithPayload};
    use token_bridge::vaa::{Self};
    use token_bridge::version_control::{
        CompleteTransferWithPayload as CompleteTransferWithPayloadControl
    };

    /// `EmitterCap` address does not agree with encoded redeemer.
    const E_INVALID_REDEEMER: u64 = 0;

    /// `complete_transfer_with_payload` deserializes a token transfer VAA with
    /// an arbitrary payload. The specified `EmitterCap` is the only authorized
    /// redeemer of this transfer. Once the transfer is redeemed, an event
    /// (`TransferRedeemed`) is emitted to reflect which Token Bridge this
    /// transfer originated from.
    public fun complete_transfer_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        the_clock: &Clock
    ): (
        Balance<CoinType>,
        TransferWithPayload,
        u16 // `wormhole::vaa::emitter_chain`
    ) {
        state::check_minimum_requirement<CompleteTransferWithPayloadControl>(
            token_bridge_state
        );

        // Parse and verify Token Bridge transfer message. This method
        // guarantees that a verified transfer message cannot be redeemed again.
        let parsed_vaa =
            vaa::parse_verify_and_consume(
                token_bridge_state,
                worm_state,
                vaa_buf,
                the_clock
            );

        // Emitting the transfer being redeemed.
        //
        // NOTE: We save the emitter chain ID to save the integrator from
        // having to `parse_and_verify` the same encoded VAA to get this info.
        let source_chain =
            complete_transfer::emit_transfer_redeemed(&parsed_vaa);

        // Finally deserialize the Wormhole message payload and handle bridging
        // out token of a given coin type.
        handle_complete_transfer_with_payload(
            token_bridge_state,
            emitter_cap,
            source_chain,
            wormhole::vaa::take_payload(parsed_vaa)
        )
    }

    fun handle_complete_transfer_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        source_chain: u16,
        transfer_vaa_payload: vector<u8>
    ): (
        Balance<CoinType>,
        TransferWithPayload,
        u16 // `wormhole::vaa::emitter_chain`
    ) {
        // Deserialize for processing.
        let parsed = transfer_with_payload::deserialize(transfer_vaa_payload);

        // Transfer must be redeemed by the contract's registered Wormhole
        // emitter.
        let redeemer = transfer_with_payload::redeemer_id(&parsed);
        assert!(redeemer == object::id(emitter_cap), E_INVALID_REDEEMER);

        // Handle bridging assets out to be returned to method caller.
        //
        // See `complete_transfer` module for more info.
        let (
            _,
            bridged_out,
        ) =
            complete_transfer::verify_and_bridge_out(
                token_bridge_state,
                transfer_with_payload::token_chain(&parsed),
                transfer_with_payload::token_address(&parsed),
                transfer_with_payload::redeemer_chain(&parsed),
                transfer_with_payload::amount(&parsed)
            );

        (bridged_out, parsed, source_chain)
    }
}

#[test_only]
module token_bridge::complete_transfer_with_payload_tests {
    use sui::balance::{Self};
    use sui::object::{Self};
    use sui::test_scenario::{Self};
    use wormhole::emitter::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::coin_wrapped_12::{Self, COIN_WRAPPED_12};
    use token_bridge::complete_transfer_with_payload::{Self};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::dummy_message::{Self};
    use token_bridge::native_asset::{Self};
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
    use token_bridge::transfer_with_payload::{Self};
    use token_bridge::wrapped_asset::{Self};

    #[test]
    /// Test the public-facing function complete_transfer_with_payload.
    /// using a native transfer VAA_ATTESTED_DECIMALS_12.
    fun test_complete_transfer_with_payload_native_asset() {
        use token_bridge::complete_transfer_with_payload::{
            complete_transfer_with_payload
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

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

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
                    wormhole::vaa::parse_and_verify(
                        &worm_state,
                        transfer_vaa,
                        &the_clock
                    )
                )
            );
        assert!(
            transfer_with_payload::redeemer_id(&expected_transfer) == object::id(&emitter_cap),
            0
        );

        // Execute complete_transfer_with_payload.
        let (
            bridged,
            parsed_transfer,
            source_chain
        ) =
            complete_transfer_with_payload<COIN_NATIVE_10>(
                &mut token_bridge_state,
                &emitter_cap,
                &mut worm_state,
                transfer_vaa,
                &the_clock
            );
        assert!(source_chain == expected_source_chain, 0);

        // Assert coin value, source chain, and parsed transfer details are correct.
        // We expect the coin value to be 300000, because that's in terms of
        // 10 decimals. The amount specifed in the VAA_ATTESTED_DECIMALS_12 is 3000, because that's
        // in terms of 8 decimals.
        let expected_bridged = 300000;
        assert!(balance::value(&bridged) == expected_bridged, 0);

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
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        balance::destroy_for_testing(bridged);
        emitter::destroy(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    /// Test the public-facing function complete_transfer_with_payload.
    /// Use an actual devnet Wormhole complete transfer with payload VAA_ATTESTED_DECIMALS_12.
    ///
    /// This test confirms that:
    ///   - complete_transfer_with_payload function deserializes
    ///     the encoded Transfer object and recovers the source chain, payload,
    ///     and additional transfer details correctly.
    ///   - a wrapped coin with the correct value is minted by the bridge
    ///     and returned by complete_transfer_with_payload
    ///
    fun test_complete_transfer_with_payload_wrapped_asset() {
        use token_bridge::complete_transfer_with_payload::{
            complete_transfer_with_payload
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

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Set up dummy `EmitterCap` as the expected redeemer.
        let emitter_cap = emitter::dummy();

        // Verify that the emitter cap is the expected redeemer.
        let expected_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    wormhole::vaa::parse_and_verify(
                        &worm_state,
                        transfer_vaa,
                        &the_clock
                    )
                )
            );
        assert!(
            transfer_with_payload::redeemer_id(&expected_transfer) == object::id(&emitter_cap),
            0
        );

        // Execute complete_transfer_with_payload.
        let (
            bridged,
            parsed_transfer,
            source_chain
        ) =
            complete_transfer_with_payload<COIN_WRAPPED_12>(
                &mut token_bridge_state,
                &emitter_cap,
                &mut worm_state,
                transfer_vaa,
                &the_clock
            );
        assert!(source_chain == expected_source_chain, 0);

        // Assert coin value, source chain, and parsed transfer details are correct.
        let expected_bridged = 3000;
        assert!(balance::value(&bridged) == expected_bridged, 0);

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
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        balance::destroy_for_testing(bridged);
        emitter::destroy(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = complete_transfer_with_payload::E_INVALID_REDEEMER,
    )]
    /// Test the public-facing function complete_transfer_with_payload.
    /// This test fails because the ecmitter_cap (recipient) is incorrect (0x2 instead of 0x3).
    ///
    fun test_cannot_complete_transfer_with_payload_invalid_redeemer() {
        use token_bridge::complete_transfer_with_payload::{
            complete_transfer_with_payload
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

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        let parsed =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    wormhole::vaa::parse_and_verify(
                        &worm_state,
                        transfer_vaa,
                        &the_clock
                    )
                )
            );

        // Because the vaa expects the dummy emitter as the redeemer, we need
        // to generate another emitter.
        let emitter_cap =
            emitter::new(&worm_state, test_scenario::ctx(scenario));
        assert!(
            transfer_with_payload::redeemer_id(&parsed) != object::id(&emitter_cap),
            0
        );

        // You shall not pass!
        let (
            bridged,
            _,
            _
        ) =
            complete_transfer_with_payload<COIN_WRAPPED_12>(
                &mut token_bridge_state,
                &emitter_cap,
                &mut worm_state,
                transfer_vaa,
                &the_clock
            );

        // Clean up.
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        balance::destroy_for_testing(bridged);
        emitter::destroy(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = token_registry::E_CANONICAL_TOKEN_INFO_MISMATCH
    )]
    /// This test demonstrates that the `CoinType` specified for the token
    /// redemption must agree with the canonical token info encoded in the VAA_ATTESTED_DECIMALS_12,
    /// which is registered with the Token Bridge.
    fun test_cannot_complete_transfer_with_payload_wrong_coin_type() {
        use token_bridge::complete_transfer_with_payload::{
            complete_transfer_with_payload
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

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        let registry = state::borrow_token_registry(&token_bridge_state);

        // Set up dummy `EmitterCap` as the expected redeemer.
        let emitter_cap = emitter::dummy();

        // Verify that the emitter cap is the expected redeemer.
        let expected_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    wormhole::vaa::parse_and_verify(
                        &worm_state,
                        transfer_vaa,
                        &the_clock
                    )
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

        // You shall not pass!
        let (
            bridged,
            _,
            _
        ) =
            complete_transfer_with_payload<COIN_NATIVE_10>(
                &mut token_bridge_state,
                &emitter_cap,
                &mut worm_state,
                transfer_vaa,
                &the_clock
            );

        // Clean up.
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        balance::destroy_for_testing(bridged);
        emitter::destroy(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }
}
