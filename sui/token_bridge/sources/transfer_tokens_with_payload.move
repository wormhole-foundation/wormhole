module token_bridge::transfer_tokens_with_payload {
    use sui::sui::{SUI};
    use sui::coin::{Coin};
    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{Self, State};
    use token_bridge::transfer_result::{Self};
    use token_bridge::transfer_tokens::{handle_transfer_tokens};
    use token_bridge::transfer_with_payload::{Self};

    public fun transfer_tokens_with_payload<CoinType>(
        emitter_cap: &EmitterCap,
        bridge_state: &mut State,
        wormhole_state: &mut WormholeState,
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<SUI>,
        recipient_chain: u16,
        recipient: ExternalAddress,
        nonce: u32,
        payload: vector<u8>,
    ): u64 {
        let result = handle_transfer_tokens<CoinType>(
            bridge_state,
            coins,
            0,
        );
        let (token_chain, token_address, normalized_amount, _)
            = transfer_result::destroy(result);

        let transfer = transfer_with_payload::new(
            normalized_amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            emitter::addr(emitter_cap),
            payload
        );

        state::publish_wormhole_message(
            bridge_state,
            wormhole_state,
            nonce,
            transfer_with_payload::serialize(transfer),
            wormhole_fee_coins
        )
    }
}

#[test_only]
module token_bridge::transfer_tokens_with_payload_test {
    use sui::coin::{Self, CoinMetadata, TreasuryCap};
    use sui::sui::{SUI};
    use sui::test_scenario::{
        Self,
        Scenario,
        next_tx,
        return_shared,
        take_shared,
        take_from_address,
        ctx,
        num_user_events
    };
    use sui::transfer::{Self};

    use wormhole::external_address::{Self};
    use wormhole::state::{Self as wormhole_state, State as WormholeState};

    use token_bridge::bridge_state_test::{
        set_up_wormhole_core_and_token_bridges
    };
    use token_bridge::create_wrapped::{Self};
    use token_bridge::wrapped_coin_12_decimals::{Self, WRAPPED_COIN_12_DECIMALS};
    use token_bridge::native_coin_4_decimals::{Self, NATIVE_COIN_4_DECIMALS};
    use token_bridge::state::{Self, State};
    use token_bridge::transfer_tokens_with_payload::{
        transfer_tokens_with_payload
    };
    use token_bridge::wrapped_coin::{WrappedCoin};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    fun test_transfer_native_token_with_payload(){
        let (admin, _, _) = people();
        let test = scenario();
        // Set up core and token bridges.
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // Initialize the coin.
        native_coin_4_decimals::test_init(ctx(&mut test));
        // Register native asset type with the token bridge, mint some coins,
        // then initiate transfer.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_4_DECIMALS>>(&test);
            let treasury_cap = take_shared<TreasuryCap<NATIVE_COIN_4_DECIMALS>>(&test);
            state::register_native_asset<NATIVE_COIN_4_DECIMALS>(
                &mut bridge_state,
                &coin_meta,
            );
            let coins = coin::mint<NATIVE_COIN_4_DECIMALS>(
                &mut treasury_cap,
                10000,
                ctx(&mut test)
            );
            let payload = x"beef0000beef0000beef";

            // Register and obtain a new wormhole emitter cap.
            let emitter_cap =
                wormhole_state::new_emitter(
                    &mut worm_state, test_scenario::ctx(&mut test)
                );

            // Call transfer tokens with payload with args defined above.
            transfer_tokens_with_payload<NATIVE_COIN_4_DECIMALS>(
                &mut emitter_cap,
                &mut bridge_state,
                &mut worm_state,
                coins,
                coin::zero<SUI>(ctx(&mut test)), // Zero fee paid to wormhole.
                3, // Recipient chain id.
                external_address::from_any_bytes(x"deadbeef0000beef"), // Recipient.
                0, // Relayer fee.
                payload,
            );

            // Clean up!
            transfer::transfer(emitter_cap, admin);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_4_DECIMALS>>(coin_meta);
            return_shared<TreasuryCap<NATIVE_COIN_4_DECIMALS>>(treasury_cap);
        };

        let tx_effects = next_tx(&mut test, admin);
        // A single user event should be emitted, corresponding to
        // publishing a Wormhole message for the token transfer with payload.
        assert!(num_user_events(&tx_effects)==1, 0);

        // Check that custody of the coins is indeed transferred to token bridge.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let cur_bal = state::balance<NATIVE_COIN_4_DECIMALS>(&mut bridge_state);
            assert!(cur_bal==10000, 0);
            return_shared<State>(bridge_state);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_transfer_wrapped_token_with_payload(){
        let (admin, _, _) = people();
        let test = scenario();
        // Set up core and token bridges.
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // Initialize the wrapped coin and register the eth chain.
        wrapped_coin_12_decimals::test_init(ctx(&mut test));
        // Register chain emitter (chain id x emitter address) that attested
        // the wrapped token.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            state::register_emitter(
                &mut bridge_state,
                2, // Chain ID.
                external_address::from_any_bytes(
                    x"00000000000000000000000000000000000000000000000000000000deadbeef"
                )
            );
            return_shared<State>(bridge_state);
        };
        // Register wrapped asset type with the token bridge, mint some coins,
        // initiate transfer.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(&test);
            let new_wrapped_coin =
                take_from_address<WrappedCoin<WRAPPED_COIN_12_DECIMALS>>(&test, admin);
            let payload = x"ddddaaaabbbb";

            // Register wrapped asset with the token bridge.
            create_wrapped::register_new_coin<WRAPPED_COIN_12_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                new_wrapped_coin,
                &mut coin_meta,
                ctx(&mut test)
            );

            let coins =
                state::mint<WRAPPED_COIN_12_DECIMALS>(
                    &mut bridge_state,
                    1000, // Amount.
                    ctx(&mut test)
                );

            // Register and obtain a new wormhole emitter cap.
            let emitter_cap =
                wormhole_state::new_emitter(
                    &mut worm_state, test_scenario::ctx(&mut test)
                );

            // Call complete transfer with payload using previous args.
            transfer_tokens_with_payload<WRAPPED_COIN_12_DECIMALS>(
                &mut emitter_cap,
                &mut bridge_state,
                &mut worm_state,
                coins,
                coin::zero<SUI>(ctx(&mut test)), // Zero fee paid to wormhole.
                3, // Recipient chain id.
                external_address::from_any_bytes(x"deadbeef0000beef"), // Recipient.
                0, // Relayer fee.
                payload,
            );

            // Clean-up!
            transfer::transfer(emitter_cap, admin);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(coin_meta);
        };
        let tx_effects = next_tx(&mut test, admin);
        // A single user event should be emitted, corresponding to
        // publishing a Wormhole message for the token transfer with payload
        assert!(num_user_events(&tx_effects)==1, 0);
        test_scenario::end(test);
    }
}
