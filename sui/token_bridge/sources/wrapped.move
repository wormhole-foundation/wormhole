/// This module uses the one-time witness (OTW)
/// Sui one-time witness pattern reference: https://examples.sui.io/basics/one-time-witness.html
module token_bridge::wrapped {
    use std::option::{Self};

    use sui::tx_context::{TxContext};
    use sui::coin::{TreasuryCap};
    use sui::object::{Self, UID};
    use sui::coin::{Self};
    use sui::url::{Url};
    use sui::transfer::{Self};

    use token_bridge::bridge_state::{Self, BridgeState};
    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::vaa;
    use token_bridge::string32::{Self};

    use wormhole::state::{Self as state, State as WormholeState};
    use wormhole::myvaa as core_vaa;

    const E_WRAPPING_NATIVE_COIN: u64 = 0;
    const E_WRAPPING_REGISTERED_NATIVE_COIN: u64 = 1;
    const E_WRAPPED_COIN_ALREADY_INITIALIZED: u64 = 2;

    /// Wrapped assets are created in two steps.
    /// 1) The coin is initialised by calling `create_wrapped_coin` in the
    /// `init` function of a OTW module.
    /// 2) The coin is registered in the token bridge in
    /// `register_wrapped_coin`.
    ///
    /// Since Step 1. takes places in an untrusted context, we want to remove
    /// all degrees of freedom. To this end, `create_wrapped_coin` just takes a
    /// VAA, and returns a `NewWrappedCoin` object. That's the only way to
    /// create a `NewWrappedCoin` object. Then this object can be passed to
    /// `register_wrapped_coin` in Step 2.
    ///
    /// This setup ensures that we don't have to trust (or verify) that the OTW
    /// initialiser did the right thing.
    ///
    /// TODO: it would be nice if we could also enforce that the OTW struct's
    /// name matches the token symbol being registered. Currently there's no way
    /// to do this in the sui framework.
    struct NewWrappedCoin<phantom CoinType> has key, store {
        id: UID,
        vaa_bytes: vector<u8>,
        treasury_cap: TreasuryCap<CoinType>,
    }

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
    /// do any checks other than the VAA in `NewWrappedCoin` is valid.
    public fun create_wrapped_coin<CoinType: drop>(
        vaa_bytes: vector<u8>,
        coin_witness: CoinType,
        ctx: &mut TxContext
    ): NewWrappedCoin<CoinType> {
        let payload = core_vaa::parse_and_get_payload(vaa_bytes);
        let asset_meta: AssetMeta = asset_meta::parse(payload);

        // The amounts in the token bridge payload are truncated to 8 decimals
        // in each of the contracts when sending tokens out, so there's no
        // precision beyond 10^-8. We could preserve the original number of
        // decimals when creating wrapped assets, and "untruncate" the amounts
        // on the way out by scaling back appropriately. This is what most other
        // chains do, but untruncating from 8 decimals to 18 decimals loses
        // log2(10^10) ~ 33 bits of precision, which we cannot afford on Aptos
        // (and Solana), as the coin type only has 64bits to begin with.
        // Contrast with Ethereum, where amounts are 256 bits.
        // So we cap the maximum decimals at 8 when creating a wrapped token.
        let max_decimals: u8 = 8;

        let parsed_decimals = asset_meta::get_decimals(&asset_meta);
        let symbol = asset_meta::get_symbol(&asset_meta);
        let name = asset_meta::get_name(&asset_meta);

        let decimals = if (max_decimals < parsed_decimals) max_decimals else parsed_decimals;
        let (treasury_cap, coin_metadata) = coin::create_currency<CoinType>(
            coin_witness,
            decimals,
            string32::to_bytes(&symbol),
            string32::to_bytes(&name),
            x"", //empty description
            option::none<Url>(), //empty url
            ctx
        );
        transfer::share_object(coin_metadata);
        NewWrappedCoin { id: object::new(ctx), vaa_bytes, treasury_cap }
    }

    public entry fun register_wrapped_coin<CoinType>(
        state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        new_wrapped_coin: NewWrappedCoin<CoinType>,
        ctx: &mut TxContext,
    ) {
        let NewWrappedCoin { id, vaa_bytes, treasury_cap } = new_wrapped_coin;
        object::delete(id);

        let vaa = vaa::parse_verify_and_replay_protect(
            state,
            bridge_state,
            vaa_bytes,
            ctx
        );
        let payload = core_vaa::destroy(vaa);

        let metadata = asset_meta::parse(payload);
        let origin_chain = asset_meta::get_token_chain(&metadata);
        let external_address = asset_meta::get_token_address(&metadata);
        let wrapped_asset_info =
            bridge_state::create_wrapped_asset_info(
                origin_chain,
                external_address,
                treasury_cap,
                ctx
            );
        assert!(origin_chain != state::get_chain_id(state), E_WRAPPING_NATIVE_COIN);
        assert!(!bridge_state::is_registered_native_asset<CoinType>(bridge_state), E_WRAPPING_REGISTERED_NATIVE_COIN);
        assert!(!bridge_state::is_wrapped_asset<CoinType>(bridge_state), E_WRAPPED_COIN_ALREADY_INITIALIZED);
        bridge_state::register_wrapped_asset<CoinType>(bridge_state, wrapped_asset_info);
    }
}
