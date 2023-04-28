// SPDX-License-Identifier: Apache 2

/// This module implements a custom type that keeps track of info relating to
/// assets (coin types) native to Sui. Token Bridge takes custody of these
/// assets when someone invokes a token transfer outbound. Likewise, Token
/// Bridge releases some of its balance from its custody of when someone redeems
/// an inbound token transfer intended for Sui.
///
/// See `token_registry` module for more details.
module token_bridge::native_asset {
    use sui::balance::{Self, Balance};
    use sui::coin::{Self, CoinMetadata};
    use sui::object::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::state::{chain_id};

    friend token_bridge::complete_transfer;
    friend token_bridge::token_registry;
    friend token_bridge::transfer_tokens;

    /// Container for storing canonical token address and custodied `Balance`.
    struct NativeAsset<phantom C> has store {
        custody: Balance<C>,
        token_address: ExternalAddress,
        decimals: u8
    }

    /// Token Bridge identifies native assets using `CoinMetadata` object `ID`.
    /// This method converts this `ID` to `ExternalAddress`.
    public fun canonical_address<C>(
        metadata: &CoinMetadata<C>
    ): ExternalAddress {
        external_address::from_id(object::id(metadata))
    }

    /// Create new `NativeAsset`.
    ///
    /// NOTE: The canonical token address is determined by the coin metadata's
    /// object ID.
    public(friend) fun new<C>(metadata: &CoinMetadata<C>): NativeAsset<C> {
        NativeAsset {
            custody: balance::zero(),
            token_address: canonical_address(metadata),
            decimals: coin::get_decimals(metadata)
        }
    }

    #[test_only]
    public fun new_test_only<C>(metadata: &CoinMetadata<C>): NativeAsset<C> {
        new(metadata)
    }

    /// Retrieve canonical token address.
    public fun token_address<C>(self: &NativeAsset<C>): ExternalAddress {
        self.token_address
    }

    /// Retrieve decimals, which originated from `CoinMetadata`.
    public fun decimals<C>(self: &NativeAsset<C>): u8 {
        self.decimals
    }

    /// Retrieve custodied `Balance` value.
    public fun custody<C>(self: &NativeAsset<C>): u64 {
        balance::value(&self.custody)
    }

    /// Retrieve canonical token chain ID (Sui's) and token address.
    public fun canonical_info<C>(
        self: &NativeAsset<C>
    ): (u16, ExternalAddress) {
        (chain_id(), self.token_address)
    }

    /// Deposit a given `Balance`. `Balance` originates from an outbound token
    /// transfer for a native asset.
    ///
    /// See `transfer_tokens` module for more info.
    public(friend) fun deposit<C>(
        self: &mut NativeAsset<C>,
        deposited: Balance<C>
    ) {
        balance::join(&mut self.custody, deposited);
    }

    #[test_only]
    public fun deposit_test_only<C>(
        self: &mut NativeAsset<C>,
        deposited: Balance<C>
    ) {
        deposit(self, deposited)
    }

    /// Withdraw a given amount from custody. This amount is determiend by an
    /// inbound token transfer payload for a native asset.
    ///
    /// See `complete_transfer` module for more info.
    public(friend) fun withdraw<C>(
        self: &mut NativeAsset<C>,
        amount: u64
    ): Balance<C> {
        balance::split(&mut self.custody, amount)
    }

    #[test_only]
    public fun withdraw_test_only<C>(
        self: &mut NativeAsset<C>,
        amount: u64
    ): Balance<C> {
        withdraw(self, amount)
    }

    #[test_only]
    public fun destroy<C>(asset: NativeAsset<C>) {
        let NativeAsset {
            custody,
            token_address: _,
            decimals: _
        } = asset;
        balance::destroy_for_testing(custody);
    }
}

#[test_only]
module token_bridge::native_asset_tests {
    use sui::balance::{Self};
    use sui::coin::{Self};
    use sui::object::{Self};
    use sui::test_scenario::{Self};
    use wormhole::external_address::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::native_asset::{Self};
    use token_bridge::token_bridge_scenario::{person};

    #[test]
    /// In this test, we exercise all the functionalities of a native asset
    /// object, including new, deposit, withdraw, to_token_info, as well as
    /// getting fields token_address, decimals, balance.
    fun test_native_asset() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Publish coin.
        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let coin_meta = coin_native_10::take_metadata(scenario);

        // Make new.
        let asset = native_asset::new_test_only(&coin_meta);

        // Assert token address and decimals are correct.
        let expected_token_address =
            external_address::from_id(object::id(&coin_meta));
        assert!(
            native_asset::token_address(&asset) == expected_token_address,
            0
        );
        assert!(
            native_asset::decimals(&asset) == coin::get_decimals(&coin_meta),
            0
        );
        assert!(native_asset::custody(&asset) == 0, 0);

        // deposit some coins into the NativeAsset coin custody
        let deposit_amount = 1000;
        let (i, n) = (0, 8);
        while (i < n) {
            native_asset::deposit_test_only(
                &mut asset,
                balance::create_for_testing(
                    deposit_amount
                )
            );
            i = i + 1;
        };
        let total_deposited = n * deposit_amount;
        assert!(native_asset::custody(&asset) == total_deposited, 0);

        let withdraw_amount = 690;
        let total_withdrawn = balance::zero();
        let i = 0;
        while (i < n) {
            let withdrawn = native_asset::withdraw_test_only(
                &mut asset,
                withdraw_amount
            );
            assert!(balance::value(&withdrawn) == withdraw_amount, 0);
            balance::join(&mut total_withdrawn, withdrawn);
            i = i + 1;
        };

        // convert to token info and assert convrsion is correct
        let (
            token_chain,
            token_address
        ) = native_asset::canonical_info<COIN_NATIVE_10>(&asset);

        assert!(token_chain == chain_id(), 0);
        assert!(token_address == expected_token_address, 0);

        // check that updated balance is correct
        let expected_remaining = total_deposited - n * withdraw_amount;
        let remaining = native_asset::custody(&asset);
        assert!(remaining == expected_remaining, 0);

        // Clean up.
        coin_native_10::return_metadata(coin_meta);
        balance::destroy_for_testing(total_withdrawn);
        native_asset::destroy(asset);

        // Done.
        test_scenario::end(my_scenario);
    }
}
