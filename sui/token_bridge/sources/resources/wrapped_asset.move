module token_bridge::wrapped_asset {
    use std::string::{String};
    use sui::balance::{Self, Balance, Supply};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{chain_id};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::normalized_amount::{cap_decimals};

    // For 'new', `burn` and `mint`
    friend token_bridge::registered_tokens;

    const E_SUI_CHAIN: u64 = 0;

    struct ForeignMetadata<phantom C> has key, store {
        id: UID,
        token_chain: u16,
        token_address: ExternalAddress,
        native_decimals: u8,
        symbol: String,
        name: String
    }

    /// WrappedAsset<C> stores all the metadata about a wrapped asset
    struct WrappedAsset<phantom C> has store {
        metadata: ForeignMetadata<C>,
        total_supply: Supply<C>,
        decimals: u8,
    }

    public fun new<C>(
        token_meta: AssetMeta,
        supply: Supply<C>,
        ctx: &mut TxContext
    ): WrappedAsset<C> {
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
            decimals: cap_decimals(native_decimals)
        }
    }

    public fun metadata<C>(self: &WrappedAsset<C>): &ForeignMetadata<C> {
        &self.metadata
    }

    public fun token_chain<C>(meta: &ForeignMetadata<C>): u16 {
        meta.token_chain
    }

    public fun token_address<C>(meta: &ForeignMetadata<C>): ExternalAddress {
        meta.token_address
    }

    public fun native_decimals<C>(meta: &ForeignMetadata<C>): u8 {
        meta.native_decimals
    }

    public fun symbol<C>(meta: &ForeignMetadata<C>): String {
        meta.symbol
    }

    public fun name<C>(meta: &ForeignMetadata<C>): String {
        meta.name
    }

    public fun total_supply<C>(self: &WrappedAsset<C>): u64 {
        balance::supply_value(&self.total_supply)
    }

    public fun decimals<C>(self: &WrappedAsset<C>): u8 {
        self.decimals
    }

    public fun canonical_info<C>(
        self: &WrappedAsset<C>
    ): (u16, ExternalAddress) {
        (self.metadata.token_chain, self.metadata.token_address)
    }

    public(friend) fun burn_balance<C>(
        self: &mut WrappedAsset<C>,
        burnable: Balance<C>
    ): u64 {
        balance::decrease_supply(&mut self.total_supply, burnable)
    }

    public(friend) fun mint_balance<C>(
        self: &mut WrappedAsset<C>,
        amount: u64
    ): Balance<C> {
        balance::increase_supply(&mut self.total_supply, amount)
    }

    #[test_only]
    public fun destroy<C>(asset: WrappedAsset<C>) {
        let WrappedAsset {
            metadata,
            total_supply,
            decimals: _
        } = asset;
        sui::balance::destroy_supply_for_testing(total_supply);

        let ForeignMetadata {
            id,
            token_chain: _,
            token_address: _,
            native_decimals: _,
            symbol: _,
            name: _
        } = metadata;
        sui::object::delete(id);
    }
}

#[test_only]
module token_bridge::wrapped_asset_test {
    use std::string::{Self};
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

        // Publish coin.
        let supply = coin_wrapped_7::init_and_take_supply(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Make new.
        let asset =
            wrapped_asset::new(
                parsed_meta,
                supply,
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
        assert!(wrapped_asset::total_supply(&asset) == 0, 0);

        let (token_chain, token_address) =
            wrapped_asset::canonical_info(&asset);
        assert!(token_chain == expected_token_chain, 0);
        assert!(token_address == expected_token_address, 0);


        // Decimals are read from `CoinMetadata`, but in this case will agree
        // with the value encoded in the VAA.
        assert!(wrapped_asset::decimals(&asset) == 7, 0);
        assert!(
            wrapped_asset::decimals(&asset) == asset_meta::native_decimals(&parsed_meta),
            0
        );

        // Clean up.
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

        // Publish coin.
        let supply = coin_wrapped_12::init_and_take_supply(scenario, caller);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Make new.
        let asset =
            wrapped_asset::new(
                parsed_meta,
                supply,
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
        assert!(wrapped_asset::total_supply(&asset) == 0, 0);

        let (token_chain, token_address) =
            wrapped_asset::canonical_info(&asset);
        assert!(token_chain == expected_token_chain, 0);
        assert!(token_address == expected_token_address, 0);

        // Decimals are read from `CoinMetadata` and will disagree with the
        // value encoded in the VAA.
        assert!(wrapped_asset::decimals(&asset) == 8, 0);
        assert!(
            wrapped_asset::decimals(&asset) != asset_meta::native_decimals(&parsed_meta),
            0
        );

        // Clean up.
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
        let invalid_asset_meta =
            asset_meta::new(
                external_address::default(),
                chain_id(),
                10,
                string::utf8(b""),
                string::utf8(b"")
            );

        // You shall not pass!
        let asset =
            wrapped_asset::new(
                invalid_asset_meta,
                coin_native_10::create_supply(),
                test_scenario::ctx(scenario)
            );

        // Clean up.
        wrapped_asset::destroy(asset);

        // Done.
        test_scenario::end(my_scenario);
    }
}
