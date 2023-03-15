module token_bridge::transfer_tokens_with_payload {
    use sui::balance::{Balance};
    use sui::sui::{SUI};
    use wormhole::emitter::{EmitterCap};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{Self, State};
    use token_bridge::transfer_tokens::{handle_transfer_tokens};
    use token_bridge::transfer_with_payload::{Self};

    public fun transfer_tokens_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        worm_state: &mut WormholeState,
        bridged: Balance<CoinType>,
        wormhole_fee: Balance<SUI>,
        recipient_chain: u16,
        recipient: ExternalAddress,
        nonce: u32,
        payload: vector<u8>,
    ): u64 {
        let (token_chain, token_address, norm_amount, _) =
            handle_transfer_tokens<CoinType>(
                token_bridge_state,
                bridged,
                0,
            );
        let transfer =
            transfer_with_payload::new_from_emitter(
                emitter_cap,
                norm_amount,
                token_address,
                token_chain,
                recipient,
                recipient_chain,
                payload
            );

        state::publish_wormhole_message(
            token_bridge_state,
            worm_state,
            nonce,
            transfer_with_payload::serialize(transfer),
            wormhole_fee
        )
    }
}

#[test_only]
module token_bridge::transfer_tokens_with_payload_test {
    use sui::balance::{Self};
    use sui::test_scenario::{
        Self,
        Scenario,
        next_tx,
        return_shared,
        take_shared,
        ctx,
        num_user_events
    };
    use sui::transfer::{Self};

    use wormhole::external_address::{Self};
    use wormhole::state::{Self as wormhole_state, State as WormholeState};

    use token_bridge::bridge_state_test::{
        set_up_wormhole_core_and_token_bridges
    };
    use token_bridge::coin_wrapped_12::{Self, COIN_WRAPPED_12};
    use token_bridge::coin_native_4::{Self, COIN_NATIVE_4};
    use token_bridge::state::{Self, State};
    use token_bridge::token_bridge_scenario::{register_dummy_emitter};
    use token_bridge::transfer_tokens_with_payload::{
        transfer_tokens_with_payload
    };

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    fun test_transfer_native_token_with_payload(){
        let (admin, _, _) = people();
        let test = scenario();
        // Set up core and token bridges.
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // Initialize the coin.
        coin_native_4::init_test_only(ctx(&mut test));
        // Register native asset type with the token bridge, mint some coins,
        // then initiate transfer.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let (treasury_cap, coin_meta) = coin_native_4::take_globals(&test);

            state::register_native_asset_test_only(
                &mut bridge_state,
                &coin_meta,
            );
            let created =
                balance::create_for_testing<COIN_NATIVE_4>(10000);
            let payload = x"beef0000beef0000beef";

            // Register and obtain a new wormhole emitter cap.
            let emitter_cap =
                wormhole_state::new_emitter(
                    &mut worm_state, test_scenario::ctx(&mut test)
                );

            // Call transfer tokens with payload with args defined above.
            transfer_tokens_with_payload<COIN_NATIVE_4>(
                &mut bridge_state,
                &mut emitter_cap,
                &mut worm_state,
                created,
                balance::zero(), // Zero fee paid to wormhole.
                3, // Recipient chain id.
                external_address::from_any_bytes(x"deadbeef0000beef"), // Recipient.
                0, // Relayer fee.
                payload,
            );

            // Clean up!
            transfer::public_transfer(emitter_cap, admin);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            coin_native_4::return_globals(treasury_cap, coin_meta);
        };

        let tx_effects = next_tx(&mut test, admin);
        // A single user event should be emitted, corresponding to
        // publishing a Wormhole message for the token transfer with payload.
        assert!(num_user_events(&tx_effects)==1, 0);

        // Check that custody of the coins is indeed transferred to token bridge.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let cur_bal = state::custody_balance<COIN_NATIVE_4>(&mut bridge_state);
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
        register_dummy_emitter(&mut test, 2);
        coin_wrapped_12::init_and_register(&mut test, admin);

        // Register wrapped asset type with the token bridge, mint some coins,
        // initiate transfer.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let payload = x"ddddaaaabbbb";

            let released =
                state::put_into_circulation_test_only<COIN_WRAPPED_12>(
                    &mut bridge_state,
                    1000, // Amount.
                );

            // Register and obtain a new wormhole emitter cap.
            let emitter_cap =
                wormhole_state::new_emitter(
                    &mut worm_state, test_scenario::ctx(&mut test)
                );

            // Call complete transfer with payload using previous args.
            transfer_tokens_with_payload<COIN_WRAPPED_12>(
                &mut bridge_state,
                &mut emitter_cap,
                &mut worm_state,
                released,
                balance::zero(), // Zero fee paid to wormhole.
                3, // Recipient chain id.
                external_address::from_any_bytes(x"deadbeef0000beef"), // Recipient.
                0, // Relayer fee.
                payload,
            );

            // Clean-up!
            wormhole::emitter::destroy_cap(emitter_cap);
            return_shared(bridge_state);
            return_shared(worm_state);
        };
        let tx_effects = next_tx(&mut test, admin);
        // A single user event should be emitted, corresponding to
        // publishing a Wormhole message for the token transfer with payload
        assert!(num_user_events(&tx_effects)==1, 0);
        test_scenario::end(test);
    }
}
