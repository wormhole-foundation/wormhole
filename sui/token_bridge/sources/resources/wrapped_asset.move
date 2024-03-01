// SPDX-License-Identifier: Apache 2

/// This module implements two custom types relating to Token Bridge wrapped
/// assets. These assets have been attested from foreign networks, whose
/// metadata is stored in `ForeignInfo`. The Token Bridge contract is the
/// only authority that can mint and burn these assets via `Supply`.
///
/// See `create_wrapped` and 'token_registry' modules for more details.
module token_bridge::wrapped_asset {
    use std::string::{String};
    use sui::balance::{Self, Balance};
    use sui::coin::{Self, TreasuryCap, CoinMetadata};
    use sui::package::{Self, UpgradeCap};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{chain_id};

    use token_bridge::string_utils;
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
    /// Coin decimals don't match the VAA.
    const E_DECIMALS_MISMATCH: u64 = 2;

    /// Container storing foreign asset info.
    struct ForeignInfo<phantom C> has store {
        token_chain: u16,
        token_address: ExternalAddress,
        native_decimals: u8,
        symbol: String
    }

    /// Container managing `ForeignInfo` and `TreasuryCap` for a wrapped asset
    /// coin type.
    struct WrappedAsset<phantom C> has store {
        info: ForeignInfo<C>,
        treasury_cap: TreasuryCap<C>,
        decimals: u8,
        upgrade_cap: UpgradeCap
    }

    /// Create new `WrappedAsset`.
    ///
    /// See `token_registry` module for more info.
    public(friend) fun new<C>(
        token_meta: AssetMeta,
        coin_meta: &mut CoinMetadata<C>,
        treasury_cap: TreasuryCap<C>,
        upgrade_cap: UpgradeCap
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

        // Set metadata.
        coin::update_name(&treasury_cap, coin_meta, name);
        coin::update_symbol(&treasury_cap, coin_meta, string_utils::to_ascii(&symbol));

        let decimals = cap_decimals(native_decimals);

        // Ensure that the `C` type has the right number of decimals. This is
        // the only field in the coinmeta that cannot be changed after the fact,
        // so we expect to receive one that already has the correct decimals
        // set.
        assert!(decimals == coin::get_decimals(coin_meta), E_DECIMALS_MISMATCH);

        let info =
            ForeignInfo {
                token_address,
                token_chain,
                native_decimals,
                symbol
            };

        WrappedAsset {
            info,
            treasury_cap,
            decimals,
            upgrade_cap
        }
    }

    #[test_only]
    public fun new_test_only<C>(
        token_meta: AssetMeta,
        coin_meta: &mut CoinMetadata<C>,
        treasury_cap: TreasuryCap<C>,
        upgrade_cap: UpgradeCap
    ): WrappedAsset<C> {
        new(token_meta, coin_meta, treasury_cap, upgrade_cap)
    }

    /// Update existing `ForeignInfo` using new `AssetMeta`.
    ///
    /// See `token_registry` module for more info.
    public(friend) fun update_metadata<C>(
        self: &mut WrappedAsset<C>,
        coin_meta: &mut CoinMetadata<C>,
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
        self.info.symbol = symbol;
        coin::update_name(&self.treasury_cap, coin_meta, name);
        coin::update_symbol(&self.treasury_cap, coin_meta, string_utils::to_ascii(&symbol));
    }

    #[test_only]
    public fun update_metadata_test_only<C>(
        self: &mut WrappedAsset<C>,
        coin_meta: &mut CoinMetadata<C>,
        token_meta: AssetMeta
    ) {
        update_metadata(self, coin_meta, token_meta)
    }

    /// Retrieve immutable reference to `ForeignInfo`.
    public fun info<C>(self: &WrappedAsset<C>): &ForeignInfo<C> {
        &self.info
    }

    /// Retrieve canonical token chain ID from `ForeignInfo`.
    public fun token_chain<C>(info: &ForeignInfo<C>): u16 {
        info.token_chain
    }

    /// Retrieve canonical token address from `ForeignInfo`.
    public fun token_address<C>(info: &ForeignInfo<C>): ExternalAddress {
        info.token_address
    }

    /// Retrieve decimal amount from `ForeignInfo`.
    ///
    /// NOTE: This is for informational purposes. This decimal amount is not
    /// used for any calculations.
    public fun native_decimals<C>(info: &ForeignInfo<C>): u8 {
        info.native_decimals
    }

    /// Retrieve asset's symbol (UTF-8) from `ForeignMetadata`.
    ///
    /// NOTE: This value can be updated.
    public fun symbol<C>(info: &ForeignInfo<C>): String {
        info.symbol
    }

    /// Retrieve total minted supply.
    public fun total_supply<C>(self: &WrappedAsset<C>): u64 {
        coin::total_supply(&self.treasury_cap)
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
        (self.info.token_chain, self.info.token_address)
    }

    /// Burn a given `Balance`. `Balance` originates from an outbound token
    /// transfer for a wrapped asset.
    ///
    /// See `transfer_tokens` module for more info.
    public(friend) fun burn<C>(
        self: &mut WrappedAsset<C>,
        burned: Balance<C>
    ): u64 {
        balance::decrease_supply(coin::supply_mut(&mut self.treasury_cap), burned)
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
        coin::mint_balance(&mut self.treasury_cap, amount)
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
            info,
            treasury_cap,
            decimals: _,
            upgrade_cap
        } = asset;
        sui::test_utils::destroy(treasury_cap);

        let ForeignInfo {
            token_chain: _,
            token_address: _,
            native_decimals: _,
            symbol: _
        } = info;

        sui::package::make_immutable(upgrade_cap);
    }
}

#[test_only]
module token_bridge::wrapped_asset_tests {
    use std::string::{Self};
    use sui::balance::{Self};
    use sui::coin::{Self, CoinMetadata};
    use sui::object::{Self};
    use sui::package::{Self};
    use sui::test_scenario::{Self};
    use wormhole::external_address::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::asset_meta::{Self};
    use token_bridge::string_utils;
    use token_bridge::coin_native_10::{COIN_NATIVE_10, Self};
    use token_bridge::coin_wrapped_12::{COIN_WRAPPED_12, Self};
    use token_bridge::coin_wrapped_7::{COIN_WRAPPED_7, Self};
    use token_bridge::token_bridge_scenario::{person};
    use token_bridge::wrapped_asset::{Self};

    #[test]
    fun test_wrapped_asset_7() {
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
        let treasury_cap =
            coin_wrapped_7::init_and_take_treasury_cap(
                scenario,
                caller
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@token_bridge),
                test_scenario::ctx(scenario)
            );

        let coin_meta: CoinMetadata<COIN_WRAPPED_7> = test_scenario::take_shared(scenario);

        // Make new.
        let asset =
            wrapped_asset::new_test_only(
                parsed_meta,
                &mut coin_meta,
                treasury_cap,
                upgrade_cap
            );

        // Verify members.
        let info = wrapped_asset::info(&asset);
        assert!(
            wrapped_asset::token_chain(info) == expected_token_chain,
            0
        );
        assert!(
            wrapped_asset::token_address(info) == expected_token_address,
            0
        );
        assert!(
            wrapped_asset::native_decimals(info) == expected_native_decimals,
            0
        );
        assert!(coin::get_symbol(&coin_meta) == string_utils::to_ascii(&expected_symbol), 0);
        assert!(coin::get_name(&coin_meta) == expected_name, 0);
        assert!(wrapped_asset::total_supply(&asset) == 0, 0);

        let (token_chain, token_address) =
            wrapped_asset::canonical_info(&asset);
        assert!(token_chain == expected_token_chain, 0);
        assert!(token_address == expected_token_address, 0);

        // Decimals are read from `CoinMetadata`, but in this case will agree
        // with the value encoded in the VAA.
        assert!(wrapped_asset::decimals(&asset) == expected_native_decimals, 0);
        assert!(coin::get_decimals(&coin_meta) == expected_native_decimals, 0);

        // Change name and symbol for update.
        let new_symbol = std::ascii::into_bytes(coin::get_symbol(&coin_meta));

        std::vector::append(&mut new_symbol, b"??? and profit");
        assert!(new_symbol != *string::bytes(&expected_symbol), 0);

        let new_name = coin::get_name(&coin_meta);
        string::append(&mut new_name, string::utf8(b"??? and profit"));
        assert!(new_name != expected_name, 0);

        let updated_meta =
            asset_meta::new(
                expected_token_address,
                expected_token_chain,
                expected_native_decimals,
                string::utf8(new_symbol),
                new_name
            );

        // Update metadata now.
        wrapped_asset::update_metadata_test_only(&mut asset, &mut coin_meta, updated_meta);

        assert!(coin::get_symbol(&coin_meta) == std::ascii::string(new_symbol), 0);
        assert!(coin::get_name(&coin_meta) == new_name, 0);

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

        test_scenario::return_shared(coin_meta);

        // Clean up.
        balance::destroy_for_testing(collected);
        wrapped_asset::destroy(asset);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_wrapped_asset_12() {
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
        let treasury_cap =
            coin_wrapped_12::init_and_take_treasury_cap(
                scenario,
                caller
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@token_bridge),
                test_scenario::ctx(scenario)
            );

        let coin_meta: CoinMetadata<COIN_WRAPPED_12> = test_scenario::take_shared(scenario);

        // Make new.
        let asset =
            wrapped_asset::new_test_only(
                parsed_meta,
                &mut coin_meta,
                treasury_cap,
                upgrade_cap
            );

        // Verify members.
        let info = wrapped_asset::info(&asset);
        assert!(
            wrapped_asset::token_chain(info) == expected_token_chain,
            0
        );
        assert!(
            wrapped_asset::token_address(info) == expected_token_address,
            0
        );
        assert!(
            wrapped_asset::native_decimals(info) == expected_native_decimals,
            0
        );
        assert!(coin::get_symbol(&coin_meta) == string_utils::to_ascii(&expected_symbol), 0);
        assert!(coin::get_name(&coin_meta) == expected_name, 0);
        assert!(wrapped_asset::total_supply(&asset) == 0, 0);

        let (token_chain, token_address) =
            wrapped_asset::canonical_info(&asset);
        assert!(token_chain == expected_token_chain, 0);
        assert!(token_address == expected_token_address, 0);

        // Decimals are read from `CoinMetadata`, but in this case will not
        // agree with the value encoded in the VAA.
        assert!(wrapped_asset::decimals(&asset) == 8, 0);
        assert!(
            coin::get_decimals(&coin_meta) == wrapped_asset::decimals(&asset),
            0
        );
        assert!(wrapped_asset::decimals(&asset) != expected_native_decimals, 0);

        // Change name and symbol for update.
        let new_symbol = std::ascii::into_bytes(coin::get_symbol(&coin_meta));

        std::vector::append(&mut new_symbol, b"??? and profit");
        assert!(new_symbol != *string::bytes(&expected_symbol), 0);

        let new_name = coin::get_name(&coin_meta);
        string::append(&mut new_name, string::utf8(b"??? and profit"));
        assert!(new_name != expected_name, 0);

        let updated_meta =
            asset_meta::new(
                expected_token_address,
                expected_token_chain,
                expected_native_decimals,
                string::utf8(new_symbol),
                new_name
            );

        // Update metadata now.
        wrapped_asset::update_metadata_test_only(&mut asset, &mut coin_meta, updated_meta);

        assert!(coin::get_symbol(&coin_meta) == std::ascii::string(new_symbol), 0);
        assert!(coin::get_name(&coin_meta) == new_name, 0);

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
        test_scenario::return_shared(coin_meta);

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

        let treasury_cap = test_scenario::take_shared<coin::TreasuryCap<COIN_NATIVE_10>>(scenario);
        let coin_meta = test_scenario::take_shared<CoinMetadata<COIN_NATIVE_10>>(scenario);

        // You shall not pass!
        let asset =
            wrapped_asset::new_test_only(
                invalid_meta,
                &mut coin_meta,
                treasury_cap,
                upgrade_cap
            );

        // Clean up.
        wrapped_asset::destroy(asset);

        abort 42
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
        let treasury_cap =
            coin_wrapped_12::init_and_take_treasury_cap(
                scenario,
                caller
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@token_bridge),
                test_scenario::ctx(scenario)
            );

        let coin_meta = test_scenario::take_shared(scenario);

        // Make new.
        let asset =
            wrapped_asset::new_test_only(
                parsed_meta,
                &mut coin_meta,
                treasury_cap,
                upgrade_cap
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
        wrapped_asset::update_metadata_test_only(&mut asset, &mut coin_meta, invalid_meta);

        // Clean up.
        wrapped_asset::destroy(asset);

        abort 42
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
        let treasury_cap =
            coin_wrapped_12::init_and_take_treasury_cap(
                scenario,
                caller
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@token_bridge),
                test_scenario::ctx(scenario)
            );

        let coin_meta = test_scenario::take_shared(scenario);

        // Make new.
        let asset =
            wrapped_asset::new_test_only(
                parsed_meta,
                &mut coin_meta,
                treasury_cap,
                upgrade_cap
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
        wrapped_asset::update_metadata_test_only(&mut asset, &mut coin_meta, invalid_meta);

        // Clean up.
        wrapped_asset::destroy(asset);

        abort 42
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
        let treasury_cap =
            coin_wrapped_12::init_and_take_treasury_cap(
                scenario,
                caller
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Upgrade cap belonging to coin type.
        let upgrade_cap =
            package::test_publish(
                object::id_from_address(@0xbadc0de),
                test_scenario::ctx(scenario)
            );

        let coin_meta = test_scenario::take_shared(scenario);

        // You shall not pass!
        let asset =
            wrapped_asset::new_test_only(
                coin_wrapped_12::token_meta(),
                &mut coin_meta,
                treasury_cap,
                upgrade_cap
            );

        // Clean up.
        wrapped_asset::destroy(asset);

        abort 42
    }
}
