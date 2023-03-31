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
    use sui::balance::{Balance};
    use sui::clock::{Clock};
    use sui::sui::{SUI};
    use wormhole::emitter::{EmitterCap};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{Self, State};
    use token_bridge::transfer_with_payload::{Self};
    use token_bridge::version_control::{
        TransferTokensWithPayload as TransferTokensWithPayloadControl
    };

    /// `transfer_tokens_with_payload` takes a `Balance` of a coin type and
    /// bridges this asset out of Sui by either joining this balance in the
    /// Token Bridge's custody for native assets or burning the balance
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
        bridged_in: Balance<CoinType>,
        wormhole_fee: Balance<SUI>,
        redeemer_chain: u16,
        redeemer: ExternalAddress,
        payload: vector<u8>,
        nonce: u32,
        the_clock: &Clock
    ): u64 {
        state::check_minimum_requirement<TransferTokensWithPayloadControl>(
            token_bridge_state
        );

        // Encode Wormhole message payload.
        let encoded_transfer_with_payload =
            bridge_in_and_serialize_transfer(
                token_bridge_state,
                emitter_cap,
                bridged_in,
                redeemer_chain,
                redeemer,
                payload
            );

        // Publish.
        state::publish_wormhole_message(
            token_bridge_state,
            worm_state,
            nonce,
            encoded_transfer_with_payload,
            wormhole_fee,
            the_clock
        )
    }

    fun bridge_in_and_serialize_transfer<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCap,
        bridged_in: Balance<CoinType>,
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
        bridged_in: Balance<CoinType>,
        redeemer_chain: u16,
        redeemer: ExternalAddress,
        payload: vector<u8>
    ): vector<u8> {
        bridge_in_and_serialize_transfer(
            token_bridge_state,
            emitter_cap,
            bridged_in,
            redeemer_chain,
            redeemer,
            payload
        )
    }
}

// #[test_only]
// module token_bridge::transfer_tokens_with_payload_tests {
//     use sui::balance::{Self};
//     use sui::test_scenario::{
//         Self,
//         Scenario,
//         next_tx,
//         return_shared,
//         take_shared,
//         ctx,
//         num_user_events
//     };
//     use sui::transfer::{Self};
//     use wormhole::external_address::{Self};
//     use wormhole::state::{Self as wormhole_state, State as WormholeState};

//     use token_bridge::coin_wrapped_12::{Self, COIN_WRAPPED_12};
//     use token_bridge::coin_native_4::{Self, COIN_NATIVE_4};
//     use token_bridge::state::{Self, State};
//     use token_bridge::token_bridge_scenario::{register_dummy_emitter};
//     use token_bridge::transfer_tokens_with_payload::{
//         transfer_tokens_with_payload
//     };

//     fun scenario(): Scenario { test_scenario::begin(@0x123233) }
//     fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

//     #[test]
//     fun test_transfer_native_token_with_payload(){
//         let (admin, _, _) = people();
//         let test = scenario();
//         // Set up core and token bridges.
//         test = set_up_wormhole_core_and_token_bridges(admin, test);
//         // Initialize the coin.
//         coin_native_4::init_test_only(ctx(&mut test));
//         // Register native asset type with the token bridge, mint some coins,
//         // then initiate transfer.
//         next_tx(&mut test, admin);{
//             let bridge_state = take_shared<State>(&test);
//             let worm_state = take_shared<WormholeState>(&test);
//             let (treasury_cap, coin_meta) = coin_native_4::take_globals(&test);

//             state::register_native_asset_test_only(
//                 &mut bridge_state,
//                 &coin_meta,
//             );
//             let created =
//                 balance::create_for_testing<COIN_NATIVE_4>(10000);
//             let payload = x"beef0000beef0000beef";

//             // Register and obtain a new wormhole emitter cap.
//             let emitter_cap =
//                 wormhole_state::new_emitter(
//                     &mut worm_state, test_scenario::ctx(&mut test)
//                 );

//             // Call transfer tokens with payload with args defined above.
//             transfer_tokens_with_payload<COIN_NATIVE_4>(
//                 &mut bridge_state,
//                 &mut emitter_cap,
//                 &mut worm_state,
//                 created,
//                 balance::zero(), // Zero fee paid to wormhole.
//                 3, // Recipient chain id.
//                 external_address::from_any_bytes(x"deadbeef0000beef"), // Recipient.
//                 0, // Relayer fee.
//                 payload,
//             );

//             // Clean up!
//             transfer::transfer(emitter_cap, admin);
//             return_shared<State>(bridge_state);
//             return_shared<WormholeState>(worm_state);
//             coin_native_4::return_globals(treasury_cap, coin_meta);
//         };

//         let tx_effects = next_tx(&mut test, admin);
//         // A single user event should be emitted, corresponding to
//         // publishing a Wormhole message for the token transfer with payload.
//         assert!(num_user_events(&tx_effects)==1, 0);

//         // Check that custody of the coins is indeed transferred to token bridge.
//         next_tx(&mut test, admin);{
//             let bridge_state = take_shared<State>(&test);
//             let cur_bal = state::custody_balance<COIN_NATIVE_4>(&mut bridge_state);
//             assert!(cur_bal==10000, 0);
//             return_shared<State>(bridge_state);
//         };
//         test_scenario::end(test);
//     }

//     #[test]
//     fun test_transfer_wrapped_token_with_payload(){
//         let (admin, _, _) = people();
//         let test = scenario();
//         // Set up core and token bridges.
//         test = set_up_wormhole_core_and_token_bridges(admin, test);
//         register_dummy_emitter(&mut test, 2);
//         coin_wrapped_12::init_and_register(&mut test, admin);

//         // Register wrapped asset type with the token bridge, mint some coins,
//         // initiate transfer.
//         next_tx(&mut test, admin);{
//             let bridge_state = take_shared<State>(&test);
//             let worm_state = take_shared<WormholeState>(&test);
//             let payload = x"ddddaaaabbbb";

//             let released =
//                 state::put_into_circulation_test_only<COIN_WRAPPED_12>(
//                     &mut bridge_state,
//                     1000, // Amount.
//                 );

//             // Register and obtain a new wormhole emitter cap.
//             let emitter_cap =
//                 wormhole_state::new_emitter(
//                     &mut worm_state, test_scenario::ctx(&mut test)
//                 );

//             // Call complete transfer with payload using previous args.
//             transfer_tokens_with_payload<COIN_WRAPPED_12>(
//                 &mut bridge_state,
//                 &mut emitter_cap,
//                 &mut worm_state,
//                 released,
//                 balance::zero(), // Zero fee paid to wormhole.
//                 3, // Recipient chain id.
//                 external_address::from_any_bytes(x"deadbeef0000beef"), // Recipient.
//                 0, // Relayer fee.
//                 payload,
//             );

//             // Clean-up!
//             wormhole::emitter::destroy_cap(emitter_cap);
//             return_shared(bridge_state);
//             return_shared(worm_state);
//         };
//         let tx_effects = next_tx(&mut test, admin);
//         // A single user event should be emitted, corresponding to
//         // publishing a Wormhole message for the token transfer with payload
//         assert!(num_user_events(&tx_effects)==1, 0);
//         test_scenario::end(test);
//     }
// }
