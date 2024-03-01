// SPDX-License-Identifier: Apache 2

#[test_only]
module token_bridge::coin_wrapped_7 {
    use sui::balance::{Balance};
    use sui::coin::{CoinMetadata, TreasuryCap};
    use sui::package::{UpgradeCap};
    use sui::test_scenario::{Self, Scenario};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::create_wrapped::{Self, WrappedAssetSetup};
    use token_bridge::state::{Self};
    use token_bridge::token_registry::{Self};
    use token_bridge::wrapped_asset::{Self};

    use token_bridge::version_control::{V__0_2_0 as V__CURRENT};

    struct COIN_WRAPPED_7 has drop {}

    // TODO: need to fix the emitter address
    // +------------------------------------------------------------------------------+
    // | Wormhole VAA v1         | nonce: 69               | time: 0                  |
    // | guardian set #0         | #1                      | consistency: 15          |
    // |------------------------------------------------------------------------------|
    // | Signature:                                                                   |
    // |   #0: 3d8fd671611d84801dc9d14a07835e8729d217b1aac77b054175d0f91294...        |
    // |------------------------------------------------------------------------------|
    // | Emitter: 0x00000000000000000000000000000000deadbeef (Ethereum)               |
    // |------------------------------------------------------------------------------|
    // | Token attestation                                                            |
    // | decimals: 7                                                                  |
    // | Token: 0x00000000000000000000000000000000deafface (Ethereum)                 |
    // | Symbol: DEC7                                                                 |
    // | Name: DECIMALS 7                                                             |
    // +------------------------------------------------------------------------------+
    const VAA: vector<u8> =
        x"010000000001003d8fd671611d84801dc9d14a07835e8729d217b1aac77b054175d0f91294040742a1ed6f3e732b2fbf208e64422816accf89dd0cd3ead20d2e0fb3d372ce221c010000000000000045000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f0200000000000000000000000000000000000000000000000000000000deafface000207000000000000000000000000000000000000000000000000000000004445433700000000000000000000000000000000000000000000444543494d414c532037";

    fun init(witness: COIN_WRAPPED_7, ctx: &mut TxContext) {
        let (
            setup,
            upgrade_cap
        ) =
            create_wrapped::new_setup_current(
                witness,
                7,
                ctx
            );
        transfer::public_transfer(setup, tx_context::sender(ctx));
        transfer::public_transfer(upgrade_cap, tx_context::sender(ctx));
    }

    #[test_only]
    public fun init_test_only(ctx: &mut TxContext) {
        init(COIN_WRAPPED_7 {}, ctx);
    }

    public fun encoded_vaa(): vector<u8> {
        VAA
    }

    #[allow(implicit_const_copy)]
    public fun token_meta(): AssetMeta {
        asset_meta::deserialize_test_only(
            wormhole::vaa::peel_payload_from_vaa(&VAA)
        )
    }

    #[test_only]
    /// for a test scenario, simply deploy the coin and expose `TreasuryCap`.
    public fun init_and_take_treasury_cap(
        scenario: &mut Scenario,
        caller: address
    ): TreasuryCap<COIN_WRAPPED_7> {
        use token_bridge::create_wrapped;

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Publish coin.
        init(COIN_WRAPPED_7 {}, test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        create_wrapped::take_treasury_cap(
            test_scenario::take_from_sender(scenario)
        )
    }

    #[test_only]
    /// For a test scenario, register this wrapped asset.
    ///
    /// NOTE: Even though this module is `#[test_only]`, this method is tagged
    /// with the same macro as a trick to allow another method within this
    /// module to call `init` using OTW.
    public fun init_and_register(
        scenario: &mut Scenario,
        caller: address
    ) {
        use token_bridge::token_bridge_scenario::{return_state, take_state};
        use wormhole::wormhole_scenario::{parse_and_verify_vaa};

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Publish coin.
        init(COIN_WRAPPED_7 {}, test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let token_bridge_state = take_state(scenario);

        let verified_vaa = parse_and_verify_vaa(scenario, VAA);
        let msg =
            token_bridge::vaa::verify_only_once(
                &mut token_bridge_state,
                verified_vaa
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let coin_meta =
            test_scenario::take_shared<CoinMetadata<COIN_WRAPPED_7>>(scenario);

        // Register the attested asset.
        create_wrapped::complete_registration(
            &mut token_bridge_state,
            &mut coin_meta,
            test_scenario::take_from_sender<
                WrappedAssetSetup<COIN_WRAPPED_7, V__CURRENT>
            >(
                scenario
            ),
            test_scenario::take_from_sender<UpgradeCap>(scenario),
            msg
        );

        test_scenario::return_shared(coin_meta);

        // Clean up.
        return_state(token_bridge_state);
    }

    #[test_only]
    /// NOTE: Even though this module is `#[test_only]`, this method is tagged
    /// with the same macro as a trick to allow another method within this
    /// module to call `init` using OTW.
    public fun init_register_and_mint(
        scenario: &mut Scenario,
        caller: address,
        amount: u64
    ): Balance<COIN_WRAPPED_7> {
        use token_bridge::token_bridge_scenario::{return_state, take_state};

        // First publish and register.
        init_and_register(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let token_bridge_state = take_state(scenario);
        let minted =
            wrapped_asset::mint_test_only(
                token_registry::borrow_mut_wrapped_test_only(
                    state::borrow_mut_token_registry_test_only(
                        &mut token_bridge_state
                    )
                ),
                amount
            );

        return_state(token_bridge_state);

        minted
    }
}

#[test_only]
module token_bridge::coin_wrapped_7_tests {
    use token_bridge::asset_meta::{Self};
    use token_bridge::coin_wrapped_7::{token_meta};

    #[test]
    fun test_native_decimals() {
        let meta = token_meta();
        assert!(asset_meta::native_decimals(&meta) == 7, 0);
        asset_meta::destroy(meta);
    }
}
