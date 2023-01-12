#[test_only]
module token_bridge::coin_witness {
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};

    use token_bridge::wrapped;

    struct COIN_WITNESS has drop {}

    fun init(coin_witness: COIN_WITNESS, ctx: &mut TxContext) {
        // Step 1. Paste token attestation VAA below. This example is ethereum beefface token.
        let vaa_bytes = x"0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";

        let new_wrapped_coin = wrapped::create_wrapped_coin(vaa_bytes, coin_witness, ctx);
        transfer::transfer(
            new_wrapped_coin,
            tx_context::sender(ctx)
        );
    }

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(COIN_WITNESS {}, ctx)
    }
}

#[test_only]
module token_bridge::coin_witness_test {
    use sui::test_scenario::{Self, Scenario, ctx, next_tx, take_from_address, return_shared, take_shared};

    use wormhole::state::{State};
    use wormhole::myu16::{Self as u16};
    use wormhole::external_address::{Self};

    use token_bridge::bridge_state::{BridgeState, is_wrapped_asset, is_registered_native_asset, origin_info, get_token_chain_from_origin_info, get_token_address_from_origin_info};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};
    use token_bridge::wrapped::{NewWrappedCoin, register_wrapped_coin};
    use token_bridge::register_chain::{submit_vaa};

    use token_bridge::coin_witness::{test_init, COIN_WITNESS};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    /// Registration VAA for the etheruem token bridge 0xdeadbeef
    const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    // call coin init to create wrapped coin and traasfer to sender
    #[test]
    fun test_create_wrapped() {
        let test = scenario();
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            test_init(ctx(&mut test))
        };
        test_scenario::end(test);
    }

    // call token bridge register wrapped coin
    #[test]
    fun test_register_wrapped() {
        let (admin, _, _) = people();
        let scenario = scenario();
        let test = test_register_wrapped_(admin, scenario);
        test_scenario::end(test);
    }

    public fun test_register_wrapped_(admin: address, test: Scenario): Scenario {
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // create and transfer new wrapped coin to sender
        next_tx(&mut test, admin); {
            test_init(ctx(&mut test))
        };
        // register chain
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<State>(&test);
            let bridge_state = take_shared<BridgeState>(&test);
            submit_vaa(&mut wormhole_state, &mut bridge_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<State>(wormhole_state);
            return_shared<BridgeState>(bridge_state);
        };
        // register wrapped coin with token bridge, handing it the treasury cap and storing metadata
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<BridgeState>(&test);
            let worm_state = take_shared<State>(&test);
            let wrapped_coin = take_from_address<NewWrappedCoin<COIN_WITNESS>>(&test, admin);
            register_wrapped_coin<COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                wrapped_coin,
                ctx(&mut test)
            );
            // assert that wrapped asset is indeed recognized by token bridge
            let is_wrapped = is_wrapped_asset<COIN_WITNESS>(&bridge_state);
            assert!(is_wrapped, 0);

            // assert that wrapped asset is not recognized as a native asset by token bridge
            let is_native = is_registered_native_asset<COIN_WITNESS>(&bridge_state);
            assert!(!is_native, 0);

            // assert origin info is correct
            let origin_info = origin_info<COIN_WITNESS>(&bridge_state);
            let chain = get_token_chain_from_origin_info(&origin_info);
            let address = get_token_address_from_origin_info(&origin_info);
            assert!(chain == u16::from_u64(2), 0);
            assert!(address == external_address::from_bytes(x"beefface"), 0);
            return_shared<BridgeState>(bridge_state);
            return_shared<State>(worm_state);
        };
        return test
    }
}
