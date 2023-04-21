// SPDX-License-Identifier: Apache 2

/// This module implements the method `transfer_tokens_with_payload` which
/// allows someone to bridge assets out of Sui to be redeemed on a foreign
/// network.
///
/// NOTE: Only assets that exist in the `TokenRegistry` can be bridged out,
/// which are native Sui assets that have been attested for via `attest_token`
/// and wrapped foreign assets that have been created using foreign asset
/// metadata via the `create_wrapped` module.
///
/// See `transfer_with_payload` module for serialization and deserialization of
/// Wormhole message payload.
module token_bridge::transfer_tokens_with_payload {
    use sui::clock::{Clock};
    use sui::coin::{Coin};
    use sui::sui::{SUI};
    use wormhole::bytes32::{Self};
    use wormhole::emitter::{EmitterCap};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{Self, State};
    use token_bridge::transfer_with_payload::{Self};
    use token_bridge::version_control::{
        TransferTokensWithPayload as TransferTokensWithPayloadControl
    };

    /// `transfer_tokens_with_payload` takes a `Coin` object of a coin type and
    /// bridges this asset out of Sui by either joining its balance in the
    /// Token Bridge's custody for native assets or burning its balance
    /// for wrapped assets.
    ///
    /// The `EmitterCap` is encoded as the sender of these assets. And
    /// associated with this transfer is an arbitrary payload, which can be
    /// consumed by the specified redeemer and used as instructions for a
    /// contract composing with Token Bridge.
    ///
    /// See `token_registry and `transfer_with_payload` module for more info.
    public fun transfer_tokens_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        worm_state: &mut WormholeState,
        bridged_in: Coin<CoinType>,
        wormhole_fee: Coin<SUI>,
        redeemer_chain: u16,
        redeemer: vector<u8>,
        payload: vector<u8>,
        nonce: u32,
        the_clock: &Clock
    ): (u64, Coin<CoinType>) {
        state::check_minimum_requirement<TransferTokensWithPayloadControl>(
            token_bridge_state
        );

        // Encode Wormhole message payload.
        let encoded_transfer_with_payload =
            bridge_in_and_serialize_transfer(
                token_bridge_state,
                emitter_cap,
                &mut bridged_in,
                redeemer_chain,
                external_address::new(bytes32::from_bytes(redeemer)),
                payload
            );

        // Publish.
        let message_sequence =
            state::publish_wormhole_message(
                token_bridge_state,
                worm_state,
                nonce,
                encoded_transfer_with_payload,
                wormhole_fee,
                the_clock
            );

        (message_sequence, bridged_in)
    }

    fun bridge_in_and_serialize_transfer<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        bridged_in: &mut Coin<CoinType>,
        redeemer_chain: u16,
        redeemer: ExternalAddress,
        payload: vector<u8>
    ): vector<u8> {
        use token_bridge::transfer_tokens::{verify_and_bridge_in};

        let (
            token_chain,
            token_address,
            norm_amount,
            _
        ) = verify_and_bridge_in(token_bridge_state, bridged_in, 0);

        transfer_with_payload::serialize(
            transfer_with_payload::new_from_emitter(
                emitter_cap,
                norm_amount,
                token_address,
                token_chain,
                redeemer,
                redeemer_chain,
                payload
            )
        )
    }

    #[test_only]
    public fun bridge_in_and_serialize_transfer_test_only<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        bridged_in: Coin<CoinType>,
        redeemer_chain: u16,
        redeemer: vector<u8>,
        payload: vector<u8>
    ): (vector<u8>, Coin<CoinType>) {
        let payload =
            bridge_in_and_serialize_transfer(
                token_bridge_state,
                emitter_cap,
                &mut bridged_in,
                redeemer_chain,
                external_address::new(bytes32::from_bytes(redeemer)),
                payload
            );

        (payload, bridged_in)
    }
}

#[test_only]
module token_bridge::transfer_tokens_with_payload_tests {
    use sui::coin::{Self};
    use sui::test_scenario::{Self};
    use sui::transfer::{public_transfer};

    use wormhole::external_address::{Self};
    use wormhole::state::{chain_id};
    use wormhole::emitter::{Self};
    use wormhole::bytes32::{Self};

    use token_bridge::coin_wrapped_7::{Self, COIN_WRAPPED_7};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::transfer_tokens_with_payload::{
        transfer_tokens_with_payload,
        bridge_in_and_serialize_transfer_test_only
    };
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        set_up_wormhole_and_token_bridge,
        register_dummy_emitter,
        return_clock,
        return_states,
        take_clock,
        take_states,
        person
    };
    use token_bridge::token_registry::{Self};
    use token_bridge::wrapped_asset::{Self};
    use token_bridge::native_asset::{Self};
    use token_bridge::transfer_with_payload::{Self};
    use token_bridge::normalized_amount::{Self};

    /// Test consts.
    const TEST_TARGET_RECIPIENT: vector<u8> = x"beef4269";
    const TEST_TARGET_CHAIN: u16 = 2;
    const TEST_NONCE: u32 = 0;
    const TEST_COIN_NATIVE_10_DECIMALS: u8 = 10;
    const TEST_COIN_WRAPPED_7_DECIMALS: u8 = 7;
    const TEST_MESSAGE_PAYLOAD: vector<u8> = x"deadbeefdeadbeef";

    #[test]
    fun test_transfer_tokens_with_payload_native_10() {
        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 6942000;
        let coin_10_balance = coin_native_10::init_register_and_mint(
            scenario,
            sender,
            transfer_amount
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be zero for COIN_NATIVE_10.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == 0, 0);
        };

        // Cache context.
        let ctx = test_scenario::ctx(scenario);

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        // Call `transfer_tokens_with_payload`.
        let (_, dust) =
            transfer_tokens_with_payload<COIN_NATIVE_10>(
                &mut token_bridge_state,
                &emitter_cap,
                &mut worm_state,
                coin::from_balance(coin_10_balance, ctx),
                coin::mint_for_testing(wormhole_fee, ctx),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD,
                TEST_NONCE,
                &the_clock,
            );
        assert!(coin::value(&dust) == 0, 0);

        // Balance check the Token Bridge after executing the transfer. The
        // balance should now reflect the `transfer_amount` defined in this
        // test.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == transfer_amount, 0);
        };

        // Done.
        return_states(token_bridge_state, worm_state);
        coin::destroy_zero(dust);
        emitter::destroy(emitter_cap);
        return_clock(the_clock);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_transfer_tokens_native_10_with_dust_refund() {
        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 1000069;
        let coin_10_balance = coin_native_10::init_register_and_mint(
            scenario,
            sender,
            transfer_amount
        );

        // This value will be used later. The contract should return dust
        // to the caller since COIN_NATIVE_10 has 10 decimals.
        let expected_dust = 69;

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be zero for COIN_NATIVE_10.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == 0, 0);
        };

        // Cache context.
        let ctx = test_scenario::ctx(scenario);

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        // Call `transfer_tokens`.
        let (_, dust) =
            transfer_tokens_with_payload<COIN_NATIVE_10>(
                &mut token_bridge_state,
                &emitter_cap,
                &mut worm_state,
                coin::from_balance(coin_10_balance, ctx),
                coin::mint_for_testing(wormhole_fee, ctx),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD,
                TEST_NONCE,
                &the_clock
        );
        assert!(coin::value(&dust) == expected_dust, 0);

        // Balance check the Token Bridge after executing the transfer. The
        // balance should now reflect the `transfer_amount` less `expected_dust`
        // defined in this test.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(
                native_asset::custody(asset) == transfer_amount - expected_dust,
                0
            );
        };

        // Done.
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        emitter::destroy(emitter_cap);
        public_transfer(dust, @0x0);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_serialize_transfer_tokens_native_10() {
        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 6942000;
        let coin_10_balance = coin_native_10::init_register_and_mint(
            scenario,
            sender,
            transfer_amount
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Cache context.
        let ctx = test_scenario::ctx(scenario);

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        // Serialize the payload.
        let (payload, dust) =
            bridge_in_and_serialize_transfer_test_only(
                &mut token_bridge_state,
                &emitter_cap,
                coin::from_balance(coin_10_balance, ctx),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD
            );
        assert!(coin::value(&dust) == 0, 0);

        // Construct expected payload from scratch and confirm that the
        // `transfer_tokens` call produces the same payload.
        let expected_token_address = token_registry::token_address<COIN_NATIVE_10>(
            &state::verified_asset<COIN_NATIVE_10>(
                &token_bridge_state
            )
        );
        let expected_amount = normalized_amount::from_raw(
            transfer_amount,
            TEST_COIN_NATIVE_10_DECIMALS
        );

        let expected_payload =
            transfer_with_payload::new_from_emitter_test_only(
                &emitter_cap,
                expected_amount,
                expected_token_address,
                chain_id(),
                external_address::new(bytes32::from_bytes(TEST_TARGET_RECIPIENT)),
                TEST_TARGET_CHAIN,
                TEST_MESSAGE_PAYLOAD
            );
        assert!(
            transfer_with_payload::serialize(expected_payload) == payload,
            0
        );

        // Done.
        coin::destroy_zero(dust);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        emitter::destroy(emitter_cap);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_transfer_tokens_with_payload_wrapped_7() {
        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 6942000;
        let coin_7_balance = coin_wrapped_7::init_register_and_mint(
            scenario,
            sender,
            transfer_amount
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be the `transfer_amount` for COIN_WRAPPED_7.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_wrapped<COIN_WRAPPED_7>(registry);
            assert!(wrapped_asset::total_supply(asset) == transfer_amount, 0);
        };

        // Cache context.
        let ctx = test_scenario::ctx(scenario);

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        // Call `transfer_tokens_with_payload`.
        let (_, dust) =
            transfer_tokens_with_payload<COIN_WRAPPED_7>(
                &mut token_bridge_state,
                &emitter_cap,
                &mut worm_state,
                coin::from_balance(coin_7_balance, ctx),
                coin::mint_for_testing(wormhole_fee, ctx),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD,
                TEST_NONCE,
                &the_clock,
            );
        assert!(coin::value(&dust) == 0, 0);

        // Balance check the Token Bridge after executing the transfer. The
        // balance should be zero, since tokens are burned when an outbound
        // wrapped token transfer occurs.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_wrapped<COIN_WRAPPED_7>(registry);
            assert!(wrapped_asset::total_supply(asset) == 0, 0);
        };

        // Done.
        return_states(token_bridge_state, worm_state);
        coin::destroy_zero(dust);
        emitter::destroy(emitter_cap);
        return_clock(the_clock);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_serialize_transfer_tokens_wrapped_7() {
        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 6942000;
        let coin_7_balance = coin_wrapped_7::init_register_and_mint(
            scenario,
            sender,
            transfer_amount
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Cache context.
        let ctx = test_scenario::ctx(scenario);

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        // Serialize the payload.
        let (payload, dust) =
            bridge_in_and_serialize_transfer_test_only(
                &mut token_bridge_state,
                &emitter_cap,
                coin::from_balance(coin_7_balance, ctx),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD
            );
        assert!(coin::value(&dust) == 0, 0);

        // Construct expected payload from scratch and confirm that the
        // `transfer_tokens` call produces the same payload.
        let expected_token_address = token_registry::token_address<COIN_WRAPPED_7>(
            &state::verified_asset<COIN_WRAPPED_7>(
                &token_bridge_state
            )
        );
        let expected_token_chain = token_registry::token_chain<COIN_WRAPPED_7>(
            &state::verified_asset<COIN_WRAPPED_7>(
                &token_bridge_state
            )
        );
        let expected_amount = normalized_amount::from_raw(
            transfer_amount,
            TEST_COIN_WRAPPED_7_DECIMALS
        );

        let expected_payload =
            transfer_with_payload::new_from_emitter_test_only(
                &emitter_cap,
                expected_amount,
                expected_token_address,
                expected_token_chain,
                 external_address::new(bytes32::from_bytes(TEST_TARGET_RECIPIENT)),
                TEST_TARGET_CHAIN,
                TEST_MESSAGE_PAYLOAD
            );
        assert!(
            transfer_with_payload::serialize(expected_payload) == payload,
            0
        );

        // Done.
        coin::destroy_zero(dust);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        emitter::destroy(emitter_cap);
        test_scenario::end(my_scenario);
    }
}
