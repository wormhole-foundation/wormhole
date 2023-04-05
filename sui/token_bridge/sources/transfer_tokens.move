// SPDX-License-Identifier: Apache 2

/// This module implements the method `transfer_tokens` which allows someone
/// to bridge assets out of Sui to be redeemed on a foreign network.
///
/// NOTE: Only assets that exist in the `TokenRegistry` can be bridged out,
/// which are native Sui assets that have been attested for via `attest_token`
/// and wrapped foreign assets that have been created using foreign asset
/// metadata via the `create_wrapped` module.
///
/// See `transfer` module for serialization and deserialization of Wormhole
/// message payload.
module token_bridge::transfer_tokens {
    use sui::balance::{Self};
    use sui::clock::{Clock};
    use sui::coin::{Self, Coin};
    use sui::sui::{SUI};
    use sui::tx_context::{Self, TxContext};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{State as WormholeState};

    use token_bridge::native_asset::{Self};
    use token_bridge::normalized_amount::{Self, NormalizedAmount};
    use token_bridge::state::{Self, State};
    use token_bridge::token_registry::{Self};
    use token_bridge::transfer::{Self};
    use token_bridge::version_control::{
        TransferTokens as TransferTokensControl
    };
    use token_bridge::wrapped_asset::{Self};

    friend token_bridge::transfer_tokens_with_payload;

    /// Relayer fee exceeds `Coin` object's value.
    const E_RELAYER_FEE_EXCEEDS_AMOUNT: u64 = 0;

    /// `transfer_tokens` takes a `Coin` object of a coin type and bridges this
    /// asset out of Sui by either joining its balance in the Token Bridge's
    /// custody for native assets or burning its balance for wrapped assets.
    ///
    /// Additionally, a `relayer_fee` of some value less than or equal to the
    /// `Coin` object's value can be specified to incentivize someone to redeem
    /// this transfer on behalf of the `recipient`.
    ///
    /// See `token_registry and `transfer_with_payload` module for more info.
    public fun transfer_tokens<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        bridged_in: Coin<CoinType>,
        wormhole_fee: Coin<SUI>,
        recipient_chain: u16,
        recipient: ExternalAddress,
        relayer_fee: u64,
        nonce: u32,
        the_clock: &Clock
    ): (u64, Coin<CoinType>) {
        state::check_minimum_requirement<TransferTokensControl>(
            token_bridge_state
        );

        let encoded_transfer =
            bridge_in_and_serialize_transfer(
                token_bridge_state,
                &mut bridged_in,
                recipient_chain,
                recipient,
                relayer_fee
            );

        // Publish with encoded `Transfer`.
        let message_sequence =
            state::publish_wormhole_message(
                token_bridge_state,
                worm_state,
                nonce,
                encoded_transfer,
                wormhole_fee,
                the_clock
            );

        // In addition to the Wormhole sequence number, return the `Coin` object
        // to the caller. This object have value if there was remaining dust
        // for a native Sui coin.
        (message_sequence, bridged_in)
    }

    /// Convenience method for those integrators that use `Coin` objects, where
    /// `bridged_in` will be destroyed if the value is zero. Otherwise it will
    /// be returned back to the transaction sender.
    public fun return_dust_to_sender<CoinType>(
        bridged_in: Coin<CoinType>,
        ctx: &TxContext
    ) {
        if (coin::value(&bridged_in) == 0) {
            coin::destroy_zero(bridged_in);
        } else {
            sui::transfer::public_transfer(bridged_in, tx_context::sender(ctx));
        };
    }

    /// For a given `CoinType`, prepare outbound transfer.
    ///
    /// This method is also used in `transfer_tokens_with_payload`.
    public(friend) fun verify_and_bridge_in<CoinType>(
        token_bridge_state: &mut State,
        bridged_in: &mut Coin<CoinType>,
        relayer_fee: u64
    ): (
        u16,
        ExternalAddress,
        NormalizedAmount,
        NormalizedAmount
    ) {
        // Disallow `relayer_fee` to be greater than the `Coin` object's value.
        let amount = coin::value(bridged_in);
        assert!(relayer_fee <= amount, E_RELAYER_FEE_EXCEEDS_AMOUNT);

        // Fetch canonical token info from registry.
        let verified = state::verified_asset<CoinType>(token_bridge_state);

        // Calculate dust. If there is any, `bridged_in` will have remaining
        // value after split. `norm_amount` is copied since it is denormalized
        // at this step.
        let decimals = token_registry::coin_decimals(&verified);
        let norm_amount = normalized_amount::from_raw(amount, decimals);

        // Split the `bridged_in` coin object to return any dust remaining on
        // that object. Only bridge in the adjusted amount after de-normalizing
        // the normalized amount.
        let adjusted_bridged_in =
            balance::split(
                coin::balance_mut(bridged_in),
                normalized_amount::to_raw(norm_amount, decimals)
            );

        // Either burn or deposit depending on `CoinType`.
        let registry = state::borrow_mut_token_registry(token_bridge_state);
        if (token_registry::is_wrapped(&verified)) {
            wrapped_asset::burn(
                token_registry::borrow_mut_wrapped(registry),
                adjusted_bridged_in
            );
        } else {
            native_asset::deposit(
                token_registry::borrow_mut_native(registry),
                adjusted_bridged_in
            );
        };

        (
            token_registry::token_chain(&verified),
            token_registry::token_address(&verified),
            norm_amount,
            normalized_amount::from_raw(relayer_fee, decimals)
        )
    }

    fun bridge_in_and_serialize_transfer<CoinType>(
        token_bridge_state: &mut State,
        bridged_in: &mut Coin<CoinType>,
        recipient_chain: u16,
        recipient: ExternalAddress,
        relayer_fee: u64
    ): vector<u8> {
        let (
            token_chain,
            token_address,
            norm_amount,
            norm_relayer_fee
        ) =
            verify_and_bridge_in(
                token_bridge_state,
                bridged_in,
                relayer_fee
            );

        transfer::serialize(
            transfer::new(
                norm_amount,
                token_address,
                token_chain,
                recipient,
                recipient_chain,
                norm_relayer_fee
            )
        )
    }

    #[test_only]
    public fun bridge_in_and_serialize_transfer_test_only<CoinType>(
        token_bridge_state: &mut State,
        bridged_in: Coin<CoinType>,
        recipient_chain: u16,
        recipient: ExternalAddress,
        relayer_fee: u64
    ): (vector<u8>, Coin<CoinType>) {
        let payload = bridge_in_and_serialize_transfer(
            token_bridge_state,
            &mut bridged_in,
            recipient_chain,
            recipient,
            relayer_fee
        );

        (payload, bridged_in)
    }
}

#[test_only]
module token_bridge::transfer_token_tests {
    use sui::coin::{Self};
    use sui::test_scenario::{Self};
    use sui::transfer::{public_transfer};

    use wormhole::external_address::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::coin_wrapped_7::{Self, COIN_WRAPPED_7};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::transfer_tokens::{Self};
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
    use token_bridge::transfer::{Self};
    use token_bridge::normalized_amount::{Self};

    /// Test consts.
    const TEST_TARGET_RECIPIENT: address = @0xbeef4269;
    const TEST_TARGET_CHAIN: u16 = 2;
    const TEST_NONCE: u32 = 0;
    const TEST_COIN_NATIVE_10_DECIMALS: u8 = 10;
    const TEST_COIN_WRAPPED_7_DECIMALS: u8 = 7;

    #[test]
    fun test_transfer_tokens_native_10() {
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

        // Define the relayer fee.
        let relayer_fee = 100000;

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be zero for COIN_NATIVE_10.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == 0, 0);
        };

        // Cache context.
        let ctx = test_scenario::ctx(scenario);

        // Call `transfer_tokens`.
        let (_, dust) = transfer_tokens::transfer_tokens<COIN_NATIVE_10>(
            &mut token_bridge_state,
            &mut worm_state,
            coin::from_balance(coin_10_balance, ctx),
            coin::mint_for_testing(wormhole_fee, ctx),
            TEST_TARGET_CHAIN,
            external_address::from_address(TEST_TARGET_RECIPIENT),
            relayer_fee,
            TEST_NONCE,
            &the_clock
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

        // Define the relayer fee.
        let relayer_fee = 100000;

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be zero for COIN_NATIVE_10.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == 0, 0);
        };

        // Cache context.
        let ctx = test_scenario::ctx(scenario);

        // Call `transfer_tokens`.
        let (_, dust) = transfer_tokens::transfer_tokens<COIN_NATIVE_10>(
            &mut token_bridge_state,
            &mut worm_state,
            coin::from_balance(coin_10_balance, ctx),
            coin::mint_for_testing(wormhole_fee, ctx),
            TEST_TARGET_CHAIN,
            external_address::from_address(TEST_TARGET_RECIPIENT),
            relayer_fee,
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

        // Define the relayer fee.
        let relayer_fee = 100000;

        // Cache context.
        let ctx = test_scenario::ctx(scenario);

        // Call `transfer_tokens`.
        let (payload, dust) = transfer_tokens::bridge_in_and_serialize_transfer_test_only(
            &mut token_bridge_state,
            coin::from_balance(coin_10_balance, ctx),
            TEST_TARGET_CHAIN,
            external_address::from_address(TEST_TARGET_RECIPIENT),
            relayer_fee
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
        let expected_relayer_fee = normalized_amount::from_raw(
            relayer_fee,
            TEST_COIN_NATIVE_10_DECIMALS
        );

        let expected_payload =
            transfer::new(
                expected_amount,
                expected_token_address,
                chain_id(),
                external_address::from_address(TEST_TARGET_RECIPIENT),
                TEST_TARGET_CHAIN,
                expected_relayer_fee
            );
        assert!(transfer::serialize(expected_payload) == payload, 0);

        // Done.
        coin::destroy_zero(dust);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_transfer_tokens_wrapped_7() {
        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 42069000;
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

        // Define the relayer fee.
        let relayer_fee = 100000;

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be the `transfer_amount` for COIN_WRAPPED_7.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_wrapped<COIN_WRAPPED_7>(registry);
            assert!(wrapped_asset::total_supply(asset) == transfer_amount, 0);
        };

        // Cache context.
        let ctx = test_scenario::ctx(scenario);

        // Call `transfer_tokens`.
        let (_, dust) = transfer_tokens::transfer_tokens<COIN_WRAPPED_7>(
            &mut token_bridge_state,
            &mut worm_state,
            coin::from_balance(coin_7_balance, ctx),
            coin::mint_for_testing(wormhole_fee, ctx),
            TEST_TARGET_CHAIN,
            external_address::from_address(TEST_TARGET_RECIPIENT),
            relayer_fee,
            TEST_NONCE,
            &the_clock
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
        coin::destroy_zero(dust);
        return_states(token_bridge_state, worm_state);
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

        // Define the relayer fee.
        let relayer_fee = 100000;

        // Cache context.
        let ctx = test_scenario::ctx(scenario);

        // Call `transfer_tokens`.
        let (payload, dust) = transfer_tokens::bridge_in_and_serialize_transfer_test_only(
            &mut token_bridge_state,
            coin::from_balance(coin_7_balance, ctx),
            TEST_TARGET_CHAIN,
            external_address::from_address(TEST_TARGET_RECIPIENT),
            relayer_fee
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
        let expected_relayer_fee = normalized_amount::from_raw(
            relayer_fee,
            TEST_COIN_WRAPPED_7_DECIMALS
        );

        let expected_payload =
            transfer::new(
                expected_amount,
                expected_token_address,
                expected_token_chain,
                external_address::from_address(TEST_TARGET_RECIPIENT),
                TEST_TARGET_CHAIN,
                expected_relayer_fee
            );
        assert!(transfer::serialize(expected_payload) == payload, 0);

        // Done.
        coin::destroy_zero(dust);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = token_registry::E_UNREGISTERED)]
    fun test_cannot_transfer_tokens_native_not_registered() {
        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Initialize COIN_NATIVE_10 (but don't register it).
        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // NOTE: This test purposely doesn't `attest` COIN_NATIVE_10.
        let transfer_amount = 6942000;
        let test_coins = coin::mint_for_testing<COIN_NATIVE_10>(
            transfer_amount,
            test_scenario::ctx(scenario)
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Define the relayer fee.
        let relayer_fee = 100000;

        // Call `transfer_tokens`.
        let (_, dust) = transfer_tokens::transfer_tokens<COIN_NATIVE_10>(
            &mut token_bridge_state,
            &mut worm_state,
            test_coins,
            coin::mint_for_testing(wormhole_fee, test_scenario::ctx(scenario)),
            TEST_TARGET_CHAIN,
            external_address::from_address(TEST_TARGET_RECIPIENT),
            relayer_fee,
            TEST_NONCE,
            &the_clock
        );
        assert!(coin::value(&dust) == 0, 0);

        // Done.
        coin::destroy_zero(dust);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = token_registry::E_UNREGISTERED)]
    fun test_cannot_transfer_tokens_wrapped_not_registered() {
        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Initialize COIN_WRAPPED_7 (but don't register it).
        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // NOTE: This test purposely doesn't `attest` COIN_WRAPPED_7.
        let transfer_amount = 42069;
        let test_coins = coin::mint_for_testing<COIN_WRAPPED_7>(
            transfer_amount,
            test_scenario::ctx(scenario)
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Define the relayer fee.
        let relayer_fee = 1000;

        // Call `transfer_tokens`.
        let (_, dust) = transfer_tokens::transfer_tokens<COIN_WRAPPED_7>(
            &mut token_bridge_state,
            &mut worm_state,
            test_coins,
            coin::mint_for_testing(wormhole_fee, test_scenario::ctx(scenario)),
            TEST_TARGET_CHAIN,
            external_address::from_address(TEST_TARGET_RECIPIENT),
            relayer_fee,
            TEST_NONCE,
            &the_clock
        );
        assert!(coin::value(&dust) == 0, 0);

        // Done.
        coin::destroy_zero(dust);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = transfer_tokens::E_RELAYER_FEE_EXCEEDS_AMOUNT)]
    fun test_cannot_transfer_tokens_fee_exceeds_amount() {
        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // NOTE: The `relayer_fee` is intentionally set to a higher number
        // than the `transfer_amount`.
        let relayer_fee = 100001;
        let transfer_amount = 100000;
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

        // The `transfer_tokens` call should revert.
        let (_, dust) = transfer_tokens::transfer_tokens<COIN_NATIVE_10>(
            &mut token_bridge_state,
            &mut worm_state,
            coin::from_balance(coin_10_balance, ctx),
            coin::mint_for_testing(wormhole_fee, ctx),
            TEST_TARGET_CHAIN,
            external_address::from_address(TEST_TARGET_RECIPIENT),
            relayer_fee,
            TEST_NONCE,
            &the_clock
        );
        assert!(coin::value(&dust) == 0, 0);

        // Done.
        coin::destroy_zero(dust);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);
        test_scenario::end(my_scenario);
    }
}
