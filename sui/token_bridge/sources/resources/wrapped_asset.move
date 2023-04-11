// SPDX-License-Identifier: Apache 2

/// This module implements two custom types relating to Token Bridge wrapped
/// assets. These assets have been attested from foreign networks, whose
/// metadata is stored in `ForeignMetadata`. The Token Bridge contract is the
/// only authority that can mint and burn these assets via `Supply`.
///
/// See `create_wrapped` and 'token_registry' modules for more details.
module token_bridge::wrapped_asset {
    use std::string::{String};
    use sui::balance::{Self, Balance, Supply};
    use sui::object::{Self, UID};
    use sui::package::{Self, UpgradeCap};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{chain_id};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::normalized_amount::{cap_decimals};

    friend token_bridge::complete_transfer;
    friend token_bridge::create_wrapped;
    friend token_bridge::token_registry;
    friend token_bridge::transfer_tokens;

    /// Token chain ID matching Sui's are not allowed.
    const E_SUI_CHAIN: u64 = 0;
    /// Canonical token info does match `AssetMeta` payload.
    const E_ASSET_META_MISMATCH: u64 = 1;

    /// Container storing foreign asset info.
    struct ForeignMetadata<phantom C> has key, store {
        id: UID,
        token_chain: u16,
        token_address: ExternalAddress,
        native_decimals: u8,
        symbol: String,
        name: String
    }

    /// Container managing `ForeignMetadata` and `Supply` for a wrapped asset
    /// coin type.
    struct WrappedAsset<phantom C> has store {
        metadata: ForeignMetadata<C>,
        total_supply: Supply<C>,
        decimals: u8,
        upgrade_cap: UpgradeCap
    }

    /// Create new `WrappedAsset`.
    ///
    /// See `token_registry` module for more info.
    public(friend) fun new<C>(
        token_meta: AssetMeta,
        supply: Supply<C>,
        upgrade_cap: UpgradeCap,
        ctx: &mut TxContext
    ): WrappedAsset<C> {
        // Verify that the upgrade cap is from the same package as coin type.
        // This cap should not have been modified prior to creating this asset
        // (i.e. should have the default upgrade policy and build version == 1).
        wormhole::package_utils::assert_package_upgrade_cap<C>(
            &upgrade_cap,
            package::compatible_policy(),
            1
        );

        let (
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        ) = asset_meta::unpack(token_meta);

        // Protect against adding `AssetMeta` which has Sui's chain ID.
        assert!(token_chain != chain_id(), E_SUI_CHAIN);

        let metadata =
            ForeignMetadata {
                id: object::new(ctx),
                token_address,
                token_chain,
                native_decimals,
                symbol,
                name
            };

        WrappedAsset {
            metadata,
            total_supply: supply,
            decimals: cap_decimals(native_decimals),
            upgrade_cap
        }
    }

    #[test_only]
    public fun new_test_only<C>(
        token_meta: AssetMeta,
        supply: Supply<C>,
        upgrade_cap: UpgradeCap,
        ctx: &mut TxContext
    ): WrappedAsset<C> {
        new(token_meta, supply, upgrade_cap, ctx)
    }

    /// Update existing `ForeignMetadata` using new `AssetMeta`.
    ///
    /// See `token_registry` module for more info.
    public(friend) fun update_metadata<C>(
        self: &mut WrappedAsset<C>,
        token_meta: AssetMeta
    ) {
        // NOTE: We ignore `native_decimals` because we do not enforce that
        // an asset's decimals on a foreign network needs to stay the same.
        let (
            token_address,
            token_chain,
            _native_decimals,
            symbol,
            name
        ) = asset_meta::unpack(token_meta);

        // Verify canonical token info. Also check that the native decimals
        // have not changed (because changing this info is not desirable, as
        // this change means the supply changed on its native network).
        //
        // NOTE: This implicitly verifies that `token_chain` is not Sui's
        // because this was checked already when the asset was first added.
        let (expected_chain, expected_address) = canonical_info<C>(self);
        assert!(
            (
                token_chain == expected_chain &&
                token_address == expected_address
            ),
            E_ASSET_META_MISMATCH
        );

        // Finally only update the name and symbol.
        self.metadata.symbol = symbol;
        self.metadata.name = name;
    }

    #[test_only]
    public fun update_metadata_test_only<C>(
        self: &mut WrappedAsset<C>,
        token_meta: AssetMeta
    ) {
        update_metadata(self, token_meta)
    }

    /// Retrieve immutable reference to `ForeignMetadata`.
    public fun metadata<C>(self: &WrappedAsset<C>): &ForeignMetadata<C> {
        &self.metadata
    }

    /// Retrieve canonical token chain ID from `ForeignMetadata`.
    public fun token_chain<C>(meta: &ForeignMetadata<C>): u16 {
        meta.token_chain
    }

    /// Retrieve canonical token address from `ForeignMetadata`.
    public fun token_address<C>(meta: &ForeignMetadata<C>): ExternalAddress {
        meta.token_address
    }

    /// Retrieve decimal amount from `ForeignMetadata`.
    ///
    /// NOTE: This is for informational purposes. This decimal amount is not
    /// used for any calculations.
    public fun native_decimals<C>(meta: &ForeignMetadata<C>): u8 {
        meta.native_decimals
    }

    /// Retrieve asset's symbol (UTF-8) from `ForeignMetadata`.
    ///
    /// NOTE: This value can be updated.
    public fun symbol<C>(meta: &ForeignMetadata<C>): String {
        meta.symbol
    }

    /// Retrieve asset's name (UTF-8) from `ForeignMetadata`.
    ///
    /// NOTE: This value can be updated.
    public fun name<C>(meta: &ForeignMetadata<C>): String {
        meta.name
    }

    /// Retrieve total minted supply.
    public fun total_supply<C>(self: &WrappedAsset<C>): u64 {
        balance::supply_value(&self.total_supply)
    }

    /// Retrieve decimals for this wrapped asset. For any asset whose native
    /// decimals is greater than the cap (8), this will be 8.
    ///
    /// See `normalized_amount` module for more info.
    public fun decimals<C>(self: &WrappedAsset<C>): u8 {
        self.decimals
    }

    /// Retrieve canonical token chain ID and token address.
    public fun canonical_info<C>(
        self: &WrappedAsset<C>
    ): (u16, ExternalAddress) {
        (self.metadata.token_chain, self.metadata.token_address)
    }

    /// Burn a given `Balance`. `Balance` originates from an outbound token
    /// transfer for a wrapped asset.
    ///
    /// See `transfer_tokens` module for more info.
    public(friend) fun burn<C>(
        self: &mut WrappedAsset<C>,
        burned: Balance<C>
    ): u64 {
        balance::decrease_supply(&mut self.total_supply, burned)
    }

    #[test_only]
    public fun burn_test_only<C>(
        self: &mut WrappedAsset<C>,
        burned: Balance<C>
    ): u64 {
        burn(self, burned)
    }

    /// Mint a given amount. This amount is determined by an inbound token
    /// transfer payload for a wrapped asset.
    ///
    /// See `complete_transfer` module for more info.
    public(friend) fun mint<C>(
        self: &mut WrappedAsset<C>,
        amount: u64
    ): Balance<C> {
        balance::increase_supply(&mut self.total_supply, amount)
    }

    #[test_only]
    public fun mint_test_only<C>(
        self: &mut WrappedAsset<C>,
        amount: u64
    ): Balance<C> {
        mint(self, amount)
    }

    #[test_only]
    public fun destroy<C>(asset: WrappedAsset<C>) {
        let WrappedAsset {
            metadata,
            total_supply,
            decimals: _,
            upgrade_cap
        } = asset;
        sui::test_utils::destroy(total_supply);

        let ForeignMetadata {
            id,
            token_chain: _,
            token_address: _,
            native_decimals: _,
            symbol: _,
            name: _
        } = metadata;
        sui::object::delete(id);

        sui::package::make_immutable(upgrade_cap);
    }
}

#[test_only]
module token_bridge::wrapped_asset_tests {
    use std::string::{Self};
    use sui::balance::{Self};
    use sui::object::{Self};
    use sui::package::{Self};
    use sui::test_scenario::{Self};
    use wormhole::external_address::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::asset_meta::{Self};
    use token_bridge::coin_native_10::{Self};
    use token_bridge::coin_wrapped_12::{Self};
    use token_bridge::coin_wrapped_7::{Self};
    use token_bridge::token_bridge_scenario::{person};
    use token_bridge::wrapped_asset::{Self};

    #[test]
    public fun test_wrapped_asset_7() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let parsed_meta = coin_wrapped_7::token_meta();
        let expected_token_chain = asset_meta::token_chain(&parsed_meta);
        let expected_token_address = asset_meta::token_address(&parsed_meta);
        let expected_native_decimals =
            asset_meta::native_decimals(&parsed_meta);
        let expected_symbol = asset_meta::symbol(&parsed_meta);
        let expected_name = asset_meta::name(&parsed_meta);

        // Publish coin.
        let supply = coin_wrapped_7::init_and_take_supply(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@token_bridge),
                test_scenario::ctx(scenario)
            );

        // Make new.
        let asset =
            wrapped_asset::new_test_only(
                parsed_meta,
                supply,
                upgrade_cap,
                test_scenario::ctx(scenario)
            );

        // Verify members.
        let metadata = wrapped_asset::metadata(&asset);
        assert!(
            wrapped_asset::token_chain(metadata) == expected_token_chain,
            0
        );
        assert!(
            wrapped_asset::token_address(metadata) == expected_token_address,
            0
        );
        assert!(
            wrapped_asset::native_decimals(metadata) == expected_native_decimals,
            0
        );
        assert!(wrapped_asset::symbol(metadata) == expected_symbol, 0);
        assert!(wrapped_asset::name(metadata) == expected_name, 0);
        assert!(wrapped_asset::total_supply(&asset) == 0, 0);

        let (token_chain, token_address) =
            wrapped_asset::canonical_info(&asset);
        assert!(token_chain == expected_token_chain, 0);
        assert!(token_address == expected_token_address, 0);

        // Decimals are read from `CoinMetadata`, but in this case will agree
        // with the value encoded in the VAA.
        assert!(wrapped_asset::decimals(&asset) == 7, 0);
        assert!(wrapped_asset::decimals(&asset) == expected_native_decimals, 0);

        // Change name and symbol for update.
        let new_symbol = wrapped_asset::symbol(metadata);
        string::append(&mut new_symbol, string::utf8(b"??? and profit"));
        assert!(new_symbol != expected_symbol, 0);

        let new_name = wrapped_asset::name(metadata);
        string::append(&mut new_name, string::utf8(b"??? and profit"));
        assert!(new_name != expected_name, 0);

        let updated_meta =
            asset_meta::new(
                expected_token_address,
                expected_token_chain,
                expected_native_decimals,
                new_symbol,
                new_name
            );

        // Update metadata now.
        wrapped_asset::update_metadata_test_only(&mut asset, updated_meta);

        let metadata = wrapped_asset::metadata(&asset);
        assert!(wrapped_asset::symbol(metadata) == new_symbol, 0);
        assert!(wrapped_asset::name(metadata) == new_name, 0);

        // Try to mint.
        let mint_amount = 420;
        let collected = balance::zero();
        let (i, n) = (0, 8);
        while (i < n) {
            let minted =
                wrapped_asset::mint_test_only(&mut asset, mint_amount);
            assert!(balance::value(&minted) == mint_amount, 0);
            balance::join(&mut collected, minted);
            i = i + 1;
        };
        assert!(balance::value(&collected) == n * mint_amount, 0);
        assert!(
            wrapped_asset::total_supply(&asset) == balance::value(&collected),
            0
        );

        // Now try to burn.
        let burn_amount = 69;
        let i = 0;
        while (i < n) {
            let burned = balance::split(&mut collected, burn_amount);
            let check_amount =
                wrapped_asset::burn_test_only(&mut asset, burned);
            assert!(check_amount == burn_amount, 0);
            i = i + 1;
        };
        let remaining = n * mint_amount - n * burn_amount;
        assert!(wrapped_asset::total_supply(&asset) == remaining, 0);
        assert!(balance::value(&collected) == remaining, 0);

        // Clean up.
        balance::destroy_for_testing(collected);
        wrapped_asset::destroy(asset);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    public fun test_wrapped_asset_12() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let parsed_meta = coin_wrapped_12::token_meta();
        let expected_token_chain = asset_meta::token_chain(&parsed_meta);
        let expected_token_address = asset_meta::token_address(&parsed_meta);
        let expected_native_decimals =
            asset_meta::native_decimals(&parsed_meta);
        let expected_symbol = asset_meta::symbol(&parsed_meta);
        let expected_name = asset_meta::name(&parsed_meta);

        // Publish coin.
        let supply = coin_wrapped_12::init_and_take_supply(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@token_bridge),
                test_scenario::ctx(scenario)
            );

        // Make new.
        let asset =
            wrapped_asset::new_test_only(
                parsed_meta,
                supply,
                upgrade_cap,
                test_scenario::ctx(scenario)
            );

        // Verify members.
        let metadata = wrapped_asset::metadata(&asset);
        assert!(
            wrapped_asset::token_chain(metadata) == expected_token_chain,
            0
        );
        assert!(
            wrapped_asset::token_address(metadata) == expected_token_address,
            0
        );
        assert!(
            wrapped_asset::native_decimals(metadata) == expected_native_decimals,
            0
        );
        assert!(wrapped_asset::symbol(metadata) == expected_symbol, 0);
        assert!(wrapped_asset::name(metadata) == expected_name, 0);
        assert!(wrapped_asset::total_supply(&asset) == 0, 0);

        let (token_chain, token_address) =
            wrapped_asset::canonical_info(&asset);
        assert!(token_chain == expected_token_chain, 0);
        assert!(token_address == expected_token_address, 0);

        // Decimals are read from `CoinMetadata`, but in this case will agree
        // with the value encoded in the VAA.
        assert!(wrapped_asset::decimals(&asset) == 8, 0);
        assert!(wrapped_asset::decimals(&asset) != expected_native_decimals, 0);

        // Change name and symbol for update.
        let new_symbol = wrapped_asset::symbol(metadata);
        string::append(&mut new_symbol, string::utf8(b"??? and profit"));
        assert!(new_symbol != expected_symbol, 0);

        let new_name = wrapped_asset::name(metadata);
        string::append(&mut new_name, string::utf8(b"??? and profit"));
        assert!(new_name != expected_name, 0);

        let updated_meta =
            asset_meta::new(
                expected_token_address,
                expected_token_chain,
                expected_native_decimals,
                new_symbol,
                new_name
            );

        // Update metadata now.
        wrapped_asset::update_metadata_test_only(&mut asset, updated_meta);

        let metadata = wrapped_asset::metadata(&asset);
        assert!(wrapped_asset::symbol(metadata) == new_symbol, 0);
        assert!(wrapped_asset::name(metadata) == new_name, 0);

        // Try to mint.
        let mint_amount = 420;
        let collected = balance::zero();
        let (i, n) = (0, 8);
        while (i < n) {
            let minted =
                wrapped_asset::mint_test_only(&mut asset, mint_amount);
            assert!(balance::value(&minted) == mint_amount, 0);
            balance::join(&mut collected, minted);
            i = i + 1;
        };
        assert!(balance::value(&collected) == n * mint_amount, 0);
        assert!(
            wrapped_asset::total_supply(&asset) == balance::value(&collected),
            0
        );

        // Now try to burn.
        let burn_amount = 69;
        let i = 0;
        while (i < n) {
            let burned = balance::split(&mut collected, burn_amount);
            let check_amount =
                wrapped_asset::burn_test_only(&mut asset, burned);
            assert!(check_amount == burn_amount, 0);
            i = i + 1;
        };
        let remaining = n * mint_amount - n * burn_amount;
        assert!(wrapped_asset::total_supply(&asset) == remaining, 0);
        assert!(balance::value(&collected) == remaining, 0);

        // Clean up.
        balance::destroy_for_testing(collected);
        wrapped_asset::destroy(asset);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wrapped_asset::E_SUI_CHAIN)]
    // In this negative test case, we attempt to register a native coin as a
    // wrapped coin.
    fun test_cannot_new_sui_chain() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize new coin type.
        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Sui's chain ID is not allowed.
        let invalid_meta =
            asset_meta::new(
                external_address::default(),
                chain_id(),
                10,
                string::utf8(b""),
                string::utf8(b"")
            );

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@token_bridge),
                test_scenario::ctx(scenario)
            );

        // You shall not pass!
        let asset =
            wrapped_asset::new_test_only(
                invalid_meta,
                coin_native_10::create_supply(),
                upgrade_cap,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        wrapped_asset::destroy(asset);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wrapped_asset::E_ASSET_META_MISMATCH)]
    /// In this negative test case, we attempt to update with a mismatching
    /// chain.
    fun test_cannot_update_metadata_asset_meta_mismatch_token_address() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let parsed_meta = coin_wrapped_12::token_meta();
        let expected_token_chain = asset_meta::token_chain(&parsed_meta);
        let expected_token_address = asset_meta::token_address(&parsed_meta);
        let expected_native_decimals =
            asset_meta::native_decimals(&parsed_meta);

        // Publish coin.
        let supply = coin_wrapped_12::init_and_take_supply(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@token_bridge),
                test_scenario::ctx(scenario)
            );

        // Make new.
        let asset =
            wrapped_asset::new_test_only(
                parsed_meta,
                supply,
                upgrade_cap,
                test_scenario::ctx(scenario)
            );

        let invalid_meta =
            asset_meta::new(
                external_address::default(),
                expected_token_chain,
                expected_native_decimals,
                string::utf8(b""),
                string::utf8(b""),
            );
        assert!(
            asset_meta::token_address(&invalid_meta) != expected_token_address,
            0
        );
        assert!(
            asset_meta::token_chain(&invalid_meta) == expected_token_chain,
            0
        );
        assert!(
            asset_meta::native_decimals(&invalid_meta) == expected_native_decimals,
            0
        );

        // You shall not pass!
        wrapped_asset::update_metadata_test_only(&mut asset, invalid_meta);

        // Clean up.
        wrapped_asset::destroy(asset);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wrapped_asset::E_ASSET_META_MISMATCH)]
    /// In this negative test case, we attempt to update with a mismatching
    /// chain.
    fun test_cannot_update_metadata_asset_meta_mismatch_token_chain() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let parsed_meta = coin_wrapped_12::token_meta();
        let expected_token_chain = asset_meta::token_chain(&parsed_meta);
        let expected_token_address = asset_meta::token_address(&parsed_meta);
        let expected_native_decimals =
            asset_meta::native_decimals(&parsed_meta);

        // Publish coin.
        let supply = coin_wrapped_12::init_and_take_supply(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@token_bridge),
                test_scenario::ctx(scenario)
            );

        // Make new.
        let asset =
            wrapped_asset::new_test_only(
                parsed_meta,
                supply,
                upgrade_cap,
                test_scenario::ctx(scenario)
            );

        let invalid_meta =
            asset_meta::new(
                expected_token_address,
                chain_id(),
                expected_native_decimals,
                string::utf8(b""),
                string::utf8(b""),
            );
        assert!(
            asset_meta::token_address(&invalid_meta) == expected_token_address,
            0
        );
        assert!(
            asset_meta::token_chain(&invalid_meta) != expected_token_chain,
            0
        );
        assert!(
            asset_meta::native_decimals(&invalid_meta) == expected_native_decimals,
            0
        );

        // You shall not pass!
        wrapped_asset::update_metadata_test_only(&mut asset, invalid_meta);

        // Clean up.
        wrapped_asset::destroy(asset);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = wormhole::package_utils::E_INVALID_UPGRADE_CAP
    )]
    fun test_cannot_new_upgrade_cap_mismatch() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Publish coin.
        let supply = coin_wrapped_12::init_and_take_supply(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@0xbadc0de),
                test_scenario::ctx(scenario)
            );

        // You shall not pass!
        let asset =
            wrapped_asset::new_test_only(
                coin_wrapped_12::token_meta(),
                supply,
                upgrade_cap,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        wrapped_asset::destroy(asset);

        // Done.
        test_scenario::end(my_scenario);
    }
}
