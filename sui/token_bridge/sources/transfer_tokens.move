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
    use sui::balance::{Self, Balance};
    use sui::clock::{Clock};
    use sui::sui::{SUI};
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

    /// Relayer fee exceeds `Balance` value.
    const E_RELAYER_FEE_EXCEEDS_AMOUNT: u64 = 0;

    /// `transfer_tokens` takes a `Balance` of a coin type and bridges this
    /// asset out of Sui by either joining this balance in the Token Bridge's
    /// custody for native assets or burning the balance for wrapped assets.
    ///
    /// Additionally, a `relayer_fee` of some value less than or equal to the
    /// `Balance` value can be specified to incentivize someone to redeem this
    /// transfer on behalf of the `recipient`.
    ///
    /// See `token_registry and `transfer_with_payload` module for more info.
    public fun transfer_tokens<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        bridged_in: Balance<CoinType>,
        wormhole_fee: Balance<SUI>,
        recipient_chain: u16,
        recipient: ExternalAddress,
        relayer_fee: u64,
        nonce: u32,
        the_clock: &Clock
    ): u64 {
        state::check_minimum_requirement<TransferTokensControl>(
            token_bridge_state
        );

        let encoded_transfer =
            bridge_in_and_serialize_transfer(
                token_bridge_state,
                bridged_in,
                recipient_chain,
                recipient,
                relayer_fee
            );

        // Publish with encoded `Transfer`.
        state::publish_wormhole_message(
            token_bridge_state,
            worm_state,
            nonce,
            encoded_transfer,
            wormhole_fee,
            the_clock
        )
    }

    /// For a given `CoinType`, prepare outbound transfer.
    ///
    /// This method is also used in `transfer_tokens_with_payload`.
    public(friend) fun verify_and_bridge_in<CoinType>(
        token_bridge_state: &mut State,
        bridged_in: Balance<CoinType>,
        relayer_fee: u64,
    ): (u16, ExternalAddress, NormalizedAmount, NormalizedAmount) {
        // Disallow `relayer_fee` to be greater than the amount in `Balance`.
        let amount = balance::value(&bridged_in);
        assert!(relayer_fee <= amount, E_RELAYER_FEE_EXCEEDS_AMOUNT);

        // Fetch canonical token info from registry.
        let verified = state::verified_asset<CoinType>(token_bridge_state);

        // Either burn or deposit depending on `CoinType`.
        let registry = state::borrow_mut_token_registry(token_bridge_state);
        if (token_registry::is_wrapped(&verified)) {
            wrapped_asset::burn_balance(
                token_registry::borrow_mut_wrapped(registry),
                bridged_in
            );
        } else {
            native_asset::deposit_balance(
                token_registry::borrow_mut_native(registry),
                bridged_in
            );
        };

        let decimals = token_registry::coin_decimals(&verified);

        (
            token_registry::token_chain(&verified),
            token_registry::token_address(&verified),
            normalized_amount::from_raw(amount, decimals),
            normalized_amount::from_raw(relayer_fee, decimals)
        )
    }

    fun bridge_in_and_serialize_transfer<CoinType>(
        token_bridge_state: &mut State,
        bridged_in: Balance<CoinType>,
        recipient_chain: u16,
        recipient: ExternalAddress,
        relayer_fee: u64
    ): vector<u8> {
        let (
            token_chain,
            token_address,
            norm_amount,
            norm_relayer_fee
        ) = verify_and_bridge_in(token_bridge_state, bridged_in, relayer_fee);

        transfer::serialize(
            transfer::new(
                norm_amount,
                token_address,
                token_chain,
                recipient,
                recipient_chain,
                norm_relayer_fee,
            )
        )
    }

    #[test_only]
    public fun bridge_in_and_serialize_transfer_test_only<CoinType>(
        token_bridge_state: &mut State,
        bridged_in: Balance<CoinType>,
        recipient_chain: u16,
        recipient: ExternalAddress,
        relayer_fee: u64
    ): vector<u8> {
        bridge_in_and_serialize_transfer(
            token_bridge_state,
            bridged_in,
            recipient_chain,
            recipient,
            relayer_fee
        )
    }
}

#[test_only]
module token_bridge::transfer_token_tests {
    fun test_transfer_tokens_native_10() {
        // use token_bridge::transfer_tokens::{transfer_tokens};

        // let user = person();
        // let my_scenario = test_scenario::begin(user);
        // let scenario = &mut my_scenario;

        // // Publish coin.
        // coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // // Set up contracts.
        // let wormhole_fee = 350;
        // set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // // Ignore effects.
        // test_scenario::next_tx(scenario, user);

        // let (token_bridge_state, worm_state) = take_states(scenario);
        // let the_clock = take_clock(scenario);
        // let coin_meta = coin_native_10::take_metadata(scenario);

        // // Emit `AssetMeta` payload.
        // let sequence =
        //     transfer_tokens(
        //         &mut token_bridge_state,
        //         &mut worm_state,
        //         balance::create_for_testing(
        //             wormhole_fee
        //         ),
        //         &coin_meta,
        //         1234, // nonce
        //         &the_clock
        //     );
        // assert!(sequence == 0, 0);

    }
}
//     use sui::balance::{Self};
//     use sui::test_scenario::{
//         Self,
//         Scenario,
//         next_tx,
//         return_shared,
//         take_shared,
//         num_user_events,
//     };
//     use wormhole::external_address::{Self};
//     use wormhole::state::{State as WormholeState};

//     use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
//     use token_bridge::coin_wrapped_12::{Self, COIN_WRAPPED_12};
//     use token_bridge::state::{Self, State};
//     use token_bridge::token_bridge_scenario::{
//         take_states,
//         register_dummy_emitter,
//         return_states,
//     };
//     use token_bridge::transfer_tokens::{Self};

//     fun scenario(): Scenario { test_scenario::begin(@0x123233) }
//     fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

//     #[test]
//     #[expected_failure(abort_code = E_RELAYER_FEE_EXCEEDS_AMOUNT)] // E_RELAYER_FEE_EXCEEDS_AMOUNT
//     fun test_transfer_native_token_too_much_relayer_fee(){
//         let (admin, _, _) = people();
//         let test = scenario();
//         // Set up core and token bridges.
//         test = set_up_wormhole_core_and_token_bridges(admin, test);
//         // Initialize the coin.
//         coin_native_10::init_and_register(&mut test, admin);
//         // Register native asset type with the token bridge, mint some coins,
//         // and initiate transfer.
//         next_tx(&mut test, admin);

//         let (bridge_state, worm_state) = take_states(&test);
//         let bridged =
//             balance::create_for_testing<COIN_NATIVE_10>(10000);

//         // You shall not pass!
//         transfer_tokens(
//             &mut bridge_state,
//             &mut worm_state,
//             bridged,
//             balance::zero(), // zero fee paid to wormhole
//             3, // recipient chain id
//             external_address::from_any_bytes(x"deadbeef0000beef"), // recipient address
//             100000000, // relayer fee (too much)
//             0 // nonce is unused field for now
//         );

//         // Clean up.
//         return_states(bridge_state, worm_state);

//         // Done.
//         test_scenario::end(test);
//     }

//     #[test]
//     fun test_transfer_native_token(){
//         let (admin, _, _) = people();
//         let test = scenario();
//         // Set up core and token bridges.
//         test = set_up_wormhole_core_and_token_bridges(admin, test);
//         // Initialize the coin.
//         let mint_amount = 10000;
//         let minted =
//             coin_native_10::init_register_and_mint(
//                 &mut test,
//                 admin,
//                 mint_amount
//             );
//         // Register native asset type with the token bridge, mint some coins,
//         // and finally initiate transfer.
//         next_tx(&mut test, admin);

//         let (bridge_state, worm_state) = take_states(&test);

//         let sequence = transfer_tokens<COIN_NATIVE_10>(
//             &mut bridge_state,
//             &mut worm_state,
//             minted,
//             balance::zero(), // zero fee paid to wormhole
//             3, // recipient chain id
//             external_address::from_bytes(x"000000000000000000000000000000000000000000000000deadbeef0000beef"), // recipient address
//             0, // relayer fee
//             0 // unused field for now
//         );
//         assert!(sequence == 0, 0);
//         return_states(bridge_state, worm_state);

//         let tx_effects = next_tx(&mut test, admin);
//         // A single user event should be emitted, corresponding to
//         // publishing a Wormhole message for the token transfer
//         assert!(num_user_events(&tx_effects)==1, 0);

//         // TODO: do multiple transfers.

//         // check that custody of the coins is indeed transferred to token bridge
//         next_tx(&mut test, admin);{
//             let bridge_state = take_shared<State>(&test);
//             let cur_bal = state::custody_balance<COIN_NATIVE_10>(&mut bridge_state);
//             assert!(cur_bal==10000, 0);
//             return_shared<State>(bridge_state);
//         };
//         test_scenario::end(test);
//     }

//     #[test]
//     fun test_transfer_wrapped_token() {
//         let (admin, _, _) = people();
//         let test = scenario();
//         // Set up core and token bridges.
//         test = set_up_wormhole_core_and_token_bridges(admin, test);
//         register_dummy_emitter(&mut test, 2);
//         coin_wrapped_12::init_and_register(&mut test, admin);

//         // Register wrapped asset type with the token bridge, mint some coins,
//         // and finally initiate transfer.
//         next_tx(&mut test, admin);{
//             let bridge_state = take_shared<State>(&test);
//             let worm_state = take_shared<WormholeState>(&test);

//             let minted =
//                 state::put_into_circulation_test_only<COIN_WRAPPED_12>(
//                     &mut bridge_state,
//                     1000, // amount
//                 );

//             transfer_tokens<COIN_WRAPPED_12>(
//                 &mut bridge_state,
//                 &mut worm_state,
//                 minted,
//                 balance::zero(), // zero fee paid to wormhole
//                 3, // recipient chain id
//                 external_address::from_bytes(x"000000000000000000000000000000000000000000000000deadbeef0000beef"), // recipient address
//                 0, // relayer fee
//                 0 // unused field for now
//             );
//             return_shared<State>(bridge_state);
//             return_shared<WormholeState>(worm_state);
//         };
//         let tx_effects = next_tx(&mut test, admin);
//         // A single user event should be emitted, corresponding to
//         // publishing a Wormhole message for the token transfer
//         assert!(num_user_events(&tx_effects)==1, 0);
//         // How to check if token was actually burned?
//         test_scenario::end(test);
//     }

// }
