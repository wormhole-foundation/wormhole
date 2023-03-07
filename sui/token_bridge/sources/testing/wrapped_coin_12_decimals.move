#[test_only]
module token_bridge::wrapped_coin_12_decimals {
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};

    use token_bridge::create_wrapped::{Self};

    struct WRAPPED_COIN_12_DECIMALS has drop {}

    fun init(coin_witness: WRAPPED_COIN_12_DECIMALS, ctx: &mut TxContext) {
        // Step 1. Paste token attestation VAA below. This example is ethereum beefface token.
        let vaa_bytes = x"0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";

        let new_wrapped_coin = create_wrapped::create_unregistered_currency(vaa_bytes, coin_witness, ctx);
        transfer::transfer(
            new_wrapped_coin,
            tx_context::sender(ctx)
        );
    }

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(WRAPPED_COIN_12_DECIMALS {}, ctx)
    }
}

#[test_only]
module token_bridge::wrapped_coin_12_decimals_test {
    use std::string::{Self};
    use std::ascii::{Self};

    use sui::test_scenario::{Self, Scenario, ctx, next_tx, take_from_address, return_shared, take_shared};
    use sui::coin::{Self, CoinMetadata};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self};

    use token_bridge::state::{State, is_registered_asset, is_wrapped_asset,
        token_info};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};
    use token_bridge::create_wrapped::{register_new_coin};
    use token_bridge::register_chain::{submit_vaa};
    use token_bridge::wrapped_coin::{WrappedCoin};

    use token_bridge::wrapped_coin_12_decimals::{test_init, WRAPPED_COIN_12_DECIMALS};
    use token_bridge::token_info::{Self};
    use token_bridge::create_wrapped::{Self};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    /// Registration VAA for the etheruem token bridge 0xdeadbeef.
    const ETHEREUM_TOKEN_REG: vector<u8> =
        x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    #[test]
    /// This helper function calls coin init to create wrapped coin and
    /// subsequently traasfers it to sender.
    fun test_create_wrapped() {
        let test = scenario();
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            test_init(ctx(&mut test))
        };
        test_scenario::end(test);
    }

    #[test]
    /// This helper function calls token bridge register wrapped coin.
    fun test_register_wrapped() {
        let (admin, _, _) = people();
        let scenario = scenario();
        let test = test_register_wrapped_(admin, scenario);
        test_scenario::end(test);
    }

    #[test]
    /// In this test, we first call test_register_wrapped_ to register a
    /// wrapped asset with the name "BEEF face Token" and symbol "BEEF".
    /// We then modify the CoinMetadata associated with this token to update
    /// the name to "BEEAA" and symbol to "BEE" by calling the function
    /// update_registered_metadata with a new vaa.
    fun test_update_registered_wrapped_coin_metadata() {
        // New vaa corresponding to beefface token but with updated name
        // and symbol.
        let new_vaa = x"01000000000100d4229a3b38107f198d82003447cadb396392abc86c8d8244b7457e0a0aeb8ca92d1493419ce0049f41ed9e32c5fb653ee516db02feb92fa259ec75499cd2ad4e010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000003309538000200000000000000000000000000000000000000000000000000000000beefface00020c42454500000000000000000000000000000000000000000000000000000000004245454141000000000000000000000000000000000000000000000000000000";
        let (admin, _, _) = people();
        let scenario = scenario();
        // First, register the wrapped coin corresponding to the VAA defined
        // above for the beefface token.
        let test = test_register_wrapped_(admin, scenario);
        // Second, call create_wrapped::update_registered_metadata
        // to update token metadata.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(&test);
            create_wrapped::update_registered_metadata<WRAPPED_COIN_12_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                new_vaa,
                &mut coin_meta,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(coin_meta);
        };
        // Check that the name and symbol in the coin metadata have indeed been
        // updated.
        next_tx(&mut test, admin); {
            let coin_meta = take_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(&test);
            assert!(coin::get_symbol(&coin_meta)==ascii::string(b"BEE"), 0);
            assert!(coin::get_name(&coin_meta)==string::utf8(b"BEEAA"), 0);
            return_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(coin_meta);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::create_wrapped::E_UNREGISTERED_WRAPPED_ASSET,
        location=token_bridge::create_wrapped
    )]
    /// In this test, we attempt to update coin metadata for an asset that
    /// does not correspond to WRAPPED_COIN_12_DECIMALS, namely an Ethereum token
    /// with the address "0x00000000000000000000000000000000000000000000000000000000beeffaaa"
    /// instead of "0x00000000000000000000000000000000000000000000000000000000beefface".
    fun test_update_registered_wrapped_coin_metadata_wrong_origin_address () {
        // New vaa corresponding to beefface token but with updated name
        // and symbol.
        let new_vaa = x"01000000000100e19f642c8b23459f0bf85a8bf3e5b0f5181b4f94ab6a4f5c290720a070021f0e6d7faa2e1d5113d40b686e6afd0899bd62110d81e6800a9e6bc8c3f22d68e6c8010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000005afeb99000200000000000000000000000000000000000000000000000000000000beeffaaa00020c42454500000000000000000000000000000000000000000000000000000000004245454141000000000000000000000000000000000000000000000000000000";
        let (admin, _, _) = people();
        let scenario = scenario();
        // First, register the wrapped coin corresponding to the VAA defined
        // above for the beefface token.
        let test = test_register_wrapped_(admin, scenario);
        // Second, call create_wrapped::update_registered_metadata
        // to update token metadata.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(&test);
            // This call fails because we cannot update the Coinmetadata for
            // WRAPPED_COIN_12_DECIMALS using a vaa that does not correspond to
            // it (origin address mismatch).
            create_wrapped::update_registered_metadata<WRAPPED_COIN_12_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                new_vaa,
                &mut coin_meta,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(coin_meta);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::create_wrapped::E_UNREGISTERED_WRAPPED_ASSET,
        location=token_bridge::create_wrapped
    )]
    /// In this test, we attempt to update coin metadata for an asset that
    /// does not correspond to WRAPPED_COIN_12_DECIMALS, namely a token
    /// with origin chain of Acala instead of Ethereum.
    fun test_update_registered_wrapped_coin_metadata_wrong_origin_chain () {
        // New vaa corresponding to beefface token but with updated name
        // and symbol.
        let new_vaa = x"01000000000100c25dedd249b73b6f53c0f63152fa17e393e437561c05a31d8d8cb259e11f49fd1f1b723eabc728d1bec32848c3c268c38935e6f3419a51ce1ec147fd18c5aabf000000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000003380b11000200000000000000000000000000000000000000000000000000000000beefface000c0c42454500000000000000000000000000000000000000000000000000000000004245454141000000000000000000000000000000000000000000000000000000";
        let (admin, _, _) = people();
        let scenario = scenario();
        // First, register the wrapped coin corresponding to the VAA defined
        // above for the beefface token.
        let test = test_register_wrapped_(admin, scenario);
        // Second, call create_wrapped::update_registered_metadata
        // to update token metadata.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(&test);
            // This call fails because we cannot update the Coinmetadata for
            // WRAPPED_COIN_12_DECIMALS using a vaa that does not correspond to
            // it (origin chain mismatch).
            create_wrapped::update_registered_metadata<WRAPPED_COIN_12_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                new_vaa,
                &mut coin_meta,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(coin_meta);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::registered_tokens::E_UNREGISTERED,
        location=token_bridge::registered_tokens
    )]
    /// In this test, we attempt to update coin metadata for an asset that
    /// has not been previously registered at all (neither a registered wrapped
    /// or native asset).
    fun test_update_registered_wrapped_coin_metadata_not_registered () {
        // New vaa corresponding to beefface token but with updated name
        // and symbol.
        let new_vaa = x"01000000000100d4229a3b38107f198d82003447cadb396392abc86c8d8244b7457e0a0aeb8ca92d1493419ce0049f41ed9e32c5fb653ee516db02feb92fa259ec75499cd2ad4e010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000003309538000200000000000000000000000000000000000000000000000000000000beefface00020c42454500000000000000000000000000000000000000000000000000000000004245454141000000000000000000000000000000000000000000000000000000";
        let (admin, _, _) = people();
        let scenario = scenario();
        // First, set up wormhole and token bridges, and register eth token bridge
        let test = set_up_wormhole_core_and_token_bridges(admin, scenario);
        // create and transfer new wrapped coin to sender
        next_tx(&mut test, admin); {
            test_init(ctx(&mut test))
        };
        // Second, register chain.
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            submit_vaa(&mut wormhole_state, &mut bridge_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
        };
        // Third, attempt to call create_wrapped::update_registered_metadata
        // to update token metadata.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(&test);
            // This call fails because we WRAPPED_COIN_12_DECIMALS has not been
            // previously registered with the token bridge either as a native
            // or wrapped asset.
            create_wrapped::update_registered_metadata<WRAPPED_COIN_12_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                new_vaa,
                &mut coin_meta,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(coin_meta);
        };
        test_scenario::end(test);
    }

    /// This is a helper function that is called in a variety of test files.
    /// It is not meant to be a standalone test!
    public fun test_register_wrapped_(admin: address, test: Scenario): Scenario {
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // Create and transfer new wrapped coin to sender.
        next_tx(&mut test, admin); {
            test_init(ctx(&mut test))
        };
        // Register chain.
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            submit_vaa(
                &mut wormhole_state,
                &mut bridge_state,
                ETHEREUM_TOKEN_REG,
                ctx(&mut test)
            );
            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
        };
        // Register wrapped coin with token bridge, handing it the treasury cap
        // and storing metadata
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(&test);
            let wrapped_coin = take_from_address<WrappedCoin<WRAPPED_COIN_12_DECIMALS>>(&test, admin);
            register_new_coin<WRAPPED_COIN_12_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                wrapped_coin,
                &mut coin_meta,
                ctx(&mut test)
            );
            // assert that wrapped asset is indeed recognized by token bridge
            let is_registered = is_registered_asset<WRAPPED_COIN_12_DECIMALS>(&bridge_state);
            assert!(is_registered, 0);

            // assert that wrapped asset is not recognized as a native asset by token bridge
            let is_wrapped = is_wrapped_asset<WRAPPED_COIN_12_DECIMALS>(&bridge_state);
            assert!(is_wrapped, 0);

            // assert origin info is correct
            let info = token_info<WRAPPED_COIN_12_DECIMALS>(&bridge_state);
            assert!(token_info::chain(&info) == 2, 0);

            let expected_addr = external_address::from_bytes(x"beefface");
            assert!(token_info::addr(&info) == expected_addr, 0);

            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<WRAPPED_COIN_12_DECIMALS>>(coin_meta);
        };
        return test
    }
}
