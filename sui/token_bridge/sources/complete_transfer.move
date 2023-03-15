module token_bridge::complete_transfer {
    use sui::balance::{Self, Balance};
    use sui::coin::{Self};
    use sui::tx_context::{Self, TxContext};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::normalized_amount::{Self, NormalizedAmount};
    use token_bridge::state::{Self, State};
    use token_bridge::transfer::{Self, Transfer};
    use token_bridge::vaa::{Self};

    // Requires `handle_complete_transfer`.
    friend token_bridge::complete_transfer_with_payload;

    const E_INVALID_TARGET: u64 = 0;
    const E_UNREGISTERED_TOKEN: u64 = 1;

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
        ctx: &mut TxContext
    ): Balance<CoinType> {
        // Parse and verify Token Bridge transfer message. This method
        // guarantees that a verified transfer message cannot be redeemed again.
        let parsed_vaa =
            vaa::parse_verify_and_consume(
                token_bridge_state,
                worm_state,
                vaa_buf,
                ctx
            );

        // Deserialize transfer message and process.
        handle_complete_transfer<CoinType>(
            token_bridge_state,
            transfer::deserialize(wormhole::vaa::take_payload(parsed_vaa)),
            ctx
        )
    }

    /// `verify_and_take_coin` is only friendly with this module and the
    /// `complete_transfer` module. For inbound transfers, the deserialized
    /// transfer message needs to be validated.
    ///
    /// This method also de-normalizes the amount encoded in the transfer based
    /// on the coin's decimals.
    ///
    /// Depending on whether this coin is a Token Bridge wrapped asset or a
    /// natively existing asset on Sui, the coin is either minted or withdrawn
    /// from Token Bridge's custody.
    public(friend) fun verify_and_take_coin<CoinType>(
        token_bridge_state: &mut State,
        token_chain: u16,
        token_address: ExternalAddress,
        recipient_chain: u16,
        amount: NormalizedAmount
    ): (Balance<CoinType>, u8) {
        // Verify that the intended chain ID for this transfer is for Sui.
        assert!(
            recipient_chain == wormhole::state::chain_id(),
            E_INVALID_TARGET
        );

        // Verify that the token info agrees with the info encoded in this
        // transfer.
        state::assert_registered_token<CoinType>(
            token_bridge_state,
            token_chain,
            token_address
        );

        let decimals = state::coin_decimals<CoinType>(token_bridge_state);

        // If the token is wrapped by Token Bridge, we will mint these tokens.
        // Otherwise, we will withdraw from custody.
        let bridged =
            state::put_into_circulation(
                token_bridge_state,
                normalized_amount::to_raw(amount, decimals)
            );

        (bridged, decimals)
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

        let (bridged, decimals) =
            verify_and_take_coin<CoinType>(
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
            let raw_amount = normalized_amount::to_raw(relayer_fee, decimals);
            balance::split(&mut bridged, raw_amount)
        };

        // Finally transfer tokens to the recipient.
        sui::transfer::public_transfer(
            coin::from_balance(bridged, ctx),
            recipient
        );

        payout
    }
}

#[test_only]
module token_bridge::complete_transfer_test {
    use sui::balance::{Self};
    use sui::coin::{Self, Coin, CoinMetadata};
    use sui::test_scenario::{Self, Scenario, next_tx, return_shared,
        take_shared, ctx, take_from_address, return_to_address};
    use wormhole::state::{State as WormholeState};

    use token_bridge::coin_wrapped_12::{Self, COIN_WRAPPED_12};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};
    use token_bridge::complete_transfer::{Self};
    use token_bridge::coin_native_4::{Self, COIN_NATIVE_4};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::state::{Self, State};
    use token_bridge::token_bridge_scenario::{
        take_states,
        register_dummy_emitter,
        return_states
    };


    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    struct OTHER_COIN_WITNESS has drop {}

    /// Registration VAA for the etheruem token bridge 0xdeadbeef.
    const ETHEREUM_TOKEN_REG: vector<u8> =
        x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    /// Used for test_complete_native_transfer and test_complete_native_transfer_wrong_coin
    const VAA_NATIVE_TRANSFER: vector<u8> = x"010000000001002ff4303d38fe2eade48868ab51d31e21c32303d512501b26bfb6a3da9e9a41635ee41360458471d7b2af59b9b2cd48a11bea714e147e0ae76d0182e1af56bf41000000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f010000000000000000000000000000000000000000000000000000000000000bb8000000000000000000000000000000000000000000000000000000000000000100150000000000000000000000000000000000000000000000000000000000124323001500000000000000000000000000000000000000000000000000000000000003e8";
    // ============================ VAA details ============================
    // emitterChain: 2,
    // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    // module: 'TokenBridge',
    // type: 'Transfer',
    // amount: 3000n,
    // tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000001',
    // tokenChain: 21,
    // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
    // chain: 21,
    // fee: 1000n
    // ============================ VAA details ============================

    #[test]
    /// An end-to-end test for complete transer native with VAA.
    fun test_complete_native_transfer_10_decimals(){
        let (admin, tx_relayer, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        register_dummy_emitter(&mut test, 2);
        // Create coin.

        let mint_amount = 10000000000;
        coin_native_10::init_register_and_deposit(
            &mut test,
            admin,
            mint_amount
        );

        // Complete transfer, sending native tokens to a recipient address.
        next_tx(&mut test, tx_relayer); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let payout = complete_transfer::complete_transfer<COIN_NATIVE_10>(
                &mut bridge_state,
                &mut worm_state,
                VAA_NATIVE_TRANSFER,
                ctx(&mut test)
            );
            assert!(balance::value(&payout) == 100000, 0);
            balance::destroy_for_testing(payout);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<COIN_NATIVE_10>>(&test, admin);
            assert!(coin::value(&coins) == 200000, 0);
            return_to_address(admin, coins);
        };
        test_scenario::end(test);
    }

    #[test]
    /// An end-to-end test for complete transer native with VAA.
    fun test_complete_native_transfer_4_decimals(){
        {
        let (admin, tx_relayer, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        register_dummy_emitter(&mut test, 2);

        coin_native_4::init_register_and_deposit(&mut test, admin, 10000000000);

        // Complete transfer, sending native tokens to a recipient address.
        next_tx(&mut test, tx_relayer); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let payout = complete_transfer::complete_transfer<COIN_NATIVE_4>(
                &mut bridge_state,
                &worm_state,
                VAA_NATIVE_TRANSFER,
                ctx(&mut test)
            );
            assert!(balance::value(&payout) == 1000, 0);
            balance::destroy_for_testing(payout);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<COIN_NATIVE_4>>(&test, admin);
            assert!(coin::value<COIN_NATIVE_4>(&coins) == 2000, 0);
            return_to_address<Coin<COIN_NATIVE_4>>(admin, coins);
        };
        test_scenario::end(test);
    }
    }

    #[test]
    #[expected_failure(abort_code = state::E_CANONICAL_TOKEN_INFO_MISMATCH)]
    fun test_complete_native_transfer_wrong_origin_chain(){
        let wrong_origin_chain_vaa = x"01000000000100b0d67f0102856458dc68a29e88e5574f4b0b129841522066adbf275f0749129a319bc6a94a7b7c8822e91e1ceb2b1e22adee0d283fba5b97dab5f7165216785e010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f01000000000000000000000000000000000000000000000000000000003b9aca4f00000000000000000000000000000000000000000000000000000000000000010017000000000000000000000000000000000000000000000000000000000012432300150000000000000000000000000000000000000000000000000000000005f5e100";
        // ============================ VAA details ============================
        // emitterChain: 2,
        // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
        // module: 'TokenBridge',
        // type: 'Transfer',
        // amount: 1000000079n,
        // tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000001',
        // tokenChain: 23, // Wrong chain! Should be 21.
        // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
        // chain: 21,
        // fee: 100000000n
        // ============================ VAA details ============================
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        register_dummy_emitter(&mut test, 2);

        let mint_amount = 10000000000;
        coin_native_10::init_register_and_deposit(
            &mut test,
            admin,
            mint_amount
        );

        // Attempt complete transfer. Fails because the origin chain of the token
        // in the VAA is not specified correctly (though the address is the native-Sui
        // of the previously registered native token).
        test_scenario::next_tx(&mut test, admin);

        let (bridge_state, worm_state) = take_states(&test);

        let payout =
            complete_transfer::complete_transfer<COIN_NATIVE_10>(
                &mut bridge_state,
                &mut worm_state,
                wrong_origin_chain_vaa,
                test_scenario::ctx(&mut test)
            );
        balance::destroy_for_testing(payout);
        return_states(bridge_state, worm_state);

        // Done.
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = state::E_CANONICAL_TOKEN_INFO_MISMATCH)]
    fun test_complete_native_transfer_wrong_coin_address(){
        let vaa_transfer_wrong_address = x"010000000001008490d3e139f3b705282df4686907dfff358dd365e7471d8d6793ade61a27d33d48fc665198bc8022bebd8f8a91f29b3df75455180bf3b2d39eb97f93be3a8caf000000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f01000000000000000000000000000000000000000000000000000000003b9aca4f00000000000000000000000000000000000000000000000000000000000004440015000000000000000000000000000000000000000000000000000000000012432300150000000000000000000000000000000000000000000000000000000005f5e100";
        // ============================ VAA details ============================
        //   emitterChain: 2,
        //   emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
        //   sequence: 1n,
        //   consistencyLevel: 15,
        //   module: 'TokenBridge',
        //   type: 'Transfer',
        //   amount: 1000000079n,
        //   tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000444', // Wrong!
        //   tokenChain: 21,
        //   toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
        //   chain: 21,
        //   fee: 100000000n
        // ============================ VAA details ============================
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        register_dummy_emitter(&mut test, 2);

        let mint_amount = 10000000000;
        coin_native_10::init_register_and_deposit(
            &mut test,
            admin,
            mint_amount
        );

        // Ignore effects.
        test_scenario::next_tx(&mut test, admin);

        let (bridge_state, worm_state) = take_states(&test);

        // You shall not pass!
        let payout =
            complete_transfer::complete_transfer<COIN_NATIVE_10>(
                &mut bridge_state,
                &mut worm_state,
                vaa_transfer_wrong_address,
                ctx(&mut test)
            );
        balance::destroy_for_testing(payout);

        // Clean up.
        return_states(bridge_state, worm_state);

        // Done.
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = 2, location=sui::balance)] // E_TOO_MUCH_FEE
    fun test_complete_native_transfer_too_much_fee(){
        let vaa_transfer_too_much_fee = x"01000000000100032a439b4cf8f793e2a0b4281344bc81af9ff9f118ff9e320cbde49b072f23a5500963f2e66632143a97f2f0a5bc5370f006d2382f2e06c09b542476b02d099e000000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f01000000000000000000000000000000000000000000000000000000003b9aca000000000000000000000000000000000000000000000000000000000000000001001500000000000000000000000000000000000000000000000000000000001243230015000000000000000000000000000000000000000000000000000009184e72a000";
        // ============================ VAA details ============================
        //   emitterChain: 2,
        //   emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
        //   module: 'TokenBridge',
        //   type: 'Transfer',
        //   amount: 1000000000n,
        //   tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000001',
        //   tokenChain: 21,
        //   toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
        //   chain: 21,
        //   fee: 10000000000000n // Wrong! Way too much fee!
        // ============================ VAA details ============================
        let (admin, tx_relayer, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        register_dummy_emitter(&mut test, 2);
        next_tx(&mut test, admin);{
            coin_native_10::init_test_only(ctx(&mut test));
        };
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<COIN_NATIVE_10>>(&test);
            state::register_native_asset_test_only(
                &mut bridge_state,
                &coin_meta,
            );
            coin_native_10::init_test_only(ctx(&mut test));
            return_shared<CoinMetadata<COIN_NATIVE_10>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let minted = balance::create_for_testing<COIN_NATIVE_10>(10000000000);
            state::take_from_circulation_test_only<COIN_NATIVE_10>(&mut bridge_state, minted);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // attempt complete transfer
        next_tx(&mut test, tx_relayer); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let payout = complete_transfer::complete_transfer<COIN_NATIVE_10>(
                &mut bridge_state,
                &mut worm_state,
                vaa_transfer_too_much_fee,
                ctx(&mut test)
            );
            balance::destroy_for_testing(payout);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::registered_tokens::E_UNREGISTERED,
        location = token_bridge::registered_tokens
    )]
    /// In this test, the generic CoinType arg to complete_transfer
    /// is not specified correctly, causing the token bridge
    /// to not recignize that coin as being registered and for
    /// the complete transfer to fail.
    fun test_complete_native_transfer_wrong_coin(){
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        register_dummy_emitter(&mut test, 2);
        next_tx(&mut test, admin);{
            coin_native_10::init_test_only(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            coin_native_4::init_test_only(ctx(&mut test));
        };
        // Register native asset type COIN_NATIVE_10 with the token bridge.
        // Note that COIN_NATIVE_4 is not registered!
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<COIN_NATIVE_10>>(&test);
            state::register_native_asset_test_only(
                &mut bridge_state,
                &coin_meta,
            );
            coin_native_10::init_test_only(ctx(&mut test));
            return_shared<CoinMetadata<COIN_NATIVE_10>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let minted = balance::create_for_testing<COIN_NATIVE_10>(10000000000);
            state::take_from_circulation_test_only<COIN_NATIVE_10>(&mut bridge_state, minted);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Attempt complete transfer with wrong coin type (COIN_NATIVE_4).
        // Fails because COIN_NATIVE_4 is unregistered.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let payout = complete_transfer::complete_transfer<COIN_NATIVE_4>(
                &mut bridge_state,
                &mut worm_state,
                VAA_NATIVE_TRANSFER,
                ctx(&mut test)
            );
            // Cleaner than asserting that balance::value(&payout) == 0.
            balance::destroy_zero(payout);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }

    #[test]
    /// Test the external-facing function complete_transfer using a real VAA.
    /// Check balance of recipient (admin = @0x124323) after complete_transfer
    /// is called.
    fun complete_wrapped_transfer_external_no_fee_test(){
        // transfer token VAA (details below)
        let vaa = x"010000000001001d7c73fe9d1fd168fd8b4e767557724f82c56469560ccffd5f5dc6c49afd15007e27d1ed19a83ae11c68b28e848d67e8674ad29b19ee8d097c3ccc78ab292813010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f010000000000000000000000000000000000000000000000000000000000000bb800000000000000000000000000000000000000000000000000000000beefface0002000000000000000000000000000000000000000000000000000000000012432300150000000000000000000000000000000000000000000000000000000000000000";
        // ============================ VAA details ============================
        // emitterChain: 2,
        // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
        // module: 'TokenBridge',
        // type: 'Transfer',
        // amount: 3000n,
        // tokenAddress: '0x00000000000000000000000000000000000000000000000000000000beefface',
        // tokenChain: 2,
        // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
        // chain: 21,
        // fee: 0n
        // ============================ VAA details ============================
        let (admin, _fee_recipient_person, _) = people();
        let scenario = scenario();
        // First register foreign chain, create wrapped asset, register wrapped asset.
        let test = set_up_wormhole_core_and_token_bridges(admin, scenario);
        test_scenario::next_tx(&mut test, admin);

        let bridge_state = test_scenario::take_shared<State>(&test);
        state::register_new_emitter_test_only(
            &mut bridge_state,
            2,
            wormhole::external_address::from_any_bytes(x"deadbeef")
        );
        test_scenario::return_shared(bridge_state);
        coin_wrapped_12::init_and_register(&mut test, admin);

        // Complete transfer of wrapped asset from foreign chain to this chain.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let payout = complete_transfer::complete_transfer<COIN_WRAPPED_12>(
                &mut bridge_state,
                &mut worm_state,
                vaa,
                ctx(&mut test)
            );
            // Cleaner than asserting that balance::value(&payout) == 0.
            balance::destroy_zero(payout);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<COIN_WRAPPED_12>>(&test, admin);
            assert!(coin::value<COIN_WRAPPED_12>(&coins) == 3000, 0);
            return_to_address<Coin<COIN_WRAPPED_12>>(admin, coins);
        };
        test_scenario::end(test);
    }

    #[test]
    /// Test the external-facing function complete_transfer using a real VAA.
    /// This time include a relayer fee when calling complete_transfer.
    fun complete_wrapped_transfer_external_relayer_fee_test(){
        // transfer token VAA (details below)
        let vaa = x"01000000000100f90d171a2c4ffde9cf214ce2ed94b384e5ee8384ef3129e1342bf6b10db8201122fb9ff501e9f28367a48f191cf5fb8ff51ff58e8091745a392ec8053a05b55e010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f010000000000000000000000000000000000000000000000000000000000000bb800000000000000000000000000000000000000000000000000000000beefface00020000000000000000000000000000000000000000000000000000000000124323001500000000000000000000000000000000000000000000000000000000000003e8";
        // ============================ VAA details ============================
        // emitterChain: 2,
        // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
        // module: 'TokenBridge',
        // type: 'Transfer',
        // amount: 3000n,
        // tokenAddress: '0x00000000000000000000000000000000000000000000000000000000beefface',
        // tokenChain: 2,
        // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
        // chain: 21,
        // fee: 1000n
        // ============================ VAA details ============================
        let (admin, tx_relayer, _) = people();
        let scenario = scenario();
        // First register foreign chain, create wrapped asset, register wrapped asset.
        let test = set_up_wormhole_core_and_token_bridges(admin, scenario);
        test_scenario::next_tx(&mut test, admin);

        let bridge_state = test_scenario::take_shared<State>(&test);
        state::register_new_emitter_test_only(
            &mut bridge_state,
            2,
            wormhole::external_address::from_any_bytes(x"deadbeef")
        );
        test_scenario::return_shared(bridge_state);
        coin_wrapped_12::init_and_register(&mut test, admin);

        // Complete transfer of wrapped asset from foreign chain to this chain.
        next_tx(&mut test, tx_relayer); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let payout = complete_transfer::complete_transfer<COIN_WRAPPED_12>(
                &mut bridge_state,
                &mut worm_state,
                vaa,
                ctx(&mut test)
            );
            assert!(balance::value(&payout) == 1000, 0);
            balance::destroy_for_testing(payout);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<COIN_WRAPPED_12>>(&test, admin);
            assert!(coin::value<COIN_WRAPPED_12>(&coins) == 2000, 0);
            return_to_address<Coin<COIN_WRAPPED_12>>(admin, coins);
        };
        test_scenario::end(test);
    }
}
