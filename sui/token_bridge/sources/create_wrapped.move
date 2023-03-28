/// This module uses the one-time witness (OTW).
/// Sui OTW eference: https://examples.sui.io/basics/one-time-witness.html
module token_bridge::create_wrapped {
    use std::option::{Self};
    use sui::coin::{Self, CoinMetadata};
    use sui::transfer::{Self};
    use sui::tx_context::{TxContext};
    use sui::url::{Url};
    use wormhole::state::{State as WormholeState};
    use wormhole::vaa::{Self as core_vaa};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::wrapped_coin::{Self, WrappedCoin};
    use token_bridge::state::{Self, State};
    use token_bridge::vaa::{Self};
    use token_bridge::token_info::{Self};

    const E_UNREGISTERED_WRAPPED_ASSET: u64 = 0;

    /// The amounts in the token bridge payload are truncated to 8 decimals
    /// in each of the contracts when sending tokens out, so there's no
    /// precision beyond 10^-8. We could preserve the original number of
    /// decimals when creating wrapped assets, and "untruncate" the amounts
    /// on the way out by scaling back appropriately. This is what most
    /// other chains do, but untruncating from 8 decimals to 18 decimals
    /// loses log2(10^10) ~ 33 bits of precision, which we cannot afford on
    /// Aptos (and Solana), as the coin type only has 64bits to begin with.
    /// Contrast with Ethereum, where amounts are 256 bits.
    /// So we cap the maximum decimals at 8 when creating a wrapped token.
    const MAX_WRAPPED_DECIMALS: u8 = 8;

    /// This function will be called from the `init` function of a module that
    /// defines a OTW type. Due to the nature of `init` functions, this function
    /// must be stateless.
    /// This means that it performs no verification of the VAA beyond parsing
    /// it. It is the responsbility of `register_new_coin` to perform the
    /// validation.
    /// This function guarantees that if the VAA is valid, then a new currency
    /// `CoinType` will be created such that:
    /// 1) the asset metadata matches the VAA
    /// 2) the treasury total supply will be 0
    ///
    /// Thanks to the above properties, `register_new_coin` does not need to
    /// do any checks other than the VAA in `WrappedCoin` is valid.
    public fun create_unregistered_currency<CoinType: drop>(
        vaa_buf: vector<u8>,
        coin_witness: CoinType,
        ctx: &mut TxContext
    ): WrappedCoin<CoinType> {
        let payload = core_vaa::peel_payload_from_vaa(&vaa_buf);
        let meta = asset_meta::deserialize(payload);

        let coin_decimals = (
            sui::math::min(
                (MAX_WRAPPED_DECIMALS as u64),
                (asset_meta::native_decimals(&meta) as u64)
            ) as u8
        );

        let (treasury_cap, coin_metadata) =
            coin::create_currency<CoinType>(
                coin_witness,
                coin_decimals,
                b"UNREGISTERED",
                b"Pending Token Bridge Registration",
                b"UNREGISTERED",
                option::none<Url>(), // No url necessary.
                ctx
            );

        transfer::public_share_object(coin_metadata);

        wrapped_coin::new(
            vaa_buf,
            treasury_cap,
            coin_decimals,
            ctx
        )
    }

    /// After executing `create_unregistered_currency`, user needs to complete
    /// the registration process by calling this method.
    ///
    /// This method destroys `WrappedCoin<CoinType>`, which warehouses the asset
    /// meta VAA and `TreasuryCap`, which are used to update the symbol and name
    /// to what was encoded in the asset meta VAA payload.
    public entry fun register_new_coin<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        new_wrapped_coin: WrappedCoin<CoinType>,
        coin_metadata: &mut CoinMetadata<CoinType>,
        ctx: &mut TxContext,
    ) {
        let (vaa_buf, treasury_cap, decimals) =
            wrapped_coin::destroy(new_wrapped_coin);

        // Deserialize to `AssetMeta`.
        let meta =
            parse_and_verify_asset_meta(
                token_bridge_state,
                worm_state,
                vaa_buf,
                ctx
            );

        // `register_wrapped_asset` uses `registered_tokens::add_new_wrapped`,
        // which will check whether the asset has already been registered and if
        // the token chain ID is not Sui's.
        //
        // If both of these conditions are met, `register_wrapped_asset` will
        // succeed and the new wrapped coin will be registered.
        state::register_wrapped_asset<CoinType>(
            token_bridge_state,
            asset_meta::token_chain(&meta),
            asset_meta::token_address(&meta),
            treasury_cap,
            decimals,
        );

        // Proceed to update coin's metadata.
        handle_update_metadata<CoinType>(
            token_bridge_state,
            &meta,
            coin_metadata
        );
    }

    /// For existing wrapped assets, we can update the existing `CoinMetadata`
    /// for a given `CoinType` (one that belongs to Token Bridge's wrapped
    /// registry) with a new asset meta VAA emitted from a foreign network.
    public entry fun update_registered_metadata<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        vaa_buf: vector<u8>,
        coin_metadata: &mut CoinMetadata<CoinType>,
        ctx: &mut TxContext
    ) {
        // Deserialize to `AssetMeta`.
        let meta =
            parse_and_verify_asset_meta(
                token_bridge_state,
                worm_state,
                vaa_buf,
                ctx
            );

        // Verify that the token info agrees with the info encoded in this
        // transfer. Checking whether this asset is wrapped may be superfluous,
        // but we want to ensure that this VAA was not generated from a native
        // Sui coin.
        let info = state::token_info<CoinType>(token_bridge_state);
        assert!(
            (
                token_info::is_wrapped(&info) &&
                token_info::equals(
                    &info,
                    asset_meta::token_chain(&meta),
                    asset_meta::token_address(&meta)
                )
            ),
            E_UNREGISTERED_WRAPPED_ASSET
        );

        // Proceed to update coin's metadata.
        handle_update_metadata<CoinType>(
            token_bridge_state,
            &meta,
            coin_metadata
        );
    }

    fun parse_and_verify_asset_meta(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        vaa_buf: vector<u8>,
        ctx: &mut TxContext
    ): AssetMeta {
        let parsed = vaa::parse_verify_and_replay_protect(
            token_bridge_state,
            worm_state,
            vaa_buf,
            ctx
        );

        asset_meta::deserialize(core_vaa::take_payload(parsed))
    }

    fun handle_update_metadata<CoinType>(
        token_bridge_state: &State,
        meta: &AssetMeta,
        coin_metadata: &mut CoinMetadata<CoinType>,
    ) {
        // We need `TreasuryCap` to grant us access to update the symbol and
        // name for a given `CoinType`.
        let treasury_cap = state::treasury_cap<CoinType>(token_bridge_state);
        coin::update_symbol(
            treasury_cap,
            coin_metadata,
            asset_meta::symbol_to_ascii(meta)
        );
        coin::update_name(
            treasury_cap,
            coin_metadata,
            asset_meta::name_to_utf8(meta)
        );
        // We are using the description of `CoinMetadata` as a convenient spot
        // to preserve a UTF-8 symbol, if it has any characters that are not in
        // the ASCII character set.
        coin::update_description(
            treasury_cap,
            coin_metadata,
            asset_meta::symbol_to_utf8(meta)
        );
    }
}
