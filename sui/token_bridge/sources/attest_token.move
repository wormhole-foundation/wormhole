module token_bridge::attest_token {
    use sui::sui::SUI;
    use sui::coin::{Coin, CoinMetadata};
    use sui::tx_context::TxContext;

    use wormhole::state::{State as WormholeState};

    use token_bridge::bridge_state::{Self as state, BridgeState};
    use token_bridge::asset_meta::{Self, AssetMeta};

    public entry fun attest_token<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        fee_coins: Coin<SUI>,
        ctx: &mut TxContext
    ) {
        let asset_meta = attest_token_internal(
            wormhole_state,
            bridge_state,
            coin_meta,
            ctx
        );
        let payload = asset_meta::encode(asset_meta);
        let nonce = 0;
        state::publish_message(
            wormhole_state,
            bridge_state,
            nonce,
            payload,
            fee_coins
        );
    }

    fun attest_token_internal<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        ctx: &mut TxContext
    ): AssetMeta {
        let asset_meta =
            state::register_native_asset<CoinType>(wormhole_state, bridge_state, coin_meta, ctx);
        return asset_meta
    }

    #[test_only]
    public fun test_attest_token_internal<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        ctx: &mut TxContext
    ): AssetMeta {
        attest_token_internal<CoinType>(
            wormhole_state,
            bridge_state,
            coin_meta,
            ctx
        )
    }
}

#[test_only]
module token_bridge::attest_token_test{
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_shared, return_shared};
    use sui::coin::{CoinMetadata};

    use wormhole::state::{State};

    use token_bridge::string32::{Self};
    use token_bridge::bridge_state::{BridgeState};
    use token_bridge::attest_token::{test_attest_token_internal};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};
    use token_bridge::native_coin_witness::{Self, NATIVE_COIN_WITNESS};
    use token_bridge::asset_meta::{Self};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    fun test_attest_token(){
        let test = scenario();
        let (admin, _, _) = people();

        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            native_coin_witness::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<State>(&test);
            let bridge_state = take_shared<BridgeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);

            let asset_meta = test_attest_token_internal<NATIVE_COIN_WITNESS>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );

            assert!(asset_meta::get_decimals(&asset_meta)==10, 0);
            assert!(asset_meta::get_symbol(&asset_meta)==string32::from_bytes(x"00"), 0);
            assert!(asset_meta::get_name(&asset_meta)==string32::from_bytes(x"11"), 0);

            return_shared<State>(wormhole_state);
            return_shared<BridgeState>(bridge_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = 0, location=0000000000000000000000000000000000000002::dynamic_field)]
    fun test_attest_token_twice_fails(){
        let test = scenario();
        let (admin, _, _) = people();

        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            native_coin_witness::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<State>(&test);
            let bridge_state = take_shared<BridgeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);

            let _asset_meta_1 = test_attest_token_internal<NATIVE_COIN_WITNESS>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let _asset_meta_2 = test_attest_token_internal<NATIVE_COIN_WITNESS>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            return_shared<State>(wormhole_state);
            return_shared<BridgeState>(bridge_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
        };
        test_scenario::end(test);
    }
}
