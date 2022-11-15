module token_bridge::attest_token {
    use sui::sui::SUI;
    use sui::coin::Coin;
    use sui::transfer::transfer;
    use sui::tx_context::{Self, TxContext};

    use token_bridge::bridge_state::{BridgeState};

    /// Spec: we need this function to accurately get
    /// the name, symbol, decimals of CoinType
    ///
    /// Calls token_bridge::register_native_asset
    /// Make attest_token permissioned? Typically this function is not permissioned.
    public fun attest_token<CoinType>(
        _bridge_state: &mut BridgeState,
        fee_coins: Coin<SUI>,
        ctx: &mut TxContext
    ): u64 {
        // assert CoinType is not a wrapped asset

        // TODO - generate a 32-byte ExternalAddress for CoinType
        // TODO - register the native asset in bridge_state.move

        transfer(fee_coins, tx_context::sender(ctx));
        return 0 //sequence number of publish message
    }
}

// TODO - actual token attestation
//        currently blocked because there is no token metadata standard yet
//        we need to know where to get info like symbols, decimals, name, etc.
//        to do the attestation. The problems are summarized below.

// Problem 1) cannot compute a TokenHash of a CoinType, AKA a unique 32-byte identifier for it
// Problem 2) cannot find the token metadata for a CoinType, e.g. symbol, name, decimals...
