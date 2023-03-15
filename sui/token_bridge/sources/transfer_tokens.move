module token_bridge::transfer_tokens {
    use sui::balance::{Self, Balance};
    use sui::sui::{SUI};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{State as WormholeState};

    use token_bridge::normalized_amount::{Self, NormalizedAmount};
    use token_bridge::state::{Self, State};
    use token_bridge::transfer::{Self};

    // `transfer_tokens_with_payload` requires `handle_transfer_tokens`.
    friend token_bridge::transfer_tokens_with_payload;

    /// Relayer fee exceeds `Coin` balance.
    const E_TOO_MUCH_RELAYER_FEE: u64 = 0;

    public fun transfer_tokens<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        bridged: Balance<CoinType>,
        wormhole_fee: Balance<SUI>,
        recipient_chain: u16,
        recipient: ExternalAddress,
        relayer_fee: u64,
        nonce: u32,
    ): u64 {
        let (
            token_chain,
            token_address,
            norm_amount,
            norm_relayer_fee
        ) = handle_transfer_tokens(token_bridge_state, bridged, relayer_fee);

        // Prepare for serialization.
        let transfer = transfer::new(
            norm_amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            norm_relayer_fee,
        );

        // Publish with encoded `Transfer`.
        state::publish_wormhole_message(
            token_bridge_state,
            worm_state,
            nonce,
            transfer::serialize(transfer),
            wormhole_fee,
        )
    }

    /// For a given `CoinType`, prepare outbound transfer.
    ///
    /// This method is also used in `transfer_tokens_with_payload`.
    public(friend) fun handle_transfer_tokens<CoinType>(
        token_bridge_state: &mut State,
        bridged: Balance<CoinType>,
        relayer_fee: u64,
    ): (u16, ExternalAddress, NormalizedAmount, NormalizedAmount) {
        // Disallow `relayer_fee` to be greater than the amount in `Coin`.
        let amount = balance::value(&bridged);
        assert!(relayer_fee <= amount, E_TOO_MUCH_RELAYER_FEE);

        // Either burn or deposit depending on `CoinType`.
        state::take_from_circulation<CoinType>(token_bridge_state, bridged);

        // Fetch canonical token info from registry.
        let (
            token_chain,
            token_address
        ) = state::token_info<CoinType>(token_bridge_state);

        // And decimals to normalize raw amounts.
        let decimals = state::coin_decimals<CoinType>(token_bridge_state);

        (
            token_chain,
            token_address,
            normalized_amount::from_raw(amount, decimals),
            normalized_amount::from_raw(relayer_fee, decimals)
        )
    }
}


#[test_only]
module token_bridge::transfer_token_test {
    use sui::balance::{Self};
    use sui::test_scenario::{
        Self,
        Scenario,
        next_tx,
        return_shared,
        take_shared,
        num_user_events,
    };
    use wormhole::external_address::{Self};
    use wormhole::state::{State as WormholeState};

    use token_bridge::bridge_state_test::{
        set_up_wormhole_core_and_token_bridges
    };
    use token_bridge::coin_wrapped_12::{Self, COIN_WRAPPED_12};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::state::{Self, State};
    use token_bridge::token_bridge_scenario::{
        take_states,
        register_dummy_emitter,
        return_states,
    };
    use token_bridge::transfer_tokens::{
        E_TOO_MUCH_RELAYER_FEE,
        transfer_tokens,
    };

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    #[expected_failure(abort_code = E_TOO_MUCH_RELAYER_FEE)] // E_TOO_MUCH_RELAYER_FEE
    fun test_transfer_native_token_too_much_relayer_fee(){
        let (admin, _, _) = people();
        let test = scenario();
        // Set up core and token bridges.
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // Initialize the coin.
        coin_native_10::init_and_register(&mut test, admin);
        // Register native asset type with the token bridge, mint some coins,
        // and initiate transfer.
        next_tx(&mut test, admin);

        let (bridge_state, worm_state) = take_states(&test);
        let bridged =
            balance::create_for_testing<COIN_NATIVE_10>(10000);

        // You shall not pass!
        transfer_tokens(
            &mut bridge_state,
            &mut worm_state,
            bridged,
            balance::zero(), // zero fee paid to wormhole
            3, // recipient chain id
            external_address::from_any_bytes(x"deadbeef0000beef"), // recipient address
            100000000, // relayer fee (too much)
            0 // nonce is unused field for now
        );

        // Clean up.
        return_states(bridge_state, worm_state);

        // Done.
        test_scenario::end(test);
    }

    #[test]
    fun test_transfer_native_token(){
        let (admin, _, _) = people();
        let test = scenario();
        // Set up core and token bridges.
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // Initialize the coin.
        let mint_amount = 10000;
        let minted =
            coin_native_10::init_register_and_mint(
                &mut test,
                admin,
                mint_amount
            );
        // Register native asset type with the token bridge, mint some coins,
        // and finally initiate transfer.
        next_tx(&mut test, admin);

        let (bridge_state, worm_state) = take_states(&test);

        let sequence = transfer_tokens<COIN_NATIVE_10>(
            &mut bridge_state,
            &mut worm_state,
            minted,
            balance::zero(), // zero fee paid to wormhole
            3, // recipient chain id
            external_address::from_bytes(x"000000000000000000000000000000000000000000000000deadbeef0000beef"), // recipient address
            0, // relayer fee
            0 // unused field for now
        );
        assert!(sequence == 0, 0);
        return_states(bridge_state, worm_state);

        let tx_effects = next_tx(&mut test, admin);
        // A single user event should be emitted, corresponding to
        // publishing a Wormhole message for the token transfer
        assert!(num_user_events(&tx_effects)==1, 0);

        // TODO: do multiple transfers.

        // check that custody of the coins is indeed transferred to token bridge
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let cur_bal = state::custody_balance<COIN_NATIVE_10>(&mut bridge_state);
            assert!(cur_bal==10000, 0);
            return_shared<State>(bridge_state);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_transfer_wrapped_token() {
        let (admin, _, _) = people();
        let test = scenario();
        // Set up core and token bridges.
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        register_dummy_emitter(&mut test, 2);
        coin_wrapped_12::init_and_register(&mut test, admin);

        // Register wrapped asset type with the token bridge, mint some coins,
        // and finally initiate transfer.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let minted =
                state::put_into_circulation_test_only<COIN_WRAPPED_12>(
                    &mut bridge_state,
                    1000, // amount
                );

            transfer_tokens<COIN_WRAPPED_12>(
                &mut bridge_state,
                &mut worm_state,
                minted,
                balance::zero(), // zero fee paid to wormhole
                3, // recipient chain id
                external_address::from_bytes(x"000000000000000000000000000000000000000000000000deadbeef0000beef"), // recipient address
                0, // relayer fee
                0 // unused field for now
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        let tx_effects = next_tx(&mut test, admin);
        // A single user event should be emitted, corresponding to
        // publishing a Wormhole message for the token transfer
        assert!(num_user_events(&tx_effects)==1, 0);
        // How to check if token was actually burned?
        test_scenario::end(test);
    }

}
