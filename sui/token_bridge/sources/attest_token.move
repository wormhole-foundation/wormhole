module token_bridge::attest_token {
    use sui::sui::SUI;
    use sui::coin::{Coin, CoinMetadata};
    use sui::tx_context::TxContext;

    use wormhole::state::{State as WormholeState};

    use token_bridge::bridge_state::{Self as state, BridgeState};
    use token_bridge::asset_meta;

    /// Spec: we need this function to accurately get
    /// the name, symbol, decimals of CoinType
    ///
    /// Calls token_bridge::register_native_asset
    /// Make attest_token permissioned? Typically this function is not permissioned.
    public fun attest_token<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        fee_coins: Coin<SUI>,
        ctx: &mut TxContext
    ): u64 {
        let asset_meta =
            state::register_native_asset<CoinType>(wormhole_state, bridge_state, coin_meta, ctx);
        let payload = asset_meta::encode(asset_meta);
        let nonce = 0;
        state::publish_message(
            wormhole_state,
            bridge_state,
            nonce,
            payload,
            fee_coins
        )
    }
}
