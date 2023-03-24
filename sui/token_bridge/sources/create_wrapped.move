/// This module implements methods that create a specific coin type reflecting a
/// wrapped (foreign) asset, whose metadata is encoded in a VAA sent from
/// another network.
///
/// Wrapped assets are created in two steps.
///   1. `prepare_registration`: This method creates a new `Supply` for a given
///      coin type and wraps an encoded asset metadata VAA. We require a one-
///      time witness (OTW) because we only want one `Supply` for a given coin
///      type. This coin will be published using this method, meaning the `init`
///      method in that package will have the asset metadata VAA hard-coded
///      (which is passed into `prepare_registration` as one of its arguments).
///      A `WrappedAssetSetup` object is transferred to the transaction sender.
///   2. `complete_registration`: This method destroys the `WrappedAssetSetup`
///      object by unpacking its members. The encoded asset metadata VAA is
///      deserialized and moved (along with the `Supply`) to the state module
///      to create `ForeignMetadata`.
///
/// Wrapped asset metadata can also be updated with a new asset metadata VAA.
/// By calling `update_registration`, Token Bridge verifies that the specific
/// coin type is registered and agrees with the encoded asset metadata's
/// canonical token info. `ForeignMetadata` will be updated based on the encoded
/// asset metadata payload.
///
/// See `state` and `wrapped_asset` modules for more details.
///
/// References:
/// https://examples.sui.io/basics/one-time-witness.html
module token_bridge::create_wrapped {
    use sui::balance::{Self, Supply};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::state::{State as WormholeState};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::state::{Self, State};
    use token_bridge::vaa::{Self};

    /// Asset metadata is for native Sui coin type.
    const E_NATIVE_ASSET: u64 = 0;
    /// Asset metadata has not been registered yet.
    const E_UNREGISTERED_FOREIGN_ASSET: u64 = 1;
    /// Failed one-time witness verification.
    const E_BAD_WITNESS: u64 = 2;

    /// Container holding new coin type's `Supply` and encoded asset metadata
    /// VAA, which are required to complete this asset's registration.
    struct WrappedAssetSetup<phantom CoinType> has key, store {
        id: UID,
        vaa_buf: vector<u8>,
        supply: Supply<CoinType>
    }

    /// This method is executed within the `init` method of an untrusted module,
    /// which defines a one-time witness (OTW) type (`CoinType`). OTW is
    /// required to ensure that only one `Supply` exists for `CoinType`. This
    /// is similar to how a `TreasuryCap` is created in `coin::create_currency`.
    ///
    /// Because this method is stateless (i.e. no dependency on Token Bridge's
    /// `State` object), the contract defers VAA verification to
    /// `complete_registration` after this method has been executed.
    public fun prepare_registration<CoinType: drop>(
        witness: CoinType,
        vaa_buf: vector<u8>,
        ctx: &mut TxContext
    ): WrappedAssetSetup<CoinType> {
        // Make sure there's only one instance of the type `CoinType`. This
        // resembles the same check for `coin::create_currency`.
        assert!(sui::types::is_one_time_witness(&witness), E_BAD_WITNESS);

        // Create `WrappedAssetSetup` object and transfer to transaction sender.
        // The owner of this object will call `complete_registration` to destroy
        // it.
        WrappedAssetSetup {
            id: object::new(ctx),
            vaa_buf,
            supply: balance::create_supply(witness),
        }
    }

    /// After executing `prepare_registration`, owner of `WrappedAssetSetup`
    /// executes this method to complete this wrapped asset's registration.
    ///
    /// This method destroys `WrappedAssetSetup`, unpacking the `Supply` and
    /// encoded asset metadata VAA. The deserialized asset metadata VAA is used
    /// to create `ForeignMetadata`.
    ///
    /// TODO: Maybe add `UpgradeCap` argument (which would come from the
    /// `CoinType` package so we can either destroy it or warehouse it in
    /// `WrappedAsset`).
    public fun complete_registration<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        unregistered: WrappedAssetSetup<CoinType>,
        ctx: &mut TxContext,
    ) {
        let WrappedAssetSetup {
            id,
            vaa_buf,
            supply
        } = unregistered;
        object::delete(id);

        // Deserialize to `AssetMeta`.
        let token_meta =
            parse_and_verify_asset_meta(
                token_bridge_state,
                worm_state,
                vaa_buf,
                ctx
            );

        // `register_wrapped_asset` uses `token_registry::add_new_wrapped`,
        // which will check whether the asset has already been registered and if
        // the token chain ID is not Sui's.
        //
        // If both of these conditions are met, `register_wrapped_asset` will
        // succeed and the new wrapped coin will be registered.
        state::register_wrapped_asset(
            token_bridge_state,
            token_meta,
            supply,
            ctx
        );
    }

    /// For registered wrapped assets, we can update `ForeignMetadata` for a
    /// given `CoinType` with a new asset meta VAA emitted from another network.
    public fun update_registration<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        ctx: &TxContext
    ) {
        // Deserialize to `AssetMeta`.
        let token_meta =
            parse_and_verify_asset_meta<CoinType>(
                token_bridge_state,
                worm_state,
                vaa_buf,
                ctx
            );

        state::assert_registered_token<CoinType>(
            token_bridge_state,
            asset_meta::token_chain(&token_meta),
            asset_meta::token_address(&token_meta)
        );

        // When a wrapped asset is updated, there is a check for whether this
        // metadata originated from Sui.
        state::update_wrapped_asset(token_bridge_state, token_meta);
    }

    fun parse_and_verify_asset_meta<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        ctx: &TxContext
    ): AssetMeta<CoinType> {
        let parsed =
            vaa::parse_verify_and_consume(
                token_bridge_state,
                worm_state,
                vaa_buf,
                ctx
            );

        // Finally deserialize the VAA payload.
        asset_meta::deserialize(wormhole::vaa::take_payload(parsed))
    }

    #[test_only]
    public fun take_supply<CoinType>(
        unregistered: WrappedAssetSetup<CoinType>
    ): Supply<CoinType> {
        let WrappedAssetSetup {
            id,
            vaa_buf: _,
            supply
        } = unregistered;
        object::delete(id);

        supply
    }
}
