/// This module implements a custom type that keeps track of both native and
/// wrapped assets via dynamic fields. These dynamic fields are keyed off using
/// coin types. This registry lives in `State`.
///
/// See `state` module for more details.
module token_bridge::token_registry {
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

    /// Asset is not registered yet.
    const E_UNREGISTERED: u64 = 0;
    /// Asset is already registered. This only applies to native assets.
    const E_ALREADY_REGISTERED: u64 = 1;
    /// Coin type belongs to a native asset.
    const E_NATIVE_ASSET: u64 = 2;
    /// coin type belongs to a wrapped asset.
    const E_WRAPPED_ASSET: u64 = 3;

    /// This container is used to store native and wrapped assets of coin type
    /// as dynamic fields under its `UID`. It also uses a mechanism to generate
    /// arbitrary token addresses for native assets.
    ///
    /// TODO: Remove `IdRegistry` in favor of using `CoinMetadata` to generate
    /// canonical token address.
    struct TokenRegistry has key, store {
        id: UID,
        native_id_registry: IdRegistry,
        num_wrapped: u64,
        num_native: u64
    }

    /// Wrapper of coin type to act as dynamic field key.
    struct Key<phantom C> has copy, drop, store {}

    /// Create new `TokenRegistry`.
    ///
    /// See `setup` module for more info.
    public(friend) fun new(ctx: &mut TxContext): TokenRegistry {
        TokenRegistry {
            id: object::new(ctx),
            native_id_registry: id_registry::new(),
            num_wrapped: 0,
            num_native: 0
        }
    }

    #[test_only]
    public fun new_test_only(ctx: &mut TxContext): TokenRegistry {
        new(ctx)
    }

    /// Retrieve number of native assets registered.
    public fun num_native(self: &TokenRegistry): u64 {
        self.num_native
    }

    /// Retrieve number of wrapped assets registered.
    public fun num_wrapped(self: &TokenRegistry): u64 {
        self.num_wrapped
    }

    /// Determine whether a particular coin type is registered.
    public fun has<C>(self: &TokenRegistry): bool {
        dynamic_field::exists_(&self.id, Key<C> {})
    }

    /// Determine whether a particular coin type is a registered wrapped asset.
    public fun is_wrapped<C>(self: &TokenRegistry): bool {
        assert!(has<C>(self), E_UNREGISTERED);
        dynamic_field::exists_with_type<Key<C>, WrappedAsset<C>>(
            &self.id,
            Key {}
        )
    }

    /// Assert that this coin type is a wrapped asset.
    public fun assert_wrapped<C>(self: &TokenRegistry) {
        assert!(is_wrapped<C>(self), E_NATIVE_ASSET);
    }

    /// Determine whether a particular coin type is a registered native asset.
    public fun is_native<C>(self: &TokenRegistry): bool {
        // `is_wrapped` asserts that `C` is registered. So if `C` is not
        // wrapped, then it is native.
        !is_wrapped<C>(self)
    }

    /// Assert that this coin type is a native asset.
    public fun assert_native<C>(self: &TokenRegistry) {
        assert!(is_native<C>(self), E_WRAPPED_ASSET);
    }

    /// Add a new wrapped asset to the registry.
    ///
    /// See `state` module for more info.
    public(friend) fun add_new_wrapped<C>(
        self: &mut TokenRegistry,
        token_meta: AssetMeta<C>,
        supply: Supply<C>,
        ctx: &mut TxContext
    ) {
        // NOTE: We do not assert that the coin type has not already been
        // registered using !has<C>(self) because `wrapped_asset::new`
        // consumes `Supply`. This `Supply` is only created once for a particuar
        // coin type via `create_wrapped::prepare_registration`. Because the
        // `Supply` is globally unique and can only be created once, there is no
        // risk that `add_new_wrapped` can be called again on the same coin
        // type.
        dynamic_field::add(
            &mut self.id,
            Key<C> {},
            wrapped_asset::new(token_meta, supply, ctx)
        );
        self.num_wrapped = self.num_wrapped + 1;
    }

    #[test_only]
    public fun add_new_wrapped_test_only<C>(
        self: &mut TokenRegistry,
        token_meta: AssetMeta<C>,
        supply: Supply<C>,
        ctx: &mut TxContext
    ) {
        add_new_wrapped(self, token_meta, supply, ctx)
    }

    /// Update existing wrapped asset's `ForeignMetadata`.
    ///
    /// See `state` module for more info.
    public(friend) fun update_wrapped<C>(
        self: &mut TokenRegistry,
        token_meta: AssetMeta<C>
    ) {
        wrapped_asset::update_metadata(borrow_mut_wrapped(self), token_meta);
    }

    /// Add a new native asset to the registry.
    ///
    /// See `state` module for more info.
    public(friend) fun add_new_native<C>(
        self: &mut TokenRegistry,
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
        self: &mut TokenRegistry,
        metadata: &CoinMetadata<C>
    ) {
        add_new_native(self, metadata)
    }

    /// For wrapped assets, burn a given `Balance`. `Balance` originates from
    /// an outbound token transfer.
    ///
    /// See `transfer_tokens` module for more info.
    public(friend) fun burn<C>(
        self: &mut TokenRegistry,
        burned: Balance<C>
    ): u64 {
        wrapped_asset::burn_balance<C>(borrow_mut_wrapped(self), burned)
    }

    #[test_only]
    public fun burn_test_only<C>(
        self: &mut TokenRegistry,
        burned: Balance<C>
    ): u64 {
        burn(self, burned)
    }

    /// For wrapped assets, mint a given amount. This amount is determined by an
    /// inbound token transfer payload.
    ///
    /// See `complete_transfer` module for more info.
    public(friend) fun mint<C>(
        self: &mut TokenRegistry,
        amount: u64
    ): Balance<C> {
        wrapped_asset::mint_balance<C>(borrow_mut_wrapped(self), amount)
    }

    #[test_only]
    public fun mint_test_only<C>(
        self: &mut TokenRegistry,
        amount: u64
    ): Balance<C> {
        mint(self, amount)
    }

    /// For native assets, deposit a given `Balance`. `Balance` originates from
    /// an outbound transfer.
    ///
    /// See `transfer_tokens` module for more info.
    public(friend) fun deposit<C>(
        self: &mut TokenRegistry,
        deposited: Balance<C>
    ) {
        native_asset::deposit_balance<C>(borrow_mut_native(self), deposited)
    }

    #[test_only]
    public fun deposit_test_only<C>(
        self: &mut TokenRegistry,
        deposited: Balance<C>
    ) {
        deposit(self, deposited)
    }

    /// For native assets, withdraw a given amount. This amount is determined by
    /// an inbound token transfer payload.
    public(friend) fun withdraw<C>(
        self: &mut TokenRegistry,
        amount: u64
    ): Balance<C> {
        native_asset::withdraw_balance(borrow_mut_native(self), amount)
    }

    #[test_only]
    public fun withdraw_test_only<C>(
        self: &mut TokenRegistry,
        amount: u64
    ): Balance<C> {
        withdraw(self, amount)
    }

    /// Retrieve custodied `Balance` for a native asset.
    public fun balance<C>(self: &TokenRegistry): u64 {
        native_asset::balance(borrow_native<C>(self))
    }

    /// Retrieve total minted `Supply` value for a wrapped asset.
    public fun total_supply<C>(self: &TokenRegistry): u64 {
        wrapped_asset::total_supply(borrow_wrapped<C>(self))
    }

    /// Retrieve specified decimals for either native or wrapped asset.
    public fun decimals<C>(self: &TokenRegistry): u8 {
        if (is_wrapped<C>(self)) {
            wrapped_asset::decimals(borrow_wrapped_unchecked<C>(self))
        } else {
            native_asset::decimals(borrow_native_unchecked<C>(self))
        }
    }

    /// Retrieve canonical token info for either native or wrapped asset.
    public fun canonical_info<C>(
        self: &TokenRegistry
    ): (u16, ExternalAddress) {
        if (is_wrapped<C>(self)) {
            wrapped_asset::canonical_info(borrow_wrapped_unchecked<C>(self))
        } else {
            native_asset::canonical_info(borrow_native_unchecked<C>(self))
        }
    }

    #[test_only]
    public fun destroy(registry: TokenRegistry) {
        let TokenRegistry {
            id: id,
            native_id_registry,
            num_wrapped: _,
            num_native: _
        } = registry;
        object::delete(id);
        id_registry::destroy(native_id_registry);
    }

    fun borrow_wrapped_unchecked<C>(self: &TokenRegistry): &WrappedAsset<C> {
        dynamic_field::borrow(&self.id, Key<C> {})
    }

    fun borrow_wrapped<C>(self: &TokenRegistry): &WrappedAsset<C> {
        assert_wrapped<C>(self);
        borrow_wrapped_unchecked(self)
    }

    fun borrow_mut_wrapped<C>(
        self: &mut TokenRegistry
    ): &mut WrappedAsset<C> {
        assert_wrapped<C>(self);
        dynamic_field::borrow_mut(&mut self.id, Key<C> {})
    }

    fun borrow_native_unchecked<C>(self: &TokenRegistry): &NativeAsset<C> {
        dynamic_field::borrow(&self.id, Key<C> {})
    }

    fun borrow_native<C>(self: &TokenRegistry): &NativeAsset<C> {
        assert_native<C>(self);
        borrow_native_unchecked(self)
    }

    fun borrow_mut_native<C>(self: &mut TokenRegistry): &mut NativeAsset<C> {
        assert_native<C>(self);
        dynamic_field::borrow_mut(&mut self.id, Key<C> {})
    }
}

// In this test, we exercise the various functionalities of TokenRegistry,
// including registering native and wrapped coins via add_new_native, and
// add_new_wrapped, minting/burning/depositing/withdrawing said tokens, and also
// storing metadata about the tokens.
#[test_only]
module token_bridge::token_registry_tests {
    use sui::balance::{Self};
    use sui::test_scenario::{Self};
    use wormhole::external_address::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::asset_meta::{Self};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::coin_wrapped_7::{Self, COIN_WRAPPED_7};
    use token_bridge::token_registry::{Self};
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
        let registry =
            token_registry::new_test_only(test_scenario::ctx(scenario));

        // Check initial state.
        assert!(token_registry::num_native(&registry) == 0, 0);
        assert!(token_registry::num_wrapped(&registry) == 0, 0);

        // Register native asset.
        let coin_meta = coin_native_10::take_metadata(scenario);
        token_registry::add_new_native_test_only(
            &mut registry,
            &coin_meta,
        );
        coin_native_10::return_metadata(coin_meta);

        // mint some native coins, then deposit them into the token registry
        let deposit_amount = 69;
        let (i, n) = (0, 8);
        while (i < n) {
            token_registry::deposit_test_only(
                &mut registry,
                balance::create_for_testing<COIN_NATIVE_10>(
                    deposit_amount
                )
            );
            i = i + 1;
        };
        let total_deposited = n * deposit_amount;
        assert!(
            token_registry::balance<COIN_NATIVE_10>(&registry) == total_deposited,
            0
        );

        // Withdraw and check balances.
        let withdraw_amount = 420;
        let withdrawn =
            token_registry::withdraw_test_only<COIN_NATIVE_10>(
                &mut registry,
                withdraw_amount
            );
        assert!(balance::value(&withdrawn) == withdraw_amount, 0);
        balance::destroy_for_testing(withdrawn);

        let expected_remaining = total_deposited - withdraw_amount;
        let remaining =
            token_registry::balance<COIN_NATIVE_10>(
                &registry
            );
        assert!(remaining == expected_remaining, 0);

        // Verify registry values.
        assert!(token_registry::num_native(&registry) == 1, 0);
        assert!(token_registry::num_wrapped(&registry) == 0, 0);
        assert!(
            token_registry::is_native<COIN_NATIVE_10>(&registry),
            0
        );
        token_registry::assert_native<COIN_NATIVE_10>(&registry);

        assert!(
            !token_registry::is_wrapped<COIN_NATIVE_10>(&registry),
            0
        );
        assert!(
            token_registry::decimals<COIN_NATIVE_10>(&registry) == 10,
            0
        );

        let (token_chain, token_address) =
            token_registry::canonical_info<COIN_NATIVE_10>(
                &registry
            );
        assert!(token_chain == chain_id(), 0);
        assert!(token_address == external_address::from_any_bytes(x"01"), 0);

        // Clean up.
        token_registry::destroy(registry);

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
        let registry =
            token_registry::new_test_only(test_scenario::ctx(scenario));

        // Check initial state.
        assert!(token_registry::num_wrapped(&registry) == 0, 0);
        assert!(token_registry::num_native(&registry) == 0, 0);

        // Register wrapped asset.
        let wrapped_token_meta = coin_wrapped_7::token_meta();
        token_registry::add_new_wrapped_test_only(
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
                token_registry::mint_test_only<COIN_WRAPPED_7>(
                    &mut registry,
                    mint_amount
                );
            assert!(balance::value(&minted) == mint_amount, 0);
            balance::join(&mut total_minted, minted);
            i = i + 1;
        };

        let total_supply =
            token_registry::total_supply<COIN_WRAPPED_7>(
                &registry
            );
        assert!(total_supply == balance::value(&total_minted), 0);

        // withdraw, check value, and re-deposit native coins into registry
        let burn_amount = 69;
        let burned =
            token_registry::burn_test_only(
                &mut registry,
                balance::split(&mut total_minted, burn_amount)
            );
        assert!(burned == burn_amount, 0);

        let expected_remaining = total_supply - burn_amount;
        let remaining =
            token_registry::total_supply<COIN_WRAPPED_7>(
                &registry
            );
        assert!(remaining == expected_remaining, 0);
        balance::destroy_for_testing(total_minted);

        // Verify registry values.
        assert!(token_registry::num_wrapped(&registry) == 1, 0);
        assert!(token_registry::num_native(&registry) == 0, 0);
        assert!(
            token_registry::is_wrapped<COIN_WRAPPED_7>(&registry),
            0
        );
        token_registry::assert_wrapped<COIN_WRAPPED_7>(&registry);

        assert!(
            !token_registry::is_native<COIN_WRAPPED_7>(&registry),
            0
        );
        assert!(
            token_registry::decimals<COIN_WRAPPED_7>(&registry) == 7,
            0
        );

        let (token_chain, token_address) =
            token_registry::canonical_info<COIN_WRAPPED_7>(
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
        token_registry::destroy(registry);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = token_registry::E_ALREADY_REGISTERED)]
    /// In this negative test case, we try to register a native token twice.
    fun test_cannot_add_new_native_again() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize new coin.
        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Initialize new token registry.
        let registy =
            token_registry::new_test_only(test_scenario::ctx(scenario));

        let coin_meta = coin_native_10::take_metadata(scenario);

        // Add new native asset.
        token_registry::add_new_native_test_only(
            &mut registy,
            &coin_meta
        );

        // You shall not pass!
        token_registry::add_new_native_test_only(
            &mut registy,
            &coin_meta
        );

        // Clean up.
        coin_native_10::return_metadata(coin_meta);
        token_registry::destroy(registy);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = token_registry::E_WRAPPED_ASSET)]
    // In this negative test case, we attempt to deposit a wrapped token into
    // a TokenRegistry object, resulting in failure. A wrapped coin can
    // only be minted and burned, not deposited.
    fun test_cannot_deposit_wrapped_asset() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let supply = coin_wrapped_7::init_and_take_supply(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Initialize new token registry.
        let registry =
            token_registry::new_test_only(test_scenario::ctx(scenario));

        token_registry::add_new_wrapped_test_only(
            &mut registry,
            coin_wrapped_7::token_meta(),
            supply,
            test_scenario::ctx(scenario)
        );

        // Mint some wrapped coins and attempt to deposit balance.
        let minted =
            token_registry::mint_test_only<COIN_WRAPPED_7>(
                &mut registry,
                420420420
            );
        // the line below will fail
        token_registry::deposit_test_only(&mut registry, minted);

        // Clean up.
        token_registry::destroy(registry);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = token_registry::E_NATIVE_ASSET)]
    // In this negative test case, we attempt to deposit a wrapped token into
    // a TokenRegistry object, resulting in failure. A wrapped coin can
    // only be minted and burned, not deposited.
    fun test_cannot_mint_native_asset() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Initialize new token registry.
        let registry =
            token_registry::new_test_only(test_scenario::ctx(scenario));

        let coin_meta = coin_native_10::take_metadata(scenario);
        token_registry::add_new_native_test_only(
            &mut registry,
            &coin_meta
        );

        // the line below will fail
        let minted =
            token_registry::mint_test_only<COIN_NATIVE_10>(
                &mut registry,
                420
            );

        // Clean up.
        coin_native_10::return_metadata(coin_meta);
        balance::destroy_for_testing(minted);
        token_registry::destroy(registry);

        // Done.
        test_scenario::end(my_scenario);
    }
}
