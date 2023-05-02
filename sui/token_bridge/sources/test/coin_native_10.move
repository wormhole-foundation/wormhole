// SPDX-License-Identifier: Apache 2

#[test_only]
module token_bridge::coin_native_10 {
    use std::option::{Self};
    use sui::balance::{Self, Balance};
    use sui::coin::{Self, CoinMetadata, TreasuryCap};
    use sui::test_scenario::{Self, Scenario};
    use sui::transfer::{Self};
    use sui::tx_context::{TxContext};

    use token_bridge::native_asset::{Self};
    use token_bridge::state::{Self};
    use token_bridge::token_registry::{Self};

    struct COIN_NATIVE_10 has drop {}

    // This module creates a Sui-native token for testing purposes,
    // for example in complete_transfer, where we create a native coin,
    // mint some and deposit in the token bridge, then complete transfer
    // and ultimately transfer a portion of those native coins to a recipient.
    fun init(coin_witness: COIN_NATIVE_10, ctx: &mut TxContext) {
        let (
            treasury_cap,
            coin_metadata
        ) =
            coin::create_currency(
                coin_witness,
                10,
                b"DEC10",
                b"Decimals 10",
                b"Coin with 10 decimals for testing purposes.",
                option::none(),
                ctx
            );

        // Allow us to mutate metadata if we need.
        transfer::public_share_object(coin_metadata);

        // Give everyone access to `TrasuryCap`.
        transfer::public_share_object(treasury_cap);
    }

    #[test_only]
    /// For a test scenario, register this native asset.
    ///
    /// NOTE: Even though this module is `#[test_only]`, this method is tagged
    /// with the same macro  as a trick to allow another method within this
    /// module to call `init` using OTW.
    public fun init_and_register(scenario: &mut Scenario, caller: address) {
        use token_bridge::token_bridge_scenario::{return_state, take_state};

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Publish coin.
        init(COIN_NATIVE_10 {}, test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let token_bridge_state = take_state(scenario);
        let coin_meta = take_metadata(scenario);

        // Register asset.
        let registry =
            state::borrow_mut_token_registry_test_only(&mut token_bridge_state);
        token_registry::add_new_native_test_only(registry, &coin_meta);

        // Clean up.
        return_state(token_bridge_state);
        return_metadata(coin_meta);
    }

    #[test_only]
    public fun init_register_and_mint(
        scenario: &mut Scenario,
        caller: address,
        amount: u64
    ): Balance<COIN_NATIVE_10> {
        // First publish and register.
        init_and_register(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Mint.
        balance::create_for_testing(amount)
    }

    #[test_only]
    public fun init_register_and_deposit(
        scenario: &mut Scenario,
        caller: address,
        amount: u64
    ) {
        use token_bridge::token_bridge_scenario::{return_state, take_state};

        let minted = init_register_and_mint(scenario, caller, amount);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let token_bridge_state = take_state(scenario);
        native_asset::deposit_test_only(
            token_registry::borrow_mut_native_test_only(
                state::borrow_mut_token_registry_test_only(
                    &mut token_bridge_state
                )
            ),
            minted
        );

        return_state(token_bridge_state);
    }

    #[test_only]
    public fun init_test_only(ctx: &mut TxContext) {
        init(COIN_NATIVE_10 {}, ctx);
    }

    public fun take_metadata(
        scenario: &Scenario
    ): CoinMetadata<COIN_NATIVE_10> {
        test_scenario::take_shared(scenario)
    }

    public fun return_metadata(
        metadata: CoinMetadata<COIN_NATIVE_10>
    ) {
        test_scenario::return_shared(metadata);
    }

    public fun take_treasury_cap(
        scenario: &Scenario
    ): TreasuryCap<COIN_NATIVE_10> {
        test_scenario::take_shared(scenario)
    }

    public fun return_treasury_cap(
        treasury_cap: TreasuryCap<COIN_NATIVE_10>
    ) {
        test_scenario::return_shared(treasury_cap);
    }

    public fun take_globals(
        scenario: &Scenario
    ): (
        TreasuryCap<COIN_NATIVE_10>,
        CoinMetadata<COIN_NATIVE_10>
    ) {
        (
            take_treasury_cap(scenario),
            take_metadata(scenario)
        )
    }

    public fun return_globals(
        treasury_cap: TreasuryCap<COIN_NATIVE_10>,
        metadata: CoinMetadata<COIN_NATIVE_10>
    ) {
        return_treasury_cap(treasury_cap);
        return_metadata(metadata);
    }
}
