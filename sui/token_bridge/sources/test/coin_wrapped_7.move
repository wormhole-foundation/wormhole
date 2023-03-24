#[test_only]
module token_bridge::coin_wrapped_7 {
    use sui::balance::{Supply};
    use sui::test_scenario::{Self, Scenario};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::create_wrapped::{Self, WrappedAssetSetup};

    struct COIN_WRAPPED_7 has drop {}

    const VAA: vector<u8> =
        x"01000000000100c99f831e64b8613bc837e2cf5e0bad5a2b9fb7dd0fe9886a74858f890e2c51d96bdccf72a5a931cc56819c8bdd8ad0ea6f56efb4b37e0608aab90f746a36e4bf0000000000000000450002000000000000000000000000deadbeefdeadbeefdeadbeefdeadbeefdeadbeef00000000000000010f0200000000000000000000000000000000000000000000000000000000beefface0002070000000000000000000000000000000000000000000000000000004445433132000000000000000000000000000000000000000000444543494d414c53203132";

    fun init(witness: COIN_WRAPPED_7, ctx: &mut TxContext) {
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

    public fun token_meta(): AssetMeta<COIN_WRAPPED_7> {
        asset_meta::deserialize(
            wormhole::vaa::peel_payload_from_vaa(&VAA)
        )
    }

    #[test_only]
    /// for a test scenario, simply deploy the coin and expose `TreasuryCap`.
    public fun init_and_take_supply(
        scenario: &mut Scenario,
        caller: address
    ): Supply<COIN_WRAPPED_7> {
        use token_bridge::create_wrapped::{take_supply};

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Publish coin.
        init(COIN_WRAPPED_7 {}, test_scenario::ctx(scenario));

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
        init(COIN_WRAPPED_7 {}, test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let (token_bridge_state, worm_state) = take_states(scenario);

        // Register the attested asset.
        create_wrapped::complete_registration(
            &mut token_bridge_state,
            &worm_state,
            test_scenario::take_from_sender<WrappedAssetSetup<COIN_WRAPPED_7>>(
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
        init(COIN_WRAPPED_7 {}, ctx)
    }
}

#[test_only]
module token_bridge::coin_wrapped_7_tests {
    use token_bridge::asset_meta::{Self};
    use token_bridge::coin_wrapped_7::{token_meta};

    #[test]
    public fun test_native_decimals() {
        assert!(asset_meta::native_decimals(&token_meta()) == 7, 0);
    }
}
