module token_bridge::attest_token {
    use sui::sui::SUI;
    use sui::coin::{Coin, CoinMetadata};

    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{Self, State};
    use token_bridge::asset_meta::{Self, AssetMeta};

    public entry fun attest_token<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        coin_meta: &CoinMetadata<CoinType>,
        fee_coins: Coin<SUI>,
        nonce: u32,
    ) {
        let asset_meta = handle_attest_token(
            token_bridge_state,
            coin_meta,
        );

        state::publish_wormhole_message(
            token_bridge_state,
            worm_state,
            nonce,
            asset_meta::serialize(asset_meta),
            fee_coins
        );
    }

    fun handle_attest_token<CoinType>(
        token_bridge_state: &mut State,
        coin_metadata: &CoinMetadata<CoinType>,
    ): AssetMeta {
        state::register_native_asset<CoinType>(
            token_bridge_state,
            coin_metadata,
        )
    }

    #[test_only]
    public fun test_handle_attest_token<CoinType>(
        token_bridge_state: &mut State,
        _worm_state: &mut WormholeState,
        coin_metadata: &CoinMetadata<CoinType>,
    ): AssetMeta {
        handle_attest_token(
            token_bridge_state,
            coin_metadata,
        )
    }
}

#[test_only]
module token_bridge::attest_token_test {
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_shared,
        return_shared};
    use sui::coin::{CoinMetadata};

    use wormhole::state::{State as WormholeState};

    use token_bridge::string32::{Self};
    use token_bridge::state::{Self, State};
    use token_bridge::attest_token::{test_handle_attest_token};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};
    use token_bridge::native_coin_10_decimals::{Self, NATIVE_COIN_10_DECIMALS};
    use token_bridge::asset_meta::{Self};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    fun test_attest_token(){
        let test = scenario();
        let (admin, someone, _) = people();

        test = set_up_wormhole_core_and_token_bridges(admin, test);

        // Init the native coin
        next_tx(&mut test, admin); {
            native_coin_10_decimals::test_init(ctx(&mut test));
        };

        // Proceed to next operation.
        next_tx(&mut test, someone);

        {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(&test);

            let asset_meta = test_handle_attest_token<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &mut wormhole_state,
                &coin_meta,
            );

            assert!(asset_meta::native_decimals(&asset_meta) == 10, 0);
            assert!(
                asset_meta::symbol(&asset_meta) == string32::from_bytes(x"00"),
                0
            );
            assert!(
                asset_meta::name(&asset_meta) == string32::from_bytes(x"11"),
                0
            );

            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
            return_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(coin_meta);
        };

        // Check that native token is registered.
        let effects = next_tx(&mut test, someone);

        // TODO: write comment of what we expect.
        let written_ids = test_scenario::written(&effects);
        assert!(std::vector::length(&written_ids) == 3, 0);

        {
            let bridge_state = take_shared<State>(&test);
            let is_registered =
                state::is_registered_asset<NATIVE_COIN_10_DECIMALS>(&mut bridge_state);
            assert!(is_registered, 0);
            return_shared<State>(bridge_state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::registered_tokens::E_ALREADY_REGISTERED,
        location=token_bridge::registered_tokens
    )]
    /// TODO: consider throwing token bridge error instead of
    /// sui::dynamic_field.
    fun test_attest_token_twice_fails(){
        let test = scenario();
        let (admin, _, _) = people();

        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            native_coin_10_decimals::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(&test);

            let _asset_meta_1 = test_handle_attest_token<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &mut wormhole_state,
                &coin_meta,
            );
            let _asset_meta_2 = test_handle_attest_token<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &mut wormhole_state,
                &coin_meta,
            );
            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
            return_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(coin_meta);
        };
        test_scenario::end(test);
    }
}
