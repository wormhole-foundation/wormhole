module token_bridge::registered_tokens {
    use sui::balance::{Balance, Supply};
    use sui::coin::{CoinMetadata};
    use sui::dynamic_field::{Self};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::id_registry::{Self, IdRegistry};

    use token_bridge::asset_meta::{AssetMeta};
    use token_bridge::native_asset::{Self, NativeAsset};
    use token_bridge::wrapped_asset::{Self, WrappedAsset};

    friend token_bridge::state;

    const E_UNREGISTERED: u64 = 0;
    const E_ALREADY_REGISTERED: u64 = 1;
    const E_CANNOT_DEPOSIT_WRAPPED_ASSET: u64 = 2;
    const E_CANNOT_WITHDRAW_WRAPPED_ASSET: u64 = 3;
    const E_CANNOT_GET_TREASURY_CAP_FOR_NON_WRAPPED_COIN: u64 = 4;
    const E_CANNOT_REGISTER_NATIVE_COIN: u64 = 5;
    const E_CANNOT_BURN_NATIVE_ASSET: u64 = 6;
    const E_CANNOT_MINT_NATIVE_ASSET: u64 = 6;

    struct RegisteredTokens has key, store {
        id: UID,
        native_id_registry: IdRegistry,
        num_wrapped: u64,
        num_native: u64
    }

    struct Key<phantom C> has copy, drop, store {}

    public fun new(ctx: &mut TxContext): RegisteredTokens {
        RegisteredTokens {
            id: object::new(ctx),
            native_id_registry: id_registry::new(),
            num_wrapped: 0,
            num_native: 0
        }
    }

    public fun num_native(self: &RegisteredTokens): u64 {
        self.num_native
    }

    public fun num_wrapped(self: &RegisteredTokens): u64 {
        self.num_wrapped
    }

    public fun has<C>(self: &RegisteredTokens): bool {
        dynamic_field::exists_(&self.id, Key<C> {})
    }

    public fun is_wrapped<C>(self: &RegisteredTokens): bool {
        assert!(has<C>(self), E_UNREGISTERED);
        dynamic_field::exists_with_type<Key<C>, WrappedAsset<C>>(
            &self.id,
            Key {}
        )
    }

    public fun is_native<C>(self: &RegisteredTokens): bool {
        // `is_wrapped` asserts that `C` is registered. So if `C` is not
        // wrapped, then it is native.
        !is_wrapped<C>(self)
    }

    public(friend) fun add_new_wrapped<C>(
        self: &mut RegisteredTokens,
        token_meta: AssetMeta,
        supply: Supply<C>,
        ctx: &mut TxContext
    ) {
        dynamic_field::add(
            &mut self.id,
            Key<C> {},
            wrapped_asset::new(token_meta, supply, ctx)
        );
        self.num_wrapped = self.num_wrapped + 1;
    }

    #[test_only]
    public fun add_new_wrapped_test_only<C>(
        self: &mut RegisteredTokens,
        token_meta: AssetMeta,
        supply: Supply<C>,
        ctx: &mut TxContext
    ) {
        // Note: we do not assert that the coin type has not already been
        // registered using !has<C>(self), because add_new_wrapped
        // consumes TreasuryCap<C> and stores it within a WrappedAsset
        // within the token bridge forever. Since the treasury cap
        // is globally unique and can only be created once, there is no
        // risk that add_new_wrapped can be called again on the same
        // coin type.
        add_new_wrapped(self, token_meta, supply, ctx)
    }

    public(friend) fun add_new_native<C>(
        self: &mut RegisteredTokens,
        metadata: &CoinMetadata<C>,
    ) {
        assert!(!has<C>(self), E_ALREADY_REGISTERED);
        let addr = id_registry::next_address(&mut self.native_id_registry);
        dynamic_field::add(
            &mut self.id,
            Key<C> {},
            native_asset::new(addr, metadata)
        );
        self.num_native = self.num_native + 1;
    }

    #[test_only]
    public fun add_new_native_test_only<C>(
        self: &mut RegisteredTokens,
        metadata: &CoinMetadata<C>
    ) {
        add_new_native(self, metadata)
    }

    public(friend) fun burn<C>(
        self: &mut RegisteredTokens,
        burned: Balance<C>
    ): u64 {
        assert!(is_wrapped<C>(self), E_CANNOT_BURN_NATIVE_ASSET);
        wrapped_asset::burn_balance<C>(
            dynamic_field::borrow_mut(&mut self.id, Key<C> {}),
            burned
        )
    }

    #[test_only]
    public fun burn_test_only<C>(
        self: &mut RegisteredTokens,
        burned: Balance<C>
    ): u64 {
        burn(self, burned)
    }

    public(friend) fun mint<C>(
        self: &mut RegisteredTokens,
        amount: u64
    ): Balance<C> {
        assert!(is_wrapped<C>(self), E_CANNOT_MINT_NATIVE_ASSET);
        wrapped_asset::mint_balance<C>(
            dynamic_field::borrow_mut(&mut self.id, Key<C> {}),
            amount
        )
    }

    #[test_only]
    public fun mint_test_only<C>(
        self: &mut RegisteredTokens,
        amount: u64
    ): Balance<C> {
        mint(self, amount)
    }

    public(friend) fun deposit<C>(
        self: &mut RegisteredTokens,
        deposited: Balance<C>
    ) {
        assert!(is_native<C>(self), E_CANNOT_DEPOSIT_WRAPPED_ASSET);
        native_asset::deposit_balance<C>(
            dynamic_field::borrow_mut(&mut self.id, Key<C> {}),
            deposited
        )
    }

    #[test_only]
    public fun deposit_test_only<C>(
        self: &mut RegisteredTokens,
        deposited: Balance<C>
    ) {
        deposit(self, deposited)
    }

    public(friend) fun withdraw<C>(
        self: &mut RegisteredTokens,
        amount: u64
    ): Balance<C> {
        assert!(is_native<C>(self), E_CANNOT_WITHDRAW_WRAPPED_ASSET);
        native_asset::withdraw_balance(
            dynamic_field::borrow_mut(&mut self.id, Key<C> {}),
            amount
        )
    }

    #[test_only]
    public fun withdraw_test_only<C>(
        self: &mut RegisteredTokens,
        amount: u64
    ): Balance<C> {
        withdraw(self, amount)
    }

    public fun balance<C>(self: &RegisteredTokens): u64 {
        native_asset::balance<C>(dynamic_field::borrow(&self.id, Key<C> {}))
    }

    public fun total_supply<C>(self: &RegisteredTokens): u64 {
        wrapped_asset::total_supply<C>(
            dynamic_field::borrow(&self.id, Key<C> {})
        )
    }

    public fun decimals<C>(self: &RegisteredTokens): u8 {
        if (is_wrapped<C>(self)) {
            wrapped_asset::decimals(borrow_wrapped<C>(self))
        } else {
            native_asset::decimals(borrow_native<C>(self))
        }
    }

    public fun canonical_info<C>(
        self: &RegisteredTokens
    ): (u16, ExternalAddress) {
        if (is_wrapped<C>(self)) {
            wrapped_asset::canonical_info(borrow_wrapped<C>(self))
        } else {
            native_asset::canonical_info(borrow_native<C>(self))
        }
    }

    #[test_only]
    public fun destroy(r: RegisteredTokens) {
        let RegisteredTokens {
            id: id,
            native_id_registry,
            num_wrapped: _,
            num_native: _
        } = r;
        object::delete(id);
        id_registry::destroy(native_id_registry);
    }

    fun borrow_wrapped<C>(self: &RegisteredTokens): &WrappedAsset<C> {
        dynamic_field::borrow(&self.id, Key<C> {})
    }

    fun borrow_native<C>(self: &RegisteredTokens): &NativeAsset<C> {
        dynamic_field::borrow(&self.id, Key<C> {})
    }
}

// In this test, we exercise the various functionalities of RegisteredTokens,
// including registering native and wrapped coins via add_new_native, and
// add_new_wrapped, minting/burning/depositing/withdrawing said tokens, and also
// storing metadata about the tokens.
#[test_only]
module token_bridge::registered_tokens_tests {
    use sui::balance::{Self};
    use sui::test_scenario::{Self};
    use wormhole::external_address::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::asset_meta::{Self};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::coin_wrapped_7::{Self, COIN_WRAPPED_7};
    use token_bridge::registered_tokens::{Self};
    use token_bridge::token_bridge_scenario::{person};

    #[test]
    fun test_registered_tokens_native() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize new coin.
        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Initialize new token registry.
        let registry = registered_tokens::new(test_scenario::ctx(scenario));

        // Check initial state.
        assert!(registered_tokens::num_native(&registry) == 0, 0);
        assert!(registered_tokens::num_wrapped(&registry) == 0, 0);

        // Register native asset.
        let coin_meta = coin_native_10::take_metadata(scenario);
        registered_tokens::add_new_native_test_only(
            &mut registry,
            &coin_meta,
        );
        coin_native_10::return_metadata(coin_meta);

        // mint some native coins, then deposit them into the token registry
        let deposit_amount = 69;
        let (i, n) = (0, 8);
        while (i < n) {
            registered_tokens::deposit_test_only(
                &mut registry,
                balance::create_for_testing<COIN_NATIVE_10>(
                    deposit_amount
                )
            );
            i = i + 1;
        };
        let total_deposited = n * deposit_amount;
        assert!(
            registered_tokens::balance<COIN_NATIVE_10>(&registry) == total_deposited,
            0
        );

        // Withdraw and check balances.
        let withdraw_amount = 420;
        let withdrawn =
            registered_tokens::withdraw_test_only<COIN_NATIVE_10>(
                &mut registry,
                withdraw_amount
            );
        assert!(balance::value(&withdrawn) == withdraw_amount, 0);
        balance::destroy_for_testing(withdrawn);

        let expected_remaining = total_deposited - withdraw_amount;
        let remaining =
            registered_tokens::balance<COIN_NATIVE_10>(
                &registry
            );
        assert!(remaining == expected_remaining, 0);

        // Verify registry values.
        assert!(registered_tokens::num_native(&registry) == 1, 0);
        assert!(registered_tokens::num_wrapped(&registry) == 0, 0);
        assert!(
            registered_tokens::is_native<COIN_NATIVE_10>(&registry),
            0
        );
        assert!(
            !registered_tokens::is_wrapped<COIN_NATIVE_10>(&registry),
            0
        );
        assert!(
            registered_tokens::decimals<COIN_NATIVE_10>(&registry) == 10,
            0
        );

        let (token_chain, token_address) =
            registered_tokens::canonical_info<COIN_NATIVE_10>(
                &registry
            );
        assert!(token_chain == chain_id(), 0);
        assert!(token_address == external_address::from_any_bytes(x"01"), 0);

        // Clean up.
        registered_tokens::destroy(registry);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_registered_tokens_wrapped() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize new coin.
        let supply = coin_wrapped_7::init_and_take_supply(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Initialize new token registry.
        let registry = registered_tokens::new(test_scenario::ctx(scenario));

        // Check initial state.
        assert!(registered_tokens::num_wrapped(&registry) == 0, 0);
        assert!(registered_tokens::num_native(&registry) == 0, 0);

        // Register wrapped asset.
        let wrapped_token_meta = coin_wrapped_7::token_meta();
        registered_tokens::add_new_wrapped_test_only(
            &mut registry,
            wrapped_token_meta,
            supply,
            test_scenario::ctx(scenario)
        );

        // Mint wrapped coin via `WrappedAsset` several times.
        let mint_amount = 420;
        let total_minted = balance::zero();
        let (i, n) = (0, 8);
        while (i < n) {
            let minted =
                registered_tokens::mint_test_only<COIN_WRAPPED_7>(
                    &mut registry,
                    mint_amount
                );
            assert!(balance::value(&minted) == mint_amount, 0);
            balance::join(&mut total_minted, minted);
            i = i + 1;
        };

        let total_supply =
            registered_tokens::total_supply<COIN_WRAPPED_7>(
                &registry
            );
        assert!(total_supply == balance::value(&total_minted), 0);

        // withdraw, check value, and re-deposit native coins into registry
        let burn_amount = 69;
        let burned =
            registered_tokens::burn_test_only<COIN_WRAPPED_7>(
                &mut registry,
                balance::split(&mut total_minted, burn_amount)
            );
        assert!(burned == burn_amount, 0);

        let expected_remaining = total_supply - burn_amount;
        let remaining =
            registered_tokens::total_supply<COIN_WRAPPED_7>(
                &registry
            );
        assert!(remaining == expected_remaining, 0);
        balance::destroy_for_testing(total_minted);

        // Verify registry values.
        assert!(registered_tokens::num_wrapped(&registry) == 1, 0);
        assert!(registered_tokens::num_native(&registry) == 0, 0);
        assert!(
            registered_tokens::is_wrapped<COIN_WRAPPED_7>(&registry),
            0
        );
        assert!(
            !registered_tokens::is_native<COIN_WRAPPED_7>(&registry),
            0
        );
        assert!(
            registered_tokens::decimals<COIN_WRAPPED_7>(&registry) == 7,
            0
        );

        let (token_chain, token_address) =
            registered_tokens::canonical_info<COIN_WRAPPED_7>(
                &registry
            );
        assert!(
            token_chain == asset_meta::token_chain(&wrapped_token_meta),
            0
        );
        assert!(
            token_address == asset_meta::token_address(&wrapped_token_meta),
            0
        );

        // Clean up.
        registered_tokens::destroy(registry);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::registered_tokens::E_ALREADY_REGISTERED
    )]
    /// In this negative test case, we try to register a native token twice.
    fun test_registered_tokens_already_registered() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // 1) Initialize RegisteredTokens object, native and wrapped coins.
        test_scenario::next_tx(scenario, caller);{
            //coin_witness::test_init(ctx(scenario));
            coin_native_10::init_test_only(test_scenario::ctx(scenario));
        };
        test_scenario::next_tx(scenario, caller);{
            let registered_tokens = registered_tokens::new(test_scenario::ctx(scenario));

            // 2) Check initial state.
            assert!(registered_tokens::num_wrapped(&registered_tokens)==0, 0);
            assert!(registered_tokens::num_native(&registered_tokens)==0, 0);

            let coin_meta = coin_native_10::take_metadata(scenario);

            // 3)  Attempt to register native coin twice.
            registered_tokens::add_new_native_test_only(
                &mut registered_tokens,
                &coin_meta
            );
            registered_tokens::add_new_native_test_only(
                &mut registered_tokens,
                &coin_meta
            );

            //4) Cleanup.
            coin_native_10::return_metadata(coin_meta);
            registered_tokens::destroy(registered_tokens);
        };

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = registered_tokens::E_CANNOT_DEPOSIT_WRAPPED_ASSET
    )]
    // In this negative test case, we attempt to deposit a wrapped token into
    // a RegisteredTokens object, resulting in failure. A wrapped coin can
    // only be minted and burned, not deposited.
    fun test_registered_tokens_deposit_wrapped_fail() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let supply = coin_wrapped_7::init_and_take_supply(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Initialize new token registry.
        let registry = registered_tokens::new(test_scenario::ctx(scenario));

        registered_tokens::add_new_wrapped_test_only(
            &mut registry,
            coin_wrapped_7::token_meta(),
            supply,
            test_scenario::ctx(scenario)
        );

        // Mint some wrapped coins and attempt to deposit balance.
        let minted =
            registered_tokens::mint_test_only<COIN_WRAPPED_7>(
                &mut registry,
                420420420
            );
        // the line below will fail
        registered_tokens::deposit_test_only<COIN_WRAPPED_7>(
            &mut registry,
            minted
        );

        // Clean up.
        registered_tokens::destroy(registry);

        // Done.
        test_scenario::end(my_scenario);
    }
}
