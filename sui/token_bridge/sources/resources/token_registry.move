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
    use wormhole::state::{chain_id};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::native_asset::{Self, NativeAsset};
    use token_bridge::wrapped_asset::{Self, WrappedAsset};

    friend token_bridge::attest_token;
    friend token_bridge::complete_transfer;
    friend token_bridge::create_wrapped;
    friend token_bridge::state;
    friend token_bridge::transfer_tokens;

    /// Asset is not registered yet.
    const E_UNREGISTERED: u64 = 0;
    /// Asset is already registered. This only applies to native assets.
    const E_ALREADY_REGISTERED: u64 = 1;
    /// Coin type belongs to a native asset.
    const E_NATIVE_ASSET: u64 = 2;
    /// coin type belongs to a wrapped asset.
    const E_WRAPPED_ASSET: u64 = 3;
    /// Input token info does not match registered info.
    const E_CANONICAL_TOKEN_INFO_MISMATCH: u64 = 4;

    /// This container is used to store native and wrapped assets of coin type
    /// as dynamic fields under its `UID`. It also uses a mechanism to generate
    /// arbitrary token addresses for native assets.
    struct TokenRegistry has key, store {
        id: UID,
        num_wrapped: u64,
        num_native: u64
    }

    /// Container to provide convenient checking of whether an asset is wrapped
    /// or native. `AssetCap` can only be created either by passing in a
    /// resource with `CoinType` or by verifying input token info against the
    /// canonical info that exists in `TokenRegistry`.
    ///
    /// NOTE: This container can be dropped after it was created.
    struct AssetCap<phantom CoinType> has drop {
        is_wrapped: bool
    }

    /// Wrapper of coin type to act as dynamic field key.
    struct Key<phantom CoinType> has copy, drop, store {}

    /// Create new `TokenRegistry`.
    ///
    /// See `setup` module for more info.
    public(friend) fun new(ctx: &mut TxContext): TokenRegistry {
        TokenRegistry {
            id: object::new(ctx),
            num_wrapped: 0,
            num_native: 0
        }
    }

    #[test_only]
    public fun new_test_only(ctx: &mut TxContext): TokenRegistry {
        new(ctx)
    }

    /// Create an `AssetCap` by verifying input token info. If the combination
    /// of token chain ID and address do not match with what exists in the
    /// registry, this method aborts.
    public fun verify_for_asset_cap<CoinType>(
        self: &TokenRegistry,
        token_chain: u16,
        token_address: ExternalAddress
    ): AssetCap<CoinType> {
        let (chain, addr) = canonical_info<CoinType>(self);
        assert!(
            token_chain == chain && addr == token_address,
            E_CANONICAL_TOKEN_INFO_MISMATCH
        );

        AssetCap { is_wrapped: token_chain != chain_id() }
    }

    /// With an `AssetCap`, retrieve canonical token info for either native or
    /// wrapped asset.
    public fun checked_canonical_info<CoinType>(
        cap: &AssetCap<CoinType>,
        self: &TokenRegistry
    ): (u16, ExternalAddress) {
        if (is_wrapped_cap(cap)) {
            wrapped_asset::canonical_info(
                borrow_wrapped_unchecked<CoinType>(self)
            )
        } else {
            native_asset::canonical_info(
                borrow_native_unchecked<CoinType>(self)
            )
        }
    }

    /// Retrieve canonical token info for either native or wrapped asset.
    public fun canonical_info<CoinType>(
        self: &TokenRegistry
    ): (u16, ExternalAddress) {
        checked_canonical_info(&new_asset_cap<CoinType>(self), self)
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
    public fun has<CoinType>(self: &TokenRegistry): bool {
        dynamic_field::exists_(&self.id, Key<CoinType> {})
    }

    /// Create `AssetCap` using the `CoinType` of `Balance`.
    public fun asset_cap_from_balance<CoinType>(
        self: &TokenRegistry,
        _: &Balance<CoinType>
    ): AssetCap<CoinType> {
        new_asset_cap<CoinType>(self)
    }

    /// Create `AssetCap` using the `CoinType` of `CoinMetadata`. This
    /// `AssetCap` should always reflect that the given `CoinType` is native.
    public fun asset_cap_from_coin_metadata<CoinType>(
        self: &TokenRegistry,
        _: &CoinMetadata<CoinType>
    ): AssetCap<CoinType> {
        let cap = new_asset_cap<CoinType>(self);

        // Verify that the `AssetCap` reflects a native asset to be safe.
        assert_native_cap(&cap);

        cap
    }

    /// Determine whether a given `CoinType` is a wrapped asset.
    public fun is_wrapped<CoinType>(self: &TokenRegistry): bool {
        is_wrapped_cap(&new_asset_cap<CoinType>(self))
    }

    /// Determine whether a given `CoinType` is a native asset.
    public fun is_native<CoinType>(self: &TokenRegistry): bool {
        !is_wrapped<CoinType>(self)
    }

    /// With an `AssetCap`, retrieve the canonical token address for a given
    /// `CoinType`.
    public fun checked_token_address<CoinType>(
        cap: &AssetCap<CoinType>,
        self: &TokenRegistry
    ): ExternalAddress {
        if (is_wrapped_cap(cap)) {
            wrapped_asset::token_address(
                wrapped_asset::metadata(
                    borrow_wrapped_unchecked<CoinType>(self)
                )
            )
        } else {
            native_asset::token_address(
                borrow_native_unchecked<CoinType>(self)
            )
        }
    }

    /// Add a new wrapped asset to the registry and return the canonical token
    /// address.
    ///
    /// See `state` module for more info.
    public(friend) fun add_new_wrapped<CoinType>(
        self: &mut TokenRegistry,
        token_meta: AssetMeta,
        supply: Supply<CoinType>,
        ctx: &mut TxContext
    ): ExternalAddress {
        // Grab canonical token address for return value.
        let token_addr = asset_meta::token_address(&token_meta);

        // NOTE: We do not assert that the coin type has not already been
        // registered using !has<CoinType>(self) because `wrapped_asset::new`
        // consumes `Supply`. This `Supply` is only created once for a particuar
        // coin type via `create_wrapped::prepare_registration`. Because the
        // `Supply` is globally unique and can only be created once, there is no
        // risk that `add_new_wrapped` can be called again on the same coin
        // type.
        let asset = wrapped_asset::new(token_meta, supply, ctx);
        dynamic_field::add(&mut self.id, Key<CoinType> {}, asset);
        self.num_wrapped = self.num_wrapped + 1;

        token_addr
    }

    #[test_only]
    public fun add_new_wrapped_test_only<CoinType>(
        self: &mut TokenRegistry,
        token_meta: AssetMeta,
        supply: Supply<CoinType>,
        ctx: &mut TxContext
    ): ExternalAddress {
        add_new_wrapped(self, token_meta, supply, ctx)
    }

    /// Update existing wrapped asset's `ForeignMetadata`.
    ///
    /// See `state` module for more info.
    public(friend) fun update_wrapped<CoinType>(
        self: &mut TokenRegistry,
        token_meta: AssetMeta
    ) {
        // NOTE: This checks canonical token info, so we do not need `AssetCap`.
        // And because of this, we only want to check if the asset is registered
        // at this point.
        assert_has<CoinType>(self);
        wrapped_asset::update_metadata(
            borrow_mut_wrapped_unchecked<CoinType>(self),
            token_meta
        );
    }

    /// Add a new native asset to the registry and return the canonical token
    /// address.
    ///
    /// NOTE: This method does not verify if `CoinType` is already in the
    /// registry because `attest_token` already takes care of this check. If
    /// This method were to be called on an already-registered asset, this
    /// will throw with an error from `sui::dynamic_field` reflectina duplicate
    /// field.
    ///
    /// See `attest_token` module for more info.
    public(friend) fun add_new_native<CoinType>(
        self: &mut TokenRegistry,
        metadata: &CoinMetadata<CoinType>,
    ): ExternalAddress {
        // Create new native asset.
        let asset = native_asset::new(metadata);
        let token_addr = native_asset::token_address(&asset);

        // Add to registry.
        dynamic_field::add(&mut self.id, Key<CoinType> {}, asset);
        self.num_native = self.num_native + 1;

        // Return the token address.
        token_addr
    }

    #[test_only]
    public fun add_new_native_test_only<CoinType>(
        self: &mut TokenRegistry,
        metadata: &CoinMetadata<CoinType>
    ): ExternalAddress {
        add_new_native(self, metadata)
    }

    /// Either mint wrapped assets or withdraw native assets from the registry's
    /// native balance custody.
    ///
    /// NOTE: Only a holder of `AssetCap` can use this method.
    ///
    /// See `complete_transfer` module for more info.
    public(friend) fun put_into_circulation<CoinType>(
        cap: &AssetCap<CoinType>,
        self: &mut TokenRegistry,
        amount: u64
    ): Balance<CoinType> {
        if (is_wrapped_cap(cap)) {
            mint(self, amount)
        } else {
            withdraw(self, amount)
        }
    }

    #[test_only]
    public fun put_into_circulation_test_only<CoinType>(
        self: &mut TokenRegistry,
        amount: u64
    ): Balance<CoinType> {
        put_into_circulation(&new_asset_cap<CoinType>(self), self, amount)
    }

    /// Either burn wrapped assets or deposit native assets into the registry's
    /// native balance custody.
    ///
    /// NOTE: Only a holder of `AssetCap` can use this method.
    ///
    /// See `transfer_tokens` module for more info.
    public(friend) fun take_from_circulation<CoinType>(
        cap: &AssetCap<CoinType>,
        self: &mut TokenRegistry,
        bridged_in: Balance<CoinType>
    ) {
        if (is_wrapped_cap(cap)) {
            burn(self, bridged_in);
        } else {
            deposit(self, bridged_in);
        }
    }

    #[test_only]
    public fun take_from_circulation_test_only<CoinType>(
        self: &mut TokenRegistry,
        bridged_in: Balance<CoinType>
    ) {
        take_from_circulation(&new_asset_cap<CoinType>(self), self, bridged_in)
    }

    /// Retrieve custodied `Balance` for a native asset.
    public fun native_balance<CoinType>(self: &TokenRegistry): u64 {
        checked_native_balance(&new_asset_cap<CoinType>(self), self)
    }

    /// With an `AssetCap`, retrieve custodied `Balance` for a native asset.
    public fun checked_native_balance<CoinType>(
        cap: &AssetCap<CoinType>,
        self: &TokenRegistry
    ): u64 {
        assert_native_cap(cap);
        native_asset::balance(borrow_native_unchecked<CoinType>(self))
    }

    /// Retrieve total minted `Supply` value for a wrapped asset.
    public fun wrapped_supply<CoinType>(self: &TokenRegistry): u64 {
        checked_wrapped_supply(&new_asset_cap<CoinType>(self), self)
    }

    /// With an `AssetCap`, retrieve total minted `Supply` value for a wrapped
    /// asset.
    public fun checked_wrapped_supply<CoinType>(
        cap: &AssetCap<CoinType>,
        self: &TokenRegistry
    ): u64 {
        assert_wrapped_cap(cap);
        wrapped_asset::total_supply(borrow_wrapped_unchecked<CoinType>(self))
    }

    /// Retrieve specified decimals for either native or wrapped asset.
    public fun checked_decimals<CoinType>(
        cap: &AssetCap<CoinType>,
        self: &TokenRegistry
    ): u8 {
        if (is_wrapped_cap(cap)) {
            wrapped_asset::decimals(borrow_wrapped_unchecked<CoinType>(self))
        } else {
            native_asset::decimals(borrow_native_unchecked<CoinType>(self))
        }
    }

    public fun decimals<CoinType>(self: &TokenRegistry): u8 {
        checked_decimals(&new_asset_cap<CoinType>(self), self)
    }

    #[test_only]
    public fun destroy(registry: TokenRegistry) {
        let TokenRegistry {
            id,
            num_wrapped: _,
            num_native: _
        } = registry;
        object::delete(id);
    }

    #[test_only]
    public fun borrow_wrapped<CoinType>(
        self: &TokenRegistry
    ): &WrappedAsset<CoinType> {
        assert_wrapped_cap(&new_asset_cap<CoinType>(self));
        borrow_wrapped_unchecked<CoinType>(self)
    }

    #[test_only]
    public fun borrow_native<CoinType>(
        self: &TokenRegistry
    ): &NativeAsset<CoinType> {
        assert_native_cap(&new_asset_cap<CoinType>(self));
        borrow_native_unchecked<CoinType>(self)
    }

    fun borrow_wrapped_unchecked<CoinType>(
        self: &TokenRegistry
    ): &WrappedAsset<CoinType> {
        dynamic_field::borrow(&self.id, Key<CoinType> {})
    }

    fun borrow_mut_wrapped_unchecked<CoinType>(
        self: &mut TokenRegistry
    ): &mut WrappedAsset<CoinType> {
        dynamic_field::borrow_mut(&mut self.id, Key<CoinType> {})
    }

    fun borrow_native_unchecked<CoinType>(
        self: &TokenRegistry
    ): &NativeAsset<CoinType> {
        dynamic_field::borrow(&self.id, Key<CoinType> {})
    }

    fun borrow_mut_native_unchecked<CoinType>(
        self: &mut TokenRegistry
    ): &mut NativeAsset<CoinType> {
        dynamic_field::borrow_mut(&mut self.id, Key<CoinType> {})
    }

    fun assert_has<CoinType>(self: &TokenRegistry) {
        assert!(has<CoinType>(self), E_UNREGISTERED);
    }

    fun new_asset_cap<CoinType>(
        self: &TokenRegistry
    ): AssetCap<CoinType> {
        assert_has<CoinType>(self);

        // We check specifically whether `CoinType` is associated with a dynamic
        // field for `WrappedAsset`. This boolean will be used as the underlying
        // value for `AssetCap`.
        let is_wrapped =
            dynamic_field::exists_with_type<Key<CoinType>, WrappedAsset<CoinType>>(
                &self.id,
                Key {}
            );
        AssetCap { is_wrapped }
    }

    fun is_wrapped_cap<CoinType>(cap: &AssetCap<CoinType>): bool {
        cap.is_wrapped
    }

    fun assert_wrapped_cap<CoinType>(cap: &AssetCap<CoinType>) {
        assert!(cap.is_wrapped, E_NATIVE_ASSET);
    }

    fun assert_native_cap<CoinType>(cap: &AssetCap<CoinType>) {
        assert!(!cap.is_wrapped, E_WRAPPED_ASSET);
    }

    /// For wrapped assets, mint a given amount. This amount is determined by an
    /// inbound token transfer payload.
    ///
    /// See `complete_transfer` module for more info.
    fun mint<CoinType>(
        self: &mut TokenRegistry,
        amount: u64
    ): Balance<CoinType> {
        wrapped_asset::mint_balance(
            borrow_mut_wrapped_unchecked(self),
            amount
        )
    }

    #[test_only]
    public fun mint_test_only<CoinType>(
        self: &mut TokenRegistry,
        amount: u64
    ): Balance<CoinType> {
        mint(self, amount)
    }

    /// For native assets, withdraw a given amount. This amount is determined by
    /// an inbound token transfer payload.
    fun withdraw<CoinType>(
        self: &mut TokenRegistry,
        amount: u64
    ): Balance<CoinType> {
        native_asset::withdraw_balance(
            borrow_mut_native_unchecked(self),
            amount
        )
    }

    #[test_only]
    public fun withdraw_test_only<CoinType>(
        self: &mut TokenRegistry,
        amount: u64
    ): Balance<CoinType> {
        withdraw(self, amount)
    }

    /// For native assets, deposit a given `Balance`. `Balance` originates from
    /// an outbound transfer.
    ///
    /// See `transfer_tokens` module for more info.
    public(friend) fun deposit<CoinType>(
        self: &mut TokenRegistry,
        deposited: Balance<CoinType>
    ) {
        native_asset::deposit_balance(
            borrow_mut_native_unchecked(self), deposited
        )
    }

    #[test_only]
    public fun deposit_test_only<CoinType>(
        self: &mut TokenRegistry,
        deposited: Balance<CoinType>
    ) {
        deposit(self, deposited)
    }

    /// For wrapped assets, burn a given `Balance`. `Balance` originates from
    /// an outbound token transfer.
    ///
    /// See `transfer_tokens` module for more info.
    fun burn<CoinType>(
        self: &mut TokenRegistry,
        burned: Balance<CoinType>
    ): u64 {
        wrapped_asset::burn_balance(
            borrow_mut_wrapped_unchecked(self),
            burned
        )
    }

    #[test_only]
    public fun burn_test_only<CoinType>(
        self: &mut TokenRegistry,
        burned: Balance<CoinType>
    ): u64 {
        burn(self, burned)
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
    use wormhole::state::{chain_id};

    use token_bridge::asset_meta::{Self};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::coin_wrapped_7::{Self, COIN_WRAPPED_7};
    use token_bridge::native_asset::{Self};
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
        let token_address =
            token_registry::add_new_native_test_only(
                &mut registry,
                &coin_meta,
            );
        let expected_token_address =
            native_asset::canonical_address(&coin_meta);
        assert!(token_address == expected_token_address, 0);

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
            token_registry::native_balance<COIN_NATIVE_10>(&registry) == total_deposited,
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
            token_registry::native_balance<COIN_NATIVE_10>(
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
        assert!(
            !token_registry::is_wrapped<COIN_NATIVE_10>(&registry),
            0
        );
        assert!(
            token_registry::decimals<COIN_NATIVE_10>(&registry) == 10,
            0
        );

        let (
            token_chain,
            token_address
        ) = token_registry::canonical_info<COIN_NATIVE_10>(&registry);
        assert!(token_chain == chain_id(), 0);
        assert!(token_address == expected_token_address, 0);

        // Clean up.
        token_registry::destroy(registry);
        coin_native_10::return_metadata(coin_meta);

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
            token_registry::wrapped_supply<COIN_WRAPPED_7>(
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
            token_registry::wrapped_supply<COIN_WRAPPED_7>(
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

        assert!(
            !token_registry::is_native<COIN_WRAPPED_7>(&registry),
            0
        );
        assert!(
            token_registry::decimals<COIN_WRAPPED_7>(&registry) == 7,
            0
        );

        let wrapped_token_meta = coin_wrapped_7::token_meta();
        let (
            token_chain,
            token_address
        ) = token_registry::canonical_info<COIN_WRAPPED_7>(&registry);
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
        asset_meta::destroy(wrapped_token_meta);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = sui::dynamic_field::EFieldAlreadyExists)]
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
        //
        // NOTE: We don't have a custom error for this. This will trigger a
        // `sui::dynamic_field` error.
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
    #[expected_failure(abort_code = sui::dynamic_field::EFieldTypeMismatch)]
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

        assert!(!token_registry::is_native<COIN_WRAPPED_7>(&registry), 0);

        // You shall not pass!
        //
        // NOTE: We don't have a custom error for this. This will trigger a
        // `sui::dynamic_field` error.
        token_registry::deposit_test_only(&mut registry, minted);

        // Clean up.
        token_registry::destroy(registry);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = sui::dynamic_field::EFieldTypeMismatch)]
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

        // Show that this asset is not wrapped.
        assert!(!token_registry::is_wrapped<COIN_NATIVE_10>(&registry), 0);

        // You shall not pass!
        //
        // NOTE: We don't have a custom error for this. This will trigger a
        // `sui::dynamic_field` error.
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
