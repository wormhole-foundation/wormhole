module token_bridge::attest_token {
    use sui::sui::SUI;
    use sui::coin::Coin;
    use sui::transfer::transfer;
    use sui::tx_context::{Self, TxContext};

    use wormhole::state::{State as WormholeState};
    use token_bridge::bridge_state::{Self as state, BridgeState};

    /// Spec: we need this function to accurately get
    /// the name, symbol, decimals of CoinType
    ///
    /// Calls token_bridge::register_native_asset
    /// Make attest_token permissioned? Typically this function is not permissioned.
    public fun attest_token<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        fee_coins: Coin<SUI>,
        ctx: &mut TxContext
    ): u64 {
        state::register_native_asset<CoinType>(wormhole_state, bridge_state, ctx);
        transfer(fee_coins, tx_context::sender(ctx));
        // TODO: publish message
        return 0 //sequence number of publish message
    }
}
