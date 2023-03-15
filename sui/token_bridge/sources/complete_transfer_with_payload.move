module token_bridge::complete_transfer_with_payload {
    use sui::balance::{Balance};
    use sui::tx_context::{TxContext};
    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{State};
    use token_bridge::transfer_with_payload::{Self, TransferWithPayload};
    use token_bridge::vaa::{Self};

    const E_INVALID_TARGET: u64 = 0;
    const E_INVALID_REDEEMER: u64 = 1;

    public fun complete_transfer_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        ctx: &mut TxContext
    ): (Balance<CoinType>, TransferWithPayload, u16) {
        // Parse and verify Token Bridge transfer message. This method
        // guarantees that a verified transfer message cannot be redeemed again.
        let parsed_vaa =
            vaa::parse_verify_and_consume(
                token_bridge_state,
                worm_state,
                vaa_buf,
                ctx
            );

        // Before destroying VAA_ATTESTED_DECIMALS_12, store the emitter chain ID for the caller.
        let source_chain = wormhole::vaa::emitter_chain(&parsed_vaa);

        // Deserialize for processing.
        let parsed_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(parsed_vaa)
            );

        let bridged =
            handle_complete_transfer_with_payload(
                token_bridge_state,
                emitter_cap,
                &parsed_transfer
            );

        (bridged, parsed_transfer, source_chain)
    }

    fun handle_complete_transfer_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        parsed: &TransferWithPayload
    ): Balance<CoinType> {
        use token_bridge::complete_transfer::{verify_and_take_coin};

        // Transfer must be redeemed by the contract's registered Wormhole
        // emitter.
        let redeemer = transfer_with_payload::redeemer(parsed);
        assert!(redeemer == emitter::addr(emitter_cap), E_INVALID_REDEEMER);

        let (bridged, _) =
            verify_and_take_coin<CoinType>(
                token_bridge_state,
                transfer_with_payload::token_chain(parsed),
                transfer_with_payload::token_address(parsed),
                transfer_with_payload::redeemer_chain(parsed),
                transfer_with_payload::amount(parsed)
            );

        bridged
    }
}

#[test_only]
module token_bridge::complete_transfer_with_payload_test {
    use sui::balance::{Self};
    use sui::test_scenario::{Self};
    use wormhole::emitter::{Self};
    use wormhole::external_address::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::coin_wrapped_12::{Self, COIN_WRAPPED_12};
    use token_bridge::complete_transfer_with_payload::{Self};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        register_dummy_emitter,
        return_states,
        set_up_wormhole_and_token_bridge,
        take_states,
        two_people
    };
    use token_bridge::transfer_with_payload::{Self};

    /// Transfer for COIN_WRAPPED_12.
    ///
    ///   signatures: [
    ///     {
    ///       guardianSetIndex: 0,
    ///       signature: 'd8e4e04ac55ed24773a31b0a89bab8c1b9201e76bd03fe0de9da1506058ab30c01344cf11a47005bdfbe47458cb289388e4a87ed271fb8306fd83656172b19dc01'
    ///     }
    ///   ],
    ///   emitterChain: 2,
    ///   emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    ///   sequence: 1n,
    ///   consistencyLevel: 15,
    ///   payload: {
    ///     module: 'TokenBridge',
    ///     type: 'TransferWithPayload',
    ///     amount: 3000n,
    ///     tokenAddress: '0x00000000000000000000000000000000000000000000000000000000beefface',
    ///     tokenChain: 2,
    ///     toAddress: '0x0000000000000000000000000000000000000000000000000000000000000003',
    ///     chain: 21,
    ///     fromAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    ///     payload: '0xaaaa'
    ///   }
    const VAA_ATTESTED_DECIMALS_12: vector<u8> =
        x"01000000000100d8e4e04ac55ed24773a31b0a89bab8c1b9201e76bd03fe0de9da1506058ab30c01344cf11a47005bdfbe47458cb289388e4a87ed271fb8306fd83656172b19dc010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f030000000000000000000000000000000000000000000000000000000000000bb800000000000000000000000000000000000000000000000000000000beefface00020000000000000000000000000000000000000000000000000000000000000003001500000000000000000000000000000000000000000000000000000000deadbeefaaaa";

    /// Transfer for NATIVE_DECIMALS_10.
    ///
    /// signatures: [
    ///     {
    ///       guardianSetIndex: 0,
    ///       signature: '2c8599ebc4e5f1ca832ad21e208226f22cff674c9db9dc6aca18b953b49c65154641e0b4074a0ff435b2b3380c87f457222ef77250722bf2aa50940b371af99901'
    ///     }
    ///   ],
    ///   emitterChain: 21,
    ///   emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    ///   sequence: 1n,
    ///   consistencyLevel: 0,
    ///   payload: {
    ///     module: 'TokenBridge',
    ///     type: 'TransferWithPayload',
    ///     amount: 3000n,
    ///     tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000001',
    ///     tokenChain: 21,
    ///     toAddress: '0x0000000000000000000000000000000000000000000000000000000000000003',
    ///     chain: 21,
    ///     fromAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    ///     payload: '0xaaaa'
    ///   }
    ///
    const VAA_NATIVE_DECIMALS_10: vector<u8> =
        x"01000000000100db621e2bd419cd8c254ec15827bded51bf79f45c0df9923c9071a50ae7b3cdec44d3ff45db0dc5caa17ad36f48bf06e34995a83c76c77eb5c541b036586c0748000000000000000000001500000000000000000000000000000000000000000000000000000000deadbeef000000000000000100030000000000000000000000000000000000000000000000000000000000000bb8000000000000000000000000000000000000000000000000000000000000000100150000000000000000000000000000000000000000000000000000000000000003001500000000000000000000000000000000000000000000000000000000deadbeefaaaa";

    #[test]
    /// Test the public-facing function complete_transfer_with_payload.
    /// using a native transfer VAA_ATTESTED_DECIMALS_12.
    fun test_complete_transfer_with_payload_native_asset() {
        use token_bridge::complete_transfer_with_payload::{
            complete_transfer_with_payload
        };

        let (user, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 0;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register Sui as a foreign emitter.
        let expected_source_chain = 21;
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

        let (bridge_state, worm_state) = take_states(scenario);
        assert!(
            state::custody_balance<COIN_NATIVE_10>(&bridge_state) == mint_amount,
            0
        );

        // Set up dummy `EmitterCap` as the expected redeemer.
        let emitter_cap =
            emitter::dummy_cap(
                external_address::from_any_bytes(x"03")
            );

        // Verify that the emitter cap is the expected redeemer.
        let expected_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    wormhole::vaa::parse_and_verify(
                        &worm_state,
                        VAA_NATIVE_DECIMALS_10,
                        test_scenario::ctx(scenario)
                    )
                )
            );
        assert!(
            transfer_with_payload::redeemer(&expected_transfer) == emitter::addr(&emitter_cap),
            0
        );

        // Execute complete_transfer_with_payload.
        let (
            bridged,
            parsed_transfer,
            source_chain
        ) =
            complete_transfer_with_payload<COIN_NATIVE_10>(
                &mut bridge_state,
                &emitter_cap,
                &mut worm_state,
                VAA_NATIVE_DECIMALS_10,
                test_scenario::ctx(scenario)
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
        assert!(
            state::custody_balance<COIN_NATIVE_10>(&bridge_state) == remaining,
            0
        );

        // Verify token info.
        let (
            expected_token_chain,
            expected_token_address
        ) = state::token_info<COIN_NATIVE_10>(&bridge_state);
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
        return_states(bridge_state, worm_state);
        balance::destroy_for_testing(bridged);
        emitter::destroy_cap(emitter_cap);

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

        let (user, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 0;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register chain ID 2 as a foreign emitter.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Register wrapped token.
        coin_wrapped_12::init_and_register(scenario, coin_deployer);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        let (bridge_state, worm_state) = take_states(scenario);

        // Set up dummy `EmitterCap` as the expected redeemer.
        let emitter_cap =
            emitter::dummy_cap(
                external_address::from_any_bytes(x"03")
            );

        // Verify that the emitter cap is the expected redeemer.
        let expected_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    wormhole::vaa::parse_and_verify(
                        &worm_state,
                        VAA_ATTESTED_DECIMALS_12,
                        test_scenario::ctx(scenario)
                    )
                )
            );
        assert!(
            transfer_with_payload::redeemer(&expected_transfer) == emitter::addr(&emitter_cap),
            0
        );

        // Execute complete_transfer_with_payload.
        let (
            bridged,
            parsed_transfer,
            source_chain
        ) =
            complete_transfer_with_payload<COIN_WRAPPED_12>(
                &mut bridge_state,
                &emitter_cap,
                &mut worm_state,
                VAA_ATTESTED_DECIMALS_12,
                test_scenario::ctx(scenario)
            );
        assert!(source_chain == expected_source_chain, 0);

        // Assert coin value, source chain, and parsed transfer details are correct.
        let expected_bridged = 3000;
        assert!(balance::value(&bridged) == expected_bridged, 0);

        // Total supply should equal the amount just minted.
        assert!(
            state::wrapped_supply<COIN_WRAPPED_12>(&bridge_state) == expected_bridged,
            0
        );

        // Verify token info.
        let (
            expected_token_chain,
            expected_token_address
        ) = state::token_info<COIN_WRAPPED_12>(&bridge_state);
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
        return_states(bridge_state, worm_state);
        balance::destroy_for_testing(bridged);
        emitter::destroy_cap(emitter_cap);

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

        let (user, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 0;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register chain ID 2 as a foreign emitter.
        register_dummy_emitter(scenario, 2);

        // Register wrapped asset with 12 decimals.
        coin_wrapped_12::init_and_register(scenario, coin_deployer);

        // Ignore effects. Begin processing as arbitrary tx executor.
        test_scenario::next_tx(scenario, user);

        let (bridge_state, worm_state) = take_states(scenario);

        // Set up dummy `EmitterCap`. Verify that this emitter is not the
        // expected redeemer.
        let emitter_cap =
            emitter::dummy_cap(
                external_address::from_any_bytes(x"1337")
            );
        let parsed =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    wormhole::vaa::parse_and_verify(
                        &worm_state,
                        VAA_ATTESTED_DECIMALS_12,
                        test_scenario::ctx(scenario)
                    )
                )
            );
        assert!(
            transfer_with_payload::sender(&parsed) != emitter::addr(&emitter_cap),
            0
        );

        // You shall not pass!
        let (
            bridged,
            _,
            _
        ) =
            complete_transfer_with_payload<COIN_WRAPPED_12>(
                &mut bridge_state,
                &emitter_cap,
                &mut worm_state,
                VAA_ATTESTED_DECIMALS_12,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        return_states(bridge_state, worm_state);
        balance::destroy_for_testing(bridged);
        emitter::destroy_cap(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = state::E_CANONICAL_TOKEN_INFO_MISMATCH)]
    /// This test demonstrates that the `CoinType` specified for the token
    /// redemption must agree with the canonical token info encoded in the VAA_ATTESTED_DECIMALS_12,
    /// which is registered with the Token Bridge.
    fun test_cannot_complete_transfer_with_payload_wrong_coin_type() {
        use token_bridge::complete_transfer_with_payload::{
            complete_transfer_with_payload
        };

        let (user, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        // Initialize Wormhole and Token Bridge.
        let wormhole_fee = 0;
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

        let (bridge_state, worm_state) = take_states(scenario);

        // Set up dummy `EmitterCap` as the expected redeemer.
        let emitter_cap =
            emitter::dummy_cap(
                external_address::from_any_bytes(x"03")
            );

        // Verify that the emitter cap is the expected redeemer.
        let expected_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(
                    wormhole::vaa::parse_and_verify(
                        &worm_state,
                        VAA_ATTESTED_DECIMALS_12,
                        test_scenario::ctx(scenario)
                    )
                )
            );
        assert!(
            transfer_with_payload::redeemer(&expected_transfer) == emitter::addr(&emitter_cap),
            0
        );

        // Also verify that the encoded token info disagrees with the expected
        // token info.
        let (
            expected_token_chain,
            expected_token_address
        ) = state::token_info<COIN_NATIVE_10>(&bridge_state);
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
                &mut bridge_state,
                &emitter_cap,
                &mut worm_state,
                VAA_ATTESTED_DECIMALS_12,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        return_states(bridge_state, worm_state);
        balance::destroy_for_testing(bridged);
        emitter::destroy_cap(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }
}
