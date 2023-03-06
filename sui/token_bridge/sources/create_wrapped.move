/// This module uses the one-time witness (OTW).
/// Sui OTW eference: https://examples.sui.io/basics/one-time-witness.html
module token_bridge::create_wrapped {
    use sui::coin::{Self};
    use std::string::{Self};
    use std::option::{Self};
    use sui::transfer::{Self};
    use sui::tx_context::{TxContext};
    use sui::url::{Url};
    use wormhole::state::{Self as wormhole_state, State as WormholeState};
    use wormhole::myvaa as core_vaa;

    use token_bridge::asset_meta::{Self};
    use token_bridge::wrapped_coin::{Self, WrappedCoin};
    use token_bridge::state::{Self, State};
    use token_bridge::vaa::{Self};

    const E_WRAPPING_NATIVE_COIN: u64 = 0;
    const E_WRAPPING_REGISTERED_NATIVE_COIN: u64 = 1;
    const E_WRAPPED_COIN_ALREADY_INITIALIZED: u64 = 2;

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
    /// it. It is the responsbility of `register_wrapped_coin` to perform the
    /// validation.
    /// This function guarantees that if the VAA is valid, then a new currency
    /// `CoinType` will be created such that:
    /// 1) the asset metadata matches the VAA
    /// 2) the treasury total supply will be 0
    ///
    /// Thanks to the above properties, `register_wrapped_coin` does not need to
    /// do any checks other than the VAA in `WrappedCoin` is valid.
    public fun create_wrapped_coin<CoinType: drop>(
        vaa_bytes: vector<u8>,
        coin_witness: CoinType,
        ctx: &mut TxContext
    ): WrappedCoin<CoinType> {
        let payload = core_vaa::parse_and_get_payload(vaa_bytes);
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
                *string::bytes(&asset_meta::symbol_to_string(&meta)),
                *string::bytes(&asset_meta::name_to_string(&meta)),
                b"", // No description necessary.
                option::none<Url>(), // No url necessary.
                ctx
            );

        transfer::share_object(coin_metadata);

        wrapped_coin::new(
            vaa_bytes,
            treasury_cap,
            coin_decimals,
            ctx
        )
    }

    public entry fun register_wrapped_coin<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        new_wrapped_coin: WrappedCoin<CoinType>,
        ctx: &mut TxContext,
    ) {
        let (vaa_bytes, treasury_cap, decimals) =
            wrapped_coin::destroy(new_wrapped_coin);

        let vaa = vaa::parse_verify_and_replay_protect(
            token_bridge_state,
            worm_state,
            vaa_bytes,
            ctx
        );
        let payload = core_vaa::destroy(vaa);

        let meta = asset_meta::deserialize(payload);
        let origin_chain = asset_meta::token_chain(&meta);
        let external_address = asset_meta::token_address(&meta);

        assert!(
            origin_chain != wormhole_state::chain_id(),
            E_WRAPPING_NATIVE_COIN
        );
        assert!(
            !state::is_registered_asset<CoinType>(token_bridge_state),
            E_WRAPPED_COIN_ALREADY_INITIALIZED
        );

        state::register_wrapped_asset<CoinType>(
            token_bridge_state,
            origin_chain,
            external_address,
            treasury_cap,
            decimals,
        );
    }
}
