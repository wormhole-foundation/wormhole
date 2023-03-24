#[test_only]
module token_bridge::coin_wrapped_12 {
    use sui::balance::{Supply};
    use sui::test_scenario::{Self, Scenario};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::create_wrapped::{Self, WrappedAssetSetup};

    struct COIN_WRAPPED_12 has drop {}

    const VAA: vector<u8> =
        x"0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";

    fun init(witness: COIN_WRAPPED_12, ctx: &mut TxContext) {
        transfer::transfer(
            create_wrapped::prepare_registration(
                witness,
                VAA,
                ctx
            ),
            tx_context::sender(ctx)
        );
    }

    public fun encoded_vaa(): vector<u8> {
        VAA
    }

    public fun token_meta(): AssetMeta<COIN_WRAPPED_12> {
        asset_meta::deserialize(
            wormhole::vaa::peel_payload_from_vaa(&VAA)
        )
    }

    #[test_only]
    /// for a test scenario, simply deploy the coin and expose `TreasuryCap`.
    public fun init_and_take_supply(
        scenario: &mut Scenario,
        caller: address
    ): Supply<COIN_WRAPPED_12> {
        use token_bridge::create_wrapped::{take_supply};

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Publish coin.
        init(COIN_WRAPPED_12 {}, test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        take_supply(test_scenario::take_from_sender(scenario))
    }

    #[test_only]
    /// For a test scenario, register this wrapped asset.
    ///
    /// NOTE: Even though this module is `#[test_only]`, this method is tagged
    /// with the same macro  as a trick to allow another method within this
    /// module to call `init` using OTW.
    public fun init_and_register(scenario: &mut Scenario, caller: address) {
        use token_bridge::token_bridge_scenario::{return_states, take_states};

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Publish coin.
        init(COIN_WRAPPED_12 {}, test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let (token_bridge_state, worm_state) = take_states(scenario);

        // Register the attested asset.
        create_wrapped::complete_registration(
            &mut token_bridge_state,
            &worm_state,
            test_scenario::take_from_sender<WrappedAssetSetup<COIN_WRAPPED_12>>(
                scenario
            ),
            test_scenario::ctx(scenario)
        );

        // Clean up.
        return_states(token_bridge_state, worm_state);
    }

    #[test_only]
    /// NOTE: Even though this module is `#[test_only]`, this method is tagged
    /// with the same macro  as a trick to allow another method within this
    /// module to call `init` using OTW.
    public fun init_test_only(ctx: &mut TxContext) {
        init(COIN_WRAPPED_12 {}, ctx)
    }
}

// #[test_only]
// module token_bridge::wrapped_coin_12_decimals_tests {
//     use std::string::{Self};
//     use std::ascii::{Self};

//     use sui::test_scenario::{Self, Scenario, ctx, next_tx, take_from_address, return_shared, take_shared};
//     use sui::coin::{Self, CoinMetadata};

//     use wormhole::state::{State as WormholeState};
//     use wormhole::external_address::{Self};

//     use token_bridge::state::{Self, State};
//     use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};
//     use token_bridge::create_wrapped::{WrappedAssetSetup, register_new_coin};
//     use token_bridge::token_bridge_scenario::{register_dummy_emitter};

//     use token_bridge::coin_wrapped_12::{init_test_only, COIN_WRAPPED_12};
//     use token_bridge::create_wrapped::{Self};

//     use token_bridge::asset_meta::{Self};
//     use token_bridge::coin_wrapped_12::{token_meta};

//     #[test]
//     public fun test_native_decimals() {
//         assert!(asset_meta::native_decimals(&token_meta()) == 12, 0);
//     }

//     fun scenario(): Scenario { test_scenario::begin(@0x123233) }
//     fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

//     /// Registration VAA for the etheruem token bridge 0xdeadbeef.
//     const ETHEREUM_TOKEN_REG: vector<u8> =
//         x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

//     #[test]
//     /// This helper function calls coin init to create wrapped coin and
//     /// subsequently traasfers it to sender.
//     fun test_create_wrapped() {
//         let test = scenario();
//         let (admin, _, _) = people();
//         next_tx(&mut test, admin); {
//             init_test_only(ctx(&mut test))
//         };
//         test_scenario::end(test);
//     }

//     #[test]
//     /// This helper function calls token bridge register wrapped coin.
//     fun test_register_wrapped() {
//         let (admin, _, _) = people();
//         let scenario = scenario();
//         let test = test_register_wrapped_(admin, scenario);
//         test_scenario::end(test);
//     }

//     #[test]
//     /// In this test, we first call test_register_wrapped_ to register a
//     /// wrapped asset with the name "BEEF face Token" and symbol "BEEF".
//     /// We then modify the CoinMetadata associated with this token to update
//     /// the name to "BEEAA" and symbol to "BEE" by calling the function
//     /// update_registered_metadata with a new vaa.
//     fun test_update_registered_wrapped_coin_metadata() {
//         // New vaa corresponding to beefface token but with updated name
//         // and symbol.
//         let new_vaa = x"01000000000100d4229a3b38107f198d82003447cadb396392abc86c8d8244b7457e0a0aeb8ca92d1493419ce0049f41ed9e32c5fb653ee516db02feb92fa259ec75499cd2ad4e010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000003309538000200000000000000000000000000000000000000000000000000000000beefface00020c42454500000000000000000000000000000000000000000000000000000000004245454141000000000000000000000000000000000000000000000000000000";
//         let (admin, _, _) = people();
//         let scenario = scenario();
//         // First, register the wrapped coin corresponding to the VAA defined
//         // above for the beefface token.
//         let test = test_register_wrapped_(admin, scenario);
//         // Second, call create_wrapped::update_registered_metadata
//         // to update token metadata.
//         next_tx(&mut test, admin); {
//             let bridge_state = take_shared<State>(&test);
//             let worm_state = take_shared<WormholeState>(&test);
//             let coin_meta = take_shared<CoinMetadata<COIN_WRAPPED_12>>(&test);
//             create_wrapped::update_registered_metadata<COIN_WRAPPED_12>(
//                 &mut bridge_state,
//                 &mut worm_state,
//                 new_vaa,
//                 &mut coin_meta,
//                 ctx(&mut test)
//             );
//             return_shared<State>(bridge_state);
//             return_shared<WormholeState>(worm_state);
//             return_shared<CoinMetadata<COIN_WRAPPED_12>>(coin_meta);
//         };
//         // Check that the name and symbol in the coin metadata have indeed been
//         // updated.
//         next_tx(&mut test, admin); {
//             let coin_meta = take_shared<CoinMetadata<COIN_WRAPPED_12>>(&test);
//             assert!(coin::get_symbol(&coin_meta)==ascii::string(b"BEE"), 0);
//             assert!(coin::get_name(&coin_meta)==string::utf8(b"BEEAA"), 0);
//             return_shared<CoinMetadata<COIN_WRAPPED_12>>(coin_meta);
//         };
//         test_scenario::end(test);
//     }

//     #[test]
//     #[expected_failure(abort_code = state::E_CANONICAL_TOKEN_INFO_MISMATCH)]
//     /// In this test, we attempt to update coin metadata for an asset that
//     /// does not correspond to COIN_WRAPPED_12, namely an Ethereum token
//     /// with the address "0x00000000000000000000000000000000000000000000000000000000beeffaaa"
//     /// instead of "0x00000000000000000000000000000000000000000000000000000000beefface".
//     fun test_update_registered_wrapped_coin_metadata_wrong_origin_address () {
//         // New vaa corresponding to beefface token but with updated name
//         // and symbol.
//         let new_vaa = x"01000000000100e19f642c8b23459f0bf85a8bf3e5b0f5181b4f94ab6a4f5c290720a070021f0e6d7faa2e1d5113d40b686e6afd0899bd62110d81e6800a9e6bc8c3f22d68e6c8010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000005afeb99000200000000000000000000000000000000000000000000000000000000beeffaaa00020c42454500000000000000000000000000000000000000000000000000000000004245454141000000000000000000000000000000000000000000000000000000";
//         let (admin, _, _) = people();
//         let scenario = scenario();
//         // First, register the wrapped coin corresponding to the VAA defined
//         // above for the beefface token.
//         let test = test_register_wrapped_(admin, scenario);
//         // Second, call create_wrapped::update_registered_metadata
//         // to update token metadata.
//         next_tx(&mut test, admin); {
//             let bridge_state = take_shared<State>(&test);
//             let worm_state = take_shared<WormholeState>(&test);
//             let coin_meta = take_shared<CoinMetadata<COIN_WRAPPED_12>>(&test);
//             // This call fails because we cannot update the Coinmetadata for
//             // COIN_WRAPPED_12 using a vaa that does not correspond to
//             // it (origin address mismatch).
//             create_wrapped::update_registered_metadata<COIN_WRAPPED_12>(
//                 &mut bridge_state,
//                 &mut worm_state,
//                 new_vaa,
//                 &mut coin_meta,
//                 ctx(&mut test)
//             );
//             return_shared<State>(bridge_state);
//             return_shared<WormholeState>(worm_state);
//             return_shared<CoinMetadata<COIN_WRAPPED_12>>(coin_meta);
//         };
//         test_scenario::end(test);
//     }

//     #[test]
//     #[expected_failure(abort_code = state::E_CANONICAL_TOKEN_INFO_MISMATCH)]
//     /// In this test, we attempt to update coin metadata for an asset that
//     /// does not correspond to COIN_WRAPPED_12, namely a token
//     /// with origin chain of Acala instead of Ethereum.
//     fun test_update_registered_wrapped_coin_metadata_wrong_origin_chain () {
//         // New vaa corresponding to beefface token but with updated name
//         // and symbol.
//         let new_vaa = x"01000000000100c25dedd249b73b6f53c0f63152fa17e393e437561c05a31d8d8cb259e11f49fd1f1b723eabc728d1bec32848c3c268c38935e6f3419a51ce1ec147fd18c5aabf000000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000003380b11000200000000000000000000000000000000000000000000000000000000beefface000c0c42454500000000000000000000000000000000000000000000000000000000004245454141000000000000000000000000000000000000000000000000000000";
//         let (admin, _, _) = people();
//         let scenario = scenario();
//         // First, register the wrapped coin corresponding to the VAA defined
//         // above for the beefface token.
//         let test = test_register_wrapped_(admin, scenario);
//         // Second, call create_wrapped::update_registered_metadata
//         // to update token metadata.
//         next_tx(&mut test, admin); {
//             let bridge_state = take_shared<State>(&test);
//             let worm_state = take_shared<WormholeState>(&test);
//             let coin_meta = take_shared<CoinMetadata<COIN_WRAPPED_12>>(&test);
//             // This call fails because we cannot update the Coinmetadata for
//             // COIN_WRAPPED_12 using a vaa that does not correspond to
//             // it (origin chain mismatch).
//             create_wrapped::update_registered_metadata<COIN_WRAPPED_12>(
//                 &mut bridge_state,
//                 &mut worm_state,
//                 new_vaa,
//                 &mut coin_meta,
//                 ctx(&mut test)
//             );
//             return_shared<State>(bridge_state);
//             return_shared<WormholeState>(worm_state);
//             return_shared<CoinMetadata<COIN_WRAPPED_12>>(coin_meta);
//         };
//         test_scenario::end(test);
//     }

//     #[test]
//     #[expected_failure(
//         abort_code = token_bridge::token_registry::E_UNREGISTERED,
//         location=token_bridge::token_registry
//     )]
//     /// In this test, we attempt to update coin metadata for an asset that
//     /// has not been previously registered at all (neither a registered wrapped
//     /// or native asset).
//     fun test_update_registered_wrapped_coin_metadata_not_registered () {
//         // New vaa corresponding to beefface token but with updated name
//         // and symbol.
//         let new_vaa = x"01000000000100d4229a3b38107f198d82003447cadb396392abc86c8d8244b7457e0a0aeb8ca92d1493419ce0049f41ed9e32c5fb653ee516db02feb92fa259ec75499cd2ad4e010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000003309538000200000000000000000000000000000000000000000000000000000000beefface00020c42454500000000000000000000000000000000000000000000000000000000004245454141000000000000000000000000000000000000000000000000000000";
//         let (admin, _, _) = people();
//         let scenario = scenario();
//         // First, set up wormhole and token bridges, and register eth token bridge
//         let test = set_up_wormhole_core_and_token_bridges(admin, scenario);
//         register_dummy_emitter(&mut test, 2);
//         // create and transfer new wrapped coin to sender
//         next_tx(&mut test, admin); {
//             init_test_only(ctx(&mut test));
//         };
//         // Third, attempt to call create_wrapped::update_registered_metadata
//         // to update token metadata.
//         next_tx(&mut test, admin); {
//             let bridge_state = take_shared<State>(&test);
//             let worm_state = take_shared<WormholeState>(&test);
//             let coin_meta = take_shared<CoinMetadata<COIN_WRAPPED_12>>(&test);
//             // This call fails because we COIN_WRAPPED_12 has not been
//             // previously registered with the token bridge either as a native
//             // or wrapped asset.
//             create_wrapped::update_registered_metadata<COIN_WRAPPED_12>(
//                 &mut bridge_state,
//                 &mut worm_state,
//                 new_vaa,
//                 &mut coin_meta,
//                 ctx(&mut test)
//             );
//             return_shared<State>(bridge_state);
//             return_shared<WormholeState>(worm_state);
//             return_shared<CoinMetadata<COIN_WRAPPED_12>>(coin_meta);
//         };
//         test_scenario::end(test);
//     }

//     /// This is a helper function that is called in a variety of test files.
//     /// It is not meant to be a standalone test!
//     public fun test_register_wrapped_(admin: address, test: Scenario): Scenario {
//         test = set_up_wormhole_core_and_token_bridges(admin, test);
//         register_dummy_emitter(&mut test, 2);
//         // Create and transfer new wrapped coin to sender.
//         next_tx(&mut test, admin); {
//             init_test_only(ctx(&mut test))
//         };
//         // Register wrapped coin with token bridge, handing it the treasury cap
//         // and storing metadata
//         next_tx(&mut test, admin);{
//             let bridge_state = take_shared<State>(&test);
//             let worm_state = take_shared<WormholeState>(&test);
//             let coin_meta = take_shared<CoinMetadata<COIN_WRAPPED_12>>(&test);
//             let wrapped_coin = take_from_address<WrappedAssetSetup<COIN_WRAPPED_12>>(&test, admin);
//             register_new_coin<COIN_WRAPPED_12>(
//                 &mut bridge_state,
//                 &mut worm_state,
//                 wrapped_coin,
//                 &mut coin_meta,
//                 ctx(&mut test)
//             );
//             // assert that wrapped asset is indeed recognized by token bridge
//             let is_registered = state::is_registered_asset<COIN_WRAPPED_12>(&bridge_state);
//             assert!(is_registered, 0);

//             // assert that wrapped asset is not recognized as a native asset by token bridge
//             let is_wrapped = state::is_wrapped_asset<COIN_WRAPPED_12>(&bridge_state);
//             assert!(is_wrapped, 0);

//             // assert origin info is correct
//             let (token_chain, token_address) = state::token_info<COIN_WRAPPED_12>(&bridge_state);
//             assert!(token_chain == 2, 0);

//             let expected_addr = external_address::from_any_bytes(x"beefface");
//             assert!(token_address == expected_addr, 0);

//             return_shared<State>(bridge_state);
//             return_shared<WormholeState>(worm_state);
//             return_shared<CoinMetadata<COIN_WRAPPED_12>>(coin_meta);
//         };
//         return test
//     }
// }
