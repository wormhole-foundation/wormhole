module token_bridge::complete_transfer {
    use sui::balance::{Self, Balance};
    use sui::clock::{Clock};
    use sui::coin::{Self};
    use sui::event::{Self};
    use sui::tx_context::{Self, TxContext};
    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::vaa::{VAA};

    use token_bridge::normalized_amount::{Self, NormalizedAmount};
    use token_bridge::state::{Self, State};
    use token_bridge::token_registry::{Self};
    use token_bridge::transfer::{Self, Transfer};
    use token_bridge::vaa::{Self};
    use token_bridge::version_control::{
        CompleteTransfer as CompleteTransferControl
    };

    // Requires `handle_complete_transfer`.
    friend token_bridge::complete_transfer_with_payload;

    const E_TARGET_NOT_SUI: u64 = 0;
    const E_UNREGISTERED_TOKEN: u64 = 1;

    struct TransferRedeemed has drop, copy {
        emitter_chain: u16,
        emitter_address: vector<u8>,
        sequence: u64
    }

    /// `complete_transfer` takes a verified Wormhole message and validates
    /// that this message was sent by a registered foreign Token Bridge contract
    /// and has a Token Bridge transfer payload.
    ///
    /// After processing the token transfer payload, coins are sent to the
    /// encoded recipient. If the specified `relayer` differs from this
    /// recipient, a relayer fee is split from this coin and sent to `relayer`.
    public fun complete_transfer<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        the_clock: &Clock,
        ctx: &mut TxContext
    ): Balance<CoinType> {
        state::check_minimum_requirement<CompleteTransferControl>(
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

        // Emitting the transfer being redeemed (and disregard return value).
        emit_transfer_redeemed(&parsed_vaa);

        // Deserialize transfer message and process.
        handle_complete_transfer<CoinType>(
            token_bridge_state,
            transfer::deserialize(wormhole::vaa::take_payload(parsed_vaa)),
            ctx
        )
    }

    /// `verify_and_bridge_out` is only friendly with this module and the
    /// `complete_transfer` module. For inbound transfers, the deserialized
    /// transfer message needs to be validated.
    ///
    /// This method also de-normalizes the amount encoded in the transfer based
    /// on the coin's decimals.
    ///
    /// Depending on whether this coin is a Token Bridge wrapped asset or a
    /// natively existing asset on Sui, the coin is either minted or withdrawn
    /// from Token Bridge's custody.
    public(friend) fun verify_and_bridge_out<CoinType>(
        token_bridge_state: &mut State,
        token_chain: u16,
        token_address: ExternalAddress,
        target_chain: u16,
        amount: NormalizedAmount
    ): (Balance<CoinType>, u8) {
        // Verify that the intended chain ID for this transfer is for Sui.
        assert!(
            target_chain == wormhole::state::chain_id(),
            E_TARGET_NOT_SUI
        );

        let registry = state::borrow_token_registry_mut(token_bridge_state);
        let verified =
            token_registry::verify_for_asset_cap<CoinType>(
                registry,
                token_chain,
                token_address
            );
        let decimals = token_registry::checked_decimals(&verified, registry);

        // If the token is wrapped by Token Bridge, we will mint these tokens.
        // Otherwise, we will withdraw from custody.
        let bridged_out =
            token_registry::put_into_circulation(
                &verified,
                registry,
                normalized_amount::to_raw(amount, decimals)
            );

        (bridged_out, decimals)
    }

    public(friend) fun emit_transfer_redeemed(parsed_vaa: &VAA): u16 {
        let (
            emitter_chain,
            emitter_address,
            sequence
        ) = wormhole::vaa::emitter_info(parsed_vaa);

        // Emit Sui event with `TransferRedeemed`.
        event::emit(
            TransferRedeemed {
                emitter_chain,
                emitter_address: external_address::to_bytes(emitter_address),
                sequence
            }
        );

        emitter_chain
    }

    fun handle_complete_transfer<CoinType>(
        token_bridge_state: &mut State,
        parsed_transfer: Transfer,
        ctx: &mut TxContext
    ): Balance<CoinType> {
        let (
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            relayer_fee
        ) = transfer::unpack(parsed_transfer);

        let (
            bridged_out,
            decimals
        ) =
            verify_and_bridge_out(
                token_bridge_state,
                token_chain,
                token_address,
                recipient_chain,
                amount
            );

        let recipient = external_address::to_address(recipient);

        // If the recipient did not redeem his own transfer, Token Bridge will
        // split the withdrawn coins and send a portion to the transaction
        // relayer.
        let payout = if (
            normalized_amount::value(&relayer_fee) == 0 ||
            recipient == tx_context::sender(ctx)
        ) {
            balance::zero()
        } else {
            balance::split(
                &mut bridged_out,
                normalized_amount::to_raw(relayer_fee, decimals)
            )
        };

        // Finally transfer tokens to the recipient.
        sui::transfer::public_transfer(
            coin::from_balance(bridged_out, ctx),
            recipient
        );

        payout
    }
}

#[test_only]
module token_bridge::complete_transfer_tests {
    use sui::balance::{Self};
    use sui::coin::{Self, Coin};
    use sui::test_scenario::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::coin_wrapped_12::{Self, COIN_WRAPPED_12};
    use token_bridge::coin_wrapped_7::{Self, COIN_WRAPPED_7};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::coin_native_4::{Self, COIN_NATIVE_4};
    use token_bridge::complete_transfer::{Self};
    use token_bridge::dummy_message::{Self};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        set_up_wormhole_and_token_bridge,
        register_dummy_emitter,
        return_clock,
        return_states,
        take_clock,
        take_states,
        three_people
    };
    use token_bridge::token_registry::{Self};
    use token_bridge::transfer::{Self};

    struct OTHER_COIN_WITNESS has drop {}

    #[test]
    /// An end-to-end test for complete transer native with VAA.
    fun test_complete_transfer_native_10_relayer_fee() {
        let transfer_vaa =
            dummy_message::encoded_transfer_vaa_native_with_fee();

        let (expected_recipient, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        let custody_amount = 500000;
        coin_native_10::init_register_and_deposit(
            scenario,
            coin_deployer,
            custody_amount
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // These will be checked later.
        let expected_relayer_fee = 100000;
        let expected_recipient_amount = 200000;
        let expected_amount = expected_relayer_fee + expected_recipient_amount;

        // Scope to allow immutable reference to `TokenRegistry`.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            assert!(
                token_registry::native_balance<COIN_NATIVE_10>(registry) == custody_amount,
                0
            );

            // Verify transfer parameters.
            let parsed =
                transfer::deserialize(
                    wormhole::vaa::take_payload(
                        wormhole::vaa::parse_and_verify(
                            &worm_state,
                            transfer_vaa,
                            &the_clock
                        )
                    )
                );

            let (
                expected_token_chain,
                expected_token_address
            ) = token_registry::canonical_info<COIN_NATIVE_10>(registry);
            assert!(transfer::token_chain(&parsed) == expected_token_chain, 0);
            assert!(transfer::token_address(&parsed) == expected_token_address, 0);

            let decimals =
                state::coin_decimals<COIN_NATIVE_10>(&token_bridge_state);

            assert!(transfer::raw_amount(&parsed, decimals) == expected_amount, 0);

            assert!(
                transfer::raw_relayer_fee(&parsed, decimals) == expected_relayer_fee,
                0
            );
            assert!(
                transfer::recipient_as_address(&parsed) == expected_recipient,
                0
            );
            assert!(transfer::recipient_chain(&parsed) == chain_id(), 0);

            // Clean up.
            transfer::destroy(parsed);
        };

        let payout =
            complete_transfer::complete_transfer<COIN_NATIVE_10>(
                &mut token_bridge_state,
                &mut worm_state,
                transfer_vaa,
                &the_clock,
                test_scenario::ctx(scenario)
            );
        assert!(balance::value(&payout) == expected_relayer_fee, 0);

        // TODO: Check for one event? `TransferRedeemed`.
        let _effects = test_scenario::next_tx(scenario, tx_relayer);

        // Check recipient's `Coin`.
        let received =
            test_scenario::take_from_address<Coin<COIN_NATIVE_10>>(
                scenario,
                expected_recipient
            );
        assert!(coin::value(&received) == expected_recipient_amount, 0);

        // And check remaining amount in custody.
        let registry = state::borrow_token_registry(&token_bridge_state);
        let remaining = custody_amount - expected_amount;
        assert!(
            token_registry::native_balance<COIN_NATIVE_10>(registry) == remaining,
            0
        );

        // Clean up.
        balance::destroy_for_testing(payout);
        coin::burn_for_testing(received);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    /// An end-to-end test for complete transfer native with VAA.
    fun test_complete_transfer_native_4_relayer_fee() {
        let transfer_vaa =
            dummy_message::encoded_transfer_vaa_native_with_fee();

        let (expected_recipient, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        let custody_amount = 5000;
        coin_native_4::init_register_and_deposit(
            scenario,
            coin_deployer,
            custody_amount
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // These will be checked later.
        let expected_relayer_fee = 1000;
        let expected_recipient_amount = 2000;
        let expected_amount = expected_relayer_fee + expected_recipient_amount;

        // Scope to allow immutable reference to `TokenRegistry`.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            assert!(
                token_registry::native_balance<COIN_NATIVE_4>(registry) == custody_amount,
                0
            );

            // Verify transfer parameters.
            let parsed =
                transfer::deserialize(
                    wormhole::vaa::take_payload(
                        wormhole::vaa::parse_and_verify(
                            &worm_state,
                            transfer_vaa,
                            &the_clock
                        )
                    )
                );

            let (
                expected_token_chain,
                expected_token_address
            ) = token_registry::canonical_info<COIN_NATIVE_4>(registry);
            assert!(transfer::token_chain(&parsed) == expected_token_chain, 0);
            assert!(transfer::token_address(&parsed) == expected_token_address, 0);

            let decimals =
                state::coin_decimals<COIN_NATIVE_4>(&token_bridge_state);

            assert!(transfer::raw_amount(&parsed, decimals) == expected_amount, 0);

            assert!(
                transfer::raw_relayer_fee(&parsed, decimals) == expected_relayer_fee,
                0
            );
            assert!(
                transfer::recipient_as_address(&parsed) == expected_recipient,
                0
            );
            assert!(transfer::recipient_chain(&parsed) == chain_id(), 0);

            // Clean up.
            transfer::destroy(parsed);
        };

        let payout =
            complete_transfer::complete_transfer<COIN_NATIVE_4>(
                &mut token_bridge_state,
                &mut worm_state,
                transfer_vaa,
                &the_clock,
                test_scenario::ctx(scenario)
            );
        assert!(balance::value(&payout) == expected_relayer_fee, 0);

        // TODO: Check for one event? `TransferRedeemed`.
        let _effects = test_scenario::next_tx(scenario, tx_relayer);

        // Check recipient's `Coin`.
        let received =
            test_scenario::take_from_address<Coin<COIN_NATIVE_4>>(
                scenario,
                expected_recipient
            );
        assert!(coin::value(&received) == expected_recipient_amount, 0);

        // And check remaining amount in custody.
        let registry = state::borrow_token_registry(&token_bridge_state);
        let remaining = custody_amount - expected_amount;
        assert!(
            token_registry::native_balance<COIN_NATIVE_4>(registry) == remaining,
            0
        );

        // Clean up.
        balance::destroy_for_testing(payout);
        coin::burn_for_testing(received);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    /// An end-to-end test for complete transfer wrapped with VAA.
    fun test_complete_transfer_wrapped_7_relayer_fee() {
        let transfer_vaa = dummy_message::encoded_transfer_vaa_wrapped_7_with_fee();

        let (expected_recipient, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        coin_wrapped_7::init_and_register(scenario, coin_deployer);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // These will be checked later.
        let expected_relayer_fee = 1000;
        let expected_recipient_amount = 2000;
        let expected_amount = expected_relayer_fee + expected_recipient_amount;

        // Scope to allow immutable reference to `TokenRegistry`.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            assert!(
                token_registry::wrapped_supply<COIN_WRAPPED_7>(registry) == 0,
                0
            );

            // Verify transfer parameters.
            let parsed =
                transfer::deserialize(
                    wormhole::vaa::take_payload(
                        wormhole::vaa::parse_and_verify(
                            &worm_state,
                            transfer_vaa,
                            &the_clock
                        )
                    )
                );

            let (
                expected_token_chain,
                expected_token_address
            ) = token_registry::canonical_info<COIN_WRAPPED_7>(registry);
            assert!(transfer::token_chain(&parsed) == expected_token_chain, 0);
            assert!(transfer::token_address(&parsed) == expected_token_address, 0);

            let decimals =
                state::coin_decimals<COIN_WRAPPED_7>(&token_bridge_state);

            assert!(transfer::raw_amount(&parsed, decimals) == expected_amount, 0);

            assert!(
                transfer::raw_relayer_fee(&parsed, decimals) == expected_relayer_fee,
                0
            );
            assert!(
                transfer::recipient_as_address(&parsed) == expected_recipient,
                0
            );
            assert!(transfer::recipient_chain(&parsed) == chain_id(), 0);

            // Clean up.
            transfer::destroy(parsed);
        };

        let payout =
            complete_transfer::complete_transfer<COIN_WRAPPED_7>(
                &mut token_bridge_state,
                &mut worm_state,
                transfer_vaa,
                &the_clock,
                test_scenario::ctx(scenario)
            );
        assert!(balance::value(&payout) == expected_relayer_fee, 0);

        // TODO: Check for one event? `TransferRedeemed`.
        let _effects = test_scenario::next_tx(scenario, tx_relayer);

        // Check recipient's `Coin`.
        let received =
            test_scenario::take_from_address<Coin<COIN_WRAPPED_7>>(
                scenario,
                expected_recipient
            );
        assert!(coin::value(&received) == expected_recipient_amount, 0);

        // And check that the amount is the total wrapped supply.
        let registry = state::borrow_token_registry(&token_bridge_state);
        assert!(
            token_registry::wrapped_supply<COIN_WRAPPED_7>(registry) == expected_amount,
            0
        );

        // Clean up.
        balance::destroy_for_testing(payout);
        coin::burn_for_testing(received);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    /// An end-to-end test for complete transfer wrapped with VAA.
    fun test_complete_transfer_wrapped_12_relayer_fee() {
        let transfer_vaa = dummy_message::encoded_transfer_vaa_wrapped_12_with_fee();

        let (expected_recipient, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        coin_wrapped_12::init_and_register(scenario, coin_deployer);

        // Ignore effects.
        //
        // NOTE: `tx_relayer` != `expected_recipient`.
        assert!(expected_recipient != tx_relayer, 0);
        test_scenario::next_tx(scenario, tx_relayer);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // These will be checked later.
        let expected_relayer_fee = 1000;
        let expected_recipient_amount = 2000;
        let expected_amount = expected_relayer_fee + expected_recipient_amount;

        // Scope to allow immutable reference to `TokenRegistry`.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            assert!(
                token_registry::wrapped_supply<COIN_WRAPPED_12>(registry) == 0,
                0
            );

            // Verify transfer parameters.
            let parsed =
                transfer::deserialize(
                    wormhole::vaa::take_payload(
                        wormhole::vaa::parse_and_verify(
                            &worm_state,
                            transfer_vaa,
                            &the_clock
                        )
                    )
                );

            let (
                expected_token_chain,
                expected_token_address
            ) = token_registry::canonical_info<COIN_WRAPPED_12>(registry);
            assert!(transfer::token_chain(&parsed) == expected_token_chain, 0);
            assert!(transfer::token_address(&parsed) == expected_token_address, 0);

            let decimals =
                state::coin_decimals<COIN_WRAPPED_12>(&token_bridge_state);

            assert!(transfer::raw_amount(&parsed, decimals) == expected_amount, 0);

            assert!(
                transfer::raw_relayer_fee(&parsed, decimals) == expected_relayer_fee,
                0
            );
            assert!(
                transfer::recipient_as_address(&parsed) == expected_recipient,
                0
            );
            assert!(transfer::recipient_chain(&parsed) == chain_id(), 0);

            // Clean up.
            transfer::destroy(parsed);
        };

        let payout = complete_transfer::complete_transfer<COIN_WRAPPED_12>(
            &mut token_bridge_state,
            &mut worm_state,
            transfer_vaa,
            &the_clock,
            test_scenario::ctx(scenario)
        );
        assert!(balance::value(&payout) == expected_relayer_fee, 0);

        // TODO: Check for one event? `TransferRedeemed`.
        let _effects = test_scenario::next_tx(scenario, tx_relayer);

        // Check recipient's `Coin`.
        let received =
            test_scenario::take_from_address<Coin<COIN_WRAPPED_12>>(
                scenario,
                expected_recipient
            );
        assert!(coin::value(&received) == expected_recipient_amount, 0);

        // And check that the amount is the total wrapped supply.
        let registry = state::borrow_token_registry(&token_bridge_state);
        assert!(
            token_registry::wrapped_supply<COIN_WRAPPED_12>(registry) == expected_amount,
            0
        );

        // Clean up.
        balance::destroy_for_testing(payout);
        coin::burn_for_testing(received);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    // #[test]
    // #[expected_failure(abort_code = state::E_CANONICAL_TOKEN_INFO_MISMATCH)]
    // fun test_complete_native_transfer_wrong_origin_chain(){
    //     let wrong_origin_chain_vaa = x"01000000000100b0d67f0102856458dc68a29e88e5574f4b0b129841522066adbf275f0749129a319bc6a94a7b7c8822e91e1ceb2b1e22adee0d283fba5b97dab5f7165216785e010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f01000000000000000000000000000000000000000000000000000000003b9aca4f00000000000000000000000000000000000000000000000000000000000000010017000000000000000000000000000000000000000000000000000000000012432300150000000000000000000000000000000000000000000000000000000005f5e100";
    //     // ============================ VAA details ============================
    //     // emitterChain: 2,
    //     // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    //     // module: 'TokenBridge',
    //     // type: 'Transfer',
    //     // amount: 1000000079n,
    //     // tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000001',
    //     // tokenChain: 23, // Wrong chain! Should be 21.
    //     // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
    //     // chain: 21,
    //     // fee: 100000000n
    //     // ============================ VAA details ============================
    //     let (caller, _, _) = people();
    //     let test = scenario();
    //     test = set_up_wormhole_core_and_token_bridges(caller, test);
    //     register_dummy_emitter(scenario, 2);

    //     let mint_amount = 10000000000;
    //     coin_native_10::init_register_and_deposit(
    //         scenario,
    //         caller,
    //         mint_amount
    //     );

    //     // Attempt complete transfer. Fails because the origin chain of the token
    //     // in the VAA is not specified correctly (though the address is the native-Sui
    //     // of the previously registered native token).
    //     test_scenario::test_scenario::next_tx(scenario, caller);

    //     let (token_bridge_state, worm_state) = take_states(scenario);

    //     let payout =
    //         complete_transfer::complete_transfer<COIN_NATIVE_10>(
    //             &mut token_bridge_state,
    //             &mut worm_state,
    //             wrong_origin_chain_vaa,
    //             test_scenario::test_scenario::ctx(scenario)
    //         );
    //     balance::destroy_for_testing(payout);
    //     return_states(token_bridge_state, worm_state);

    //     // Done.
    //     test_scenario::end(test);
    // }

    // #[test]
    // #[expected_failure(abort_code = state::E_CANONICAL_TOKEN_INFO_MISMATCH)]
    // fun test_complete_native_transfer_wrong_coin_address(){
    //     let vaa_transfer_wrong_address = x"010000000001008490d3e139f3b705282df4686907dfff358dd365e7471d8d6793ade61a27d33d48fc665198bc8022bebd8f8a91f29b3df75455180bf3b2d39eb97f93be3a8caf000000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f01000000000000000000000000000000000000000000000000000000003b9aca4f00000000000000000000000000000000000000000000000000000000000004440015000000000000000000000000000000000000000000000000000000000012432300150000000000000000000000000000000000000000000000000000000005f5e100";
    //     // ============================ VAA details ============================
    //     //   emitterChain: 2,
    //     //   emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    //     //   sequence: 1n,
    //     //   consistencyLevel: 15,
    //     //   module: 'TokenBridge',
    //     //   type: 'Transfer',
    //     //   amount: 1000000079n,
    //     //   tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000444', // Wrong!
    //     //   tokenChain: 21,
    //     //   toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
    //     //   chain: 21,
    //     //   fee: 100000000n
    //     // ============================ VAA details ============================
    //     let (caller, _, _) = people();
    //     let test = scenario();
    //     test = set_up_wormhole_core_and_token_bridges(caller, test);
    //     register_dummy_emitter(scenario, 2);

    //     let mint_amount = 10000000000;
    //     coin_native_10::init_register_and_deposit(
    //         scenario,
    //         caller,
    //         mint_amount
    //     );

    //     // Ignore effects.
    //     test_scenario::test_scenario::next_tx(scenario, caller);

    //     let (token_bridge_state, worm_state) = take_states(scenario);

    //     // You shall not pass!
    //     let payout =
    //         complete_transfer::complete_transfer<COIN_NATIVE_10>(
    //             &mut token_bridge_state,
    //             &mut worm_state,
    //             vaa_transfer_wrong_address,
    //             test_scenario::ctx(scenario)
    //         );
    //     balance::destroy_for_testing(payout);

    //     // Clean up.
    //     return_states(token_bridge_state, worm_state);

    //     // Done.
    //     test_scenario::end(test);
    // }


    // #[test]
    // #[expected_failure(
    //     abort_code = token_bridge::token_registry::E_UNREGISTERED,
    //     location = token_bridge::token_registry
    // )]
    // /// In this test, the generic CoinType arg to complete_transfer
    // /// is not specified correctly, causing the token bridge
    // /// to not recignize that coin as being registered and for
    // /// the complete transfer to fail.
    // fun test_complete_native_transfer_wrong_coin(){
    //     let (caller, _, _) = people();
    //     let test = scenario();
    //     test = set_up_wormhole_core_and_token_bridges(caller, test);
    //     register_dummy_emitter(scenario, 2);
    //     test_scenario::next_tx(scenario, caller);{
    //         coin_native_10::init_test_only(test_scenario::ctx(scenario));
    //     };
    //     test_scenario::next_tx(scenario, caller);{
    //         coin_native_4::init_test_only(test_scenario::ctx(scenario));
    //     };
    //     // Register native asset type COIN_NATIVE_10 with the token bridge.
    //     // Note that COIN_NATIVE_4 is not registered!
    //     test_scenario::next_tx(scenario, caller);{
    //         let token_bridge_state = take_shared<State>(scenario);
    //         let worm_state = take_shared<WormholeState>(scenario);
    //         let coin_meta = take_shared<CoinMetadata<COIN_NATIVE_10>>(scenario);
    //         state::register_native_asset_test_only(
    //             &mut token_bridge_state,
    //             &coin_meta,
    //         );
    //         coin_native_10::init_test_only(test_scenario::ctx(scenario));
    //         return_shared<CoinMetadata<COIN_NATIVE_10>>(coin_meta);
    //         return_shared<State>(token_bridge_state);
    //         return_shared<WormholeState>(worm_state);
    //     };
    //     // Create a treasury cap for the native asset type, mint some tokens,
    //     // and deposit the native tokens into the token bridge.
    //     test_scenario::next_tx(scenario, caller); {
    //         let token_bridge_state = take_shared<State>(scenario);
    //         let worm_state = take_shared<WormholeState>(scenario);
    //         let minted = balance::create_for_testing<COIN_NATIVE_10>(10000000000);
    //         state::take_from_circulation_test_only<COIN_NATIVE_10>(&mut token_bridge_state, minted);
    //         return_shared<State>(token_bridge_state);
    //         return_shared<WormholeState>(worm_state);
    //     };
    //     // Attempt complete transfer with wrong coin type (COIN_NATIVE_4).
    //     // Fails because COIN_NATIVE_4 is unregistered.
    //     test_scenario::next_tx(scenario, caller); {
    //         let token_bridge_state = take_shared<State>(scenario);
    //         let worm_state = take_shared<WormholeState>(scenario);

    //         let payout = complete_transfer::complete_transfer<COIN_NATIVE_4>(
    //             &mut token_bridge_state,
    //             &mut worm_state,
    //             VAA_NATIVE_TRANSFER,
    //             test_scenario::ctx(scenario)
    //         );
    //         // Cleaner than asserting that balance::value(&payout) == 0.
    //         balance::destroy_zero(payout);
    //         return_shared<State>(token_bridge_state);
    //         return_shared<WormholeState>(worm_state);
    //     };
    //     test_scenario::end(test);
    // }

    // #[test]
    // /// Test the external-facing function complete_transfer using a real VAA.
    // /// Check balance of recipient (caller = @0x124323) after complete_transfer
    // /// is called.
    // fun complete_wrapped_transfer_external_no_fee_test(){
    //     // transfer token VAA (details below)
    //     let vaa = x"010000000001001d7c73fe9d1fd168fd8b4e767557724f82c56469560ccffd5f5dc6c49afd15007e27d1ed19a83ae11c68b28e848d67e8674ad29b19ee8d097c3ccc78ab292813010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f010000000000000000000000000000000000000000000000000000000000000bb800000000000000000000000000000000000000000000000000000000beefface0002000000000000000000000000000000000000000000000000000000000012432300150000000000000000000000000000000000000000000000000000000000000000";
    //     // ============================ VAA details ============================
    //     // emitterChain: 2,
    //     // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    //     // module: 'TokenBridge',
    //     // type: 'Transfer',
    //     // amount: 3000n,
    //     // tokenAddress: '0x00000000000000000000000000000000000000000000000000000000beefface',
    //     // tokenChain: 2,
    //     // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
    //     // chain: 21,
    //     // fee: 0n
    //     // ============================ VAA details ============================
    //     let (caller, _fee_recipient_person, _) = people();
    //     let scenario = scenario();
    //     // First register foreign chain, create wrapped asset, register wrapped asset.
    //     let test = set_up_wormhole_core_and_token_bridges(caller, scenario);
    //     test_scenario::test_scenario::next_tx(scenario, caller);

    //     let token_bridge_state = test_scenario::take_shared<State>(scenario);
    //     state::register_new_emitter_test_only(
    //         &mut token_bridge_state,
    //         2,
    //         wormhole::external_address::from_any_bytes(x"deadbeef")
    //     );
    //     test_scenario::return_shared(token_bridge_state);
    //     coin_wrapped_12::init_and_register(scenario, caller);

    //     // Complete transfer of wrapped asset from foreign chain to this chain.
    //     test_scenario::next_tx(scenario, caller); {
    //         let token_bridge_state = take_shared<State>(scenario);
    //         let worm_state = take_shared<WormholeState>(scenario);

    //         let payout = complete_transfer::complete_transfer<COIN_WRAPPED_12>(
    //             &mut token_bridge_state,
    //             &mut worm_state,
    //             vaa,
    //             test_scenario::ctx(scenario)
    //         );
    //         // Cleaner than asserting that balance::value(&payout) == 0.
    //         balance::destroy_zero(payout);
    //         return_shared<State>(token_bridge_state);
    //         return_shared<WormholeState>(worm_state);
    //     };

    //     // Check balances after.
    //     test_scenario::next_tx(scenario, caller);{
    //         let coins = take_from_address<Coin<COIN_WRAPPED_12>>(scenario, caller);
    //         assert!(coin::value<COIN_WRAPPED_12>(&coins) == 3000, 0);
    //         return_to_address<Coin<COIN_WRAPPED_12>>(caller, coins);
    //     };
    //     test_scenario::end(test);
    // }

    // #[test]
    // /// Test the external-facing function complete_transfer using a real VAA.
    // /// This time include a relayer fee when calling complete_transfer.
    // fun complete_wrapped_transfer_external_relayer_fee_test(){
    //     // transfer token VAA (details below)
    //     let vaa = x"01000000000100f90d171a2c4ffde9cf214ce2ed94b384e5ee8384ef3129e1342bf6b10db8201122fb9ff501e9f28367a48f191cf5fb8ff51ff58e8091745a392ec8053a05b55e010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f010000000000000000000000000000000000000000000000000000000000000bb800000000000000000000000000000000000000000000000000000000beefface00020000000000000000000000000000000000000000000000000000000000124323001500000000000000000000000000000000000000000000000000000000000003e8";
    //     // ============================ VAA details ============================
    //     // emitterChain: 2,
    //     // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    //     // module: 'TokenBridge',
    //     // type: 'Transfer',
    //     // amount: 3000n,
    //     // tokenAddress: '0x00000000000000000000000000000000000000000000000000000000beefface',
    //     // tokenChain: 2,
    //     // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
    //     // chain: 21,
    //     // fee: 1000n
    //     // ============================ VAA details ============================
    //     let (caller, tx_relayer, _) = people();
    //     let scenario = scenario();
    //     // First register foreign chain, create wrapped asset, register wrapped asset.
    //     let test = set_up_wormhole_core_and_token_bridges(caller, scenario);
    //     test_scenario::test_scenario::next_tx(scenario, caller);

    //     let token_bridge_state = test_scenario::take_shared<State>(scenario);
    //     state::register_new_emitter_test_only(
    //         &mut token_bridge_state,
    //         2,
    //         wormhole::external_address::from_any_bytes(x"deadbeef")
    //     );
    //     test_scenario::return_shared(token_bridge_state);
    //     coin_wrapped_12::init_and_register(scenario, caller);

    //     // Complete transfer of wrapped asset from foreign chain to this chain.
    //     test_scenario::next_tx(scenario, tx_relayer); {
    //         let token_bridge_state = take_shared<State>(scenario);
    //         let worm_state = take_shared<WormholeState>(scenario);

    //         let payout = complete_transfer::complete_transfer<COIN_WRAPPED_12>(
    //             &mut token_bridge_state,
    //             &mut worm_state,
    //             vaa,
    //             test_scenario::ctx(scenario)
    //         );
    //         assert!(balance::value(&payout) == 1000, 0);
    //         balance::destroy_for_testing(payout);
    //         return_shared<State>(token_bridge_state);
    //         return_shared<WormholeState>(worm_state);
    //     };

    //     // Check balances after.
    //     test_scenario::next_tx(scenario, caller);{
    //         let coins = take_from_address<Coin<COIN_WRAPPED_12>>(scenario, caller);
    //         assert!(coin::value<COIN_WRAPPED_12>(&coins) == 2000, 0);
    //         return_to_address<Coin<COIN_WRAPPED_12>>(caller, coins);
    //     };
    //     test_scenario::end(test);
    // }
}
