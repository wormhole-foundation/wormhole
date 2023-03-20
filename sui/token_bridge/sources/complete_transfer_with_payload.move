module token_bridge::complete_transfer_with_payload {
    use sui::tx_context::{TxContext};
    use sui::coin::{Coin};

    use wormhole::state::{State as WormholeState};
    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::vaa::{emitter_chain};

    use token_bridge::complete_transfer::{verify_transfer_details};
    use token_bridge::state::{State};
    use token_bridge::transfer_with_payload::{Self, TransferWithPayload};
    use token_bridge::vaa::{Self};

    const E_INVALID_TARGET: u64 = 0;
    const E_INVALID_REDEEMER: u64 = 1;

    public fun complete_transfer_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        worm_state: &mut WormholeState,
        vaa: vector<u8>,
        ctx: &mut TxContext
    ): (Coin<CoinType>, TransferWithPayload, u16) {
        // Parse and verify Token Bridge transfer message. This method
        // guarantees that a verified transfer message cannot be redeemed again.
        let transfer_vaa =
            vaa::parse_verify_and_replay_protect(
                token_bridge_state,
                worm_state,
                vaa,
                ctx
            );

        // Before destroying VAA, store the emitter chain ID for the caller.
        let source_chain = emitter_chain(&transfer_vaa);

        // Deserialize for processing.
        let parsed_transfer =
            transfer_with_payload::deserialize(
                wormhole::vaa::take_payload(transfer_vaa)
            );
        let token_coin =
            handle_complete_transfer_with_payload(
                token_bridge_state,
                emitter_cap,
                &parsed_transfer,
                ctx
            );

        (token_coin, parsed_transfer, source_chain)
    }

    fun handle_complete_transfer_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        parsed_transfer: &TransferWithPayload,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        let redeemer = transfer_with_payload::recipient(parsed_transfer);

        // Transfer must be redeemed by the contract's registered Wormhole
        // emitter.
        assert!(redeemer == emitter::addr(emitter_cap), E_INVALID_REDEEMER);

        let (token_coin, _) =
            verify_transfer_details<CoinType>(
                token_bridge_state,
                transfer_with_payload::token_chain(parsed_transfer),
                transfer_with_payload::token_address(parsed_transfer),
                transfer_with_payload::recipient_chain(parsed_transfer),
                transfer_with_payload::amount(parsed_transfer),
                ctx
            );

        token_coin
    }

    #[test_only]
    /// This method is exists to expose `handle_complete_transfer_with_payload`
    /// and validate its job. `handle_complete_transfer_with_payload` is used by
    /// `complete_transfer_with_payload`.
    public fun complete_transfer_with_payload_test_only<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        _worm_state: &mut WormholeState,
        parsed_transfer: TransferWithPayload,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        handle_complete_transfer_with_payload<CoinType>(
                token_bridge_state,
                emitter_cap,
                &parsed_transfer,
                ctx
            )
    }
}

#[test_only]
module token_bridge::complete_transfer_with_payload_test {
    use sui::coin::{Self, CoinMetadata};
    use sui::test_scenario::{Self, Scenario};
    use wormhole::external_address::{Self};
    use wormhole::state::{Self as wormhole_state, State as WormholeState};

    use token_bridge::bridge_state_test::{
        set_up_wormhole_core_and_token_bridges
    };
    use token_bridge::complete_transfer_with_payload::{Self};
    use token_bridge::native_coin_10_decimals::{Self, NATIVE_COIN_10_DECIMALS};
    use token_bridge::wrapped_coin_12_decimals::{WRAPPED_COIN_12_DECIMALS};

    use token_bridge::state::{Self, State};
    use token_bridge::wrapped_coin_12_decimals_test::{Self};
    use token_bridge::register_chain::{Self};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    /// Mock registration VAA for Sui token bridge w/ fake address 0x1
    const SUI_REGISTRATION_VAA : vector<u8> = x"010000000001006c4df448af19846c6aa1d8df584696248bfd772dba9521118c6e447005b4f2712c5ce46367b0a769fbedaa5a4224b20e1ae381c802e24487165c4501185e1a9a01000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000312ace100000000000000000000000000000000000000000000546f6b656e427269646765010000001500000000000000000000000000000000000000000000000000000000deadbeef";

    /// VAA for transfer token with payload for token originating from Ethereum.
    /// This VAA is used to test complete_transfer_with_payload::complete_transfer_with_payload
    /// in the test_complete_transfer_wrapped test below.
    /// This VAA is signed by the guardian with public key beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe.
    const VAA : vector<u8> = x"01000000000100d8e4e04ac55ed24773a31b0a89bab8c1b9201e76bd03fe0de9da1506058ab30c01344cf11a47005bdfbe47458cb289388e4a87ed271fb8306fd83656172b19dc010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f030000000000000000000000000000000000000000000000000000000000000bb800000000000000000000000000000000000000000000000000000000beefface00020000000000000000000000000000000000000000000000000000000000000003001500000000000000000000000000000000000000000000000000000000deadbeefaaaa";
    // ========================================= VAA Details =========================================
    //   signatures: [
    //     {
    //       guardianSetIndex: 0,
    //       signature: 'd8e4e04ac55ed24773a31b0a89bab8c1b9201e76bd03fe0de9da1506058ab30c01344cf11a47005bdfbe47458cb289388e4a87ed271fb8306fd83656172b19dc01'
    //     }
    //   ],
    //   emitterChain: 2,
    //   emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    //   sequence: 1n,
    //   consistencyLevel: 15,
    //   payload: {
    //     module: 'TokenBridge',
    //     type: 'TransferWithPayload',
    //     amount: 3000n,
    //     tokenAddress: '0x00000000000000000000000000000000000000000000000000000000beefface',
    //     tokenChain: 2,
    //     toAddress: '0x0000000000000000000000000000000000000000000000000000000000000003',
    //     chain: 21,
    //     fromAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    //     payload: '0xaaaa'
    //   },
    //

    /// VAA for transfer token with payload for token originating from Sui.
    /// This VAA is signed by the guardian with public key beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe.
    const VAA_NATIVE : vector<u8> = x"01000000000100db621e2bd419cd8c254ec15827bded51bf79f45c0df9923c9071a50ae7b3cdec44d3ff45db0dc5caa17ad36f48bf06e34995a83c76c77eb5c541b036586c0748000000000000000000001500000000000000000000000000000000000000000000000000000000deadbeef000000000000000100030000000000000000000000000000000000000000000000000000000000000bb8000000000000000000000000000000000000000000000000000000000000000100150000000000000000000000000000000000000000000000000000000000000003001500000000000000000000000000000000000000000000000000000000deadbeefaaaa";
    //   signatures: [
    //     {
    //       guardianSetIndex: 0,
    //       signature: '2c8599ebc4e5f1ca832ad21e208226f22cff674c9db9dc6aca18b953b49c65154641e0b4074a0ff435b2b3380c87f457222ef77250722bf2aa50940b371af99901'
    //     }
    //   ],
    //   emitterChain: 21,
    //   emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    //   sequence: 1n,
    //   consistencyLevel: 0,
    //   payload: {
    //     module: 'TokenBridge',
    //     type: 'TransferWithPayload',
    //     amount: 3000n,
    //     tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000001',
    //     tokenChain: 21,
    //     toAddress: '0x0000000000000000000000000000000000000000000000000000000000000003',
    //     chain: 21,
    //     fromAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    //     payload: '0xaaaa'
    //   },

    #[test]
    /// Test the public-facing function complete_transfer_with_payload.
    /// using a native transfer VAA.
    fun test_complete_transfer_native(){
        use token_bridge::transfer_with_payload::{Self};

        let (admin, _, _) = people();
        let test = scenario();
        // Initializes core and token bridge.
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // Initialize native token.
        test_scenario::next_tx(&mut test, admin);{
            native_coin_10_decimals::test_init(test_scenario::ctx(&mut test));
        };
        // Register Sui token bridge emitter.
        test_scenario::next_tx(&mut test, admin); {
            let wormhole_state = test_scenario::take_shared<WormholeState>(&test);
            let bridge_state = test_scenario::take_shared<State>(&test);
            register_chain::submit_vaa(
                &mut bridge_state,
                &mut wormhole_state,
                SUI_REGISTRATION_VAA,
                test_scenario::ctx(&mut test)
            );
            test_scenario::return_shared<WormholeState>(wormhole_state);
            test_scenario::return_shared<State>(bridge_state);
        };
        // Register native asset type with the token bridge.
        test_scenario::next_tx(&mut test, admin);{
            let bridge_state = test_scenario::take_shared<State>(&test);
            let worm_state = test_scenario::take_shared<WormholeState>(&test);
            let coin_meta =
                test_scenario::take_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(&test);
            state::register_native_asset<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &coin_meta,
            );
            test_scenario::return_shared<State>(bridge_state);
            test_scenario::return_shared<WormholeState>(worm_state);
            test_scenario::return_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(coin_meta);
        };
        // Get the treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        test_scenario::next_tx(&mut test, admin); {
            let bridge_state = test_scenario::take_shared<State>(&test);
            let worm_state = test_scenario::take_shared<WormholeState>(&test);
            let t_cap =
                test_scenario::take_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(&test);
            let coins =
                coin::mint<NATIVE_COIN_10_DECIMALS>(
                    &mut t_cap,
                    10000000000, // amount
                    test_scenario::ctx(&mut test)
                );
            state::deposit<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                coins
            );
            test_scenario::return_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(t_cap);
            test_scenario::return_shared<State>(bridge_state);
            test_scenario::return_shared<WormholeState>(worm_state);
        };
        // complete transfer with payload (send native tokens + payload)
        test_scenario::next_tx(&mut test, admin); {
            let bridge_state = test_scenario::take_shared<State>(&test);
            let worm_state = test_scenario::take_shared<WormholeState>(&test);

            // Register and obtain a new emitter capability.
            // Emitter_cap_1 is discarded and not used.
            let emitter_cap_1 =
                wormhole_state::new_emitter(
                    &mut worm_state, test_scenario::ctx(&mut test)
                );
            // Emitter_cap_2 has the address 0x03 (because it is the third emitter to be
            // registered with wormhole), which coincidentally is the recipient address
            // of the transfer_with_payload VAA defined above.
            let emitter_cap_2 =
                wormhole_state::new_emitter(
                    &mut worm_state, test_scenario::ctx(&mut test)
                );

            // Execute complete_transfer_with_payload.
            let (token_coins, parsed_transfer, source_chain) =
                complete_transfer_with_payload::complete_transfer_with_payload<NATIVE_COIN_10_DECIMALS>(
                    &mut bridge_state,
                    &emitter_cap_2,
                    &mut worm_state,
                    VAA_NATIVE,
                    test_scenario::ctx(&mut test)
                );

            // Assert coin value, source chain, and parsed transfer details are correct.
            // We expect the coin value to be 300000, because that's in terms of
            // 10 decimals. The amount specifed in the VAA is 3000, because that's
            // in terms of 8 decimals.
            assert!(coin::value(&token_coins) == 300000, 0);
            assert!(source_chain == 21, 0);
            assert!(transfer_with_payload::token_address(&parsed_transfer)==external_address::from_any_bytes(x"01"), 0);
            assert!(transfer_with_payload::sender(&parsed_transfer)==external_address::from_any_bytes(x"deadbeef"), 0);
            assert!(transfer_with_payload::payload(&parsed_transfer)==x"aaaa", 0);

            // Clean-up!
            test_scenario::return_shared<State>(bridge_state);
            test_scenario::return_shared<WormholeState>(worm_state);

            // Trash remaining objects.
            sui::transfer::transfer(token_coins, @0x0);
            sui::transfer::transfer(emitter_cap_1, @0x0);
            sui::transfer::transfer(emitter_cap_2, @0x0);
        };
        test_scenario::end(test);
    }

    #[test]
    /// Test the public-facing function complete_transfer_with_payload.
    /// Use an actual devnet Wormhole complete transfer with payload VAA.
    ///
    /// This test confirms that:
    ///   - complete_transfer_with_payload function deserializes
    ///     the encoded Transfer object and recovers the source chain, payload,
    ///     and additional transfer details correctly.
    ///   - a wrapped coin with the correct value is minted by the bridge
    ///     and returned by complete_transfer_with_payload
    ///
    fun test_complete_transfer_wrapped(){
        use token_bridge::transfer_with_payload::{Self};

        let (admin, _, _) = people();
        let test = scenario();
        // Initializes core and token bridge, registers devnet Ethereum token bridge,
        // and registers wrapped token COIN_WITNESS with Sui token bridge
        test = wrapped_coin_12_decimals_test::test_register_wrapped_(admin, test);

        // complete transfer with payload (send native tokens + payload)
        test_scenario::next_tx(&mut test, admin); {
            let bridge_state = test_scenario::take_shared<State>(&test);
            let worm_state = test_scenario::take_shared<WormholeState>(&test);

            // Register and obtain a new emitter capability.
            // Emitter_cap_1 is discarded and not used.
            let emitter_cap_1 =
                wormhole_state::new_emitter(
                    &mut worm_state, test_scenario::ctx(&mut test)
                );
            // Emitter_cap_2 has the address 0x03 (because it is the third emitter to be
            // registered with wormhole), which coincidentally is the recipient address
            // of the transfer_with_payload VAA defined above.
            let emitter_cap_2 =
                wormhole_state::new_emitter(
                    &mut worm_state, test_scenario::ctx(&mut test)
                );

            // Execute complete_transfer_with_payload.
            let (token_coins, parsed_transfer, source_chain) =
                complete_transfer_with_payload::complete_transfer_with_payload<WRAPPED_COIN_12_DECIMALS>(
                    &mut bridge_state,
                    &emitter_cap_2,
                    &mut worm_state,
                    VAA,
                    test_scenario::ctx(&mut test)
                );

            // Assert coin value, source chain, and parsed transfer details are correct.
            assert!(coin::value(&token_coins) == 3000, 0);
            assert!(source_chain == 2, 0);
            assert!(transfer_with_payload::token_address(&parsed_transfer)==external_address::from_any_bytes(x"beefface"), 0);
            assert!(transfer_with_payload::sender(&parsed_transfer)==external_address::from_any_bytes(x"deadbeef"), 0);
            assert!(transfer_with_payload::payload(&parsed_transfer)==x"aaaa", 0);

            // Clean-up!
            test_scenario::return_shared<State>(bridge_state);
            test_scenario::return_shared<WormholeState>(worm_state);

            // Trash remaining objects.
            sui::transfer::transfer(token_coins, @0x0);
            sui::transfer::transfer(emitter_cap_1, @0x0);
            sui::transfer::transfer(emitter_cap_2, @0x0);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::complete_transfer_with_payload::E_INVALID_REDEEMER,
        location=token_bridge::complete_transfer_with_payload
    )]
    /// Test the public-facing function complete_transfer_with_payload.
    /// This test fails because the ecmitter_cap (recipient) is incorrect (0x2 instead of 0x3).
    ///
    fun test_complete_transfer_wrapped_wrong_recipient(){
        let (admin, _, _) = people();
        let test = scenario();
        // Initializes core and token bridge, registers devnet Ethereum token bridge,
        // and registers wrapped token COIN_WITNESS with Sui token bridge
        test = wrapped_coin_12_decimals_test::test_register_wrapped_(admin, test);

        // complete transfer with payload (send native tokens + payload)
        test_scenario::next_tx(&mut test, admin); {
            let bridge_state = test_scenario::take_shared<State>(&test);
            let worm_state = test_scenario::take_shared<WormholeState>(&test);

            let emitter_cap =
                wormhole_state::new_emitter(
                    &mut worm_state, test_scenario::ctx(&mut test)
                );

            // Execute complete_transfer_with_payload.
            let (token_coins, _parsed_transfer, _source_chain) =
                complete_transfer_with_payload::complete_transfer_with_payload<WRAPPED_COIN_12_DECIMALS>(
                    &mut bridge_state,
                    &emitter_cap, // Incorrect recipient.
                    &mut worm_state,
                    VAA,
                    test_scenario::ctx(&mut test)
                );

            // Clean-up!
            test_scenario::return_shared<State>(bridge_state);
            test_scenario::return_shared<WormholeState>(worm_state);

            // Trash remaining objects.
            sui::transfer::transfer(token_coins, @0x0);
            sui::transfer::transfer(emitter_cap, @0x0);
        };
        test_scenario::end(test);
    }
}
