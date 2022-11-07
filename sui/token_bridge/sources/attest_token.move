module token_bridge::attest_token {
    use sui::sui::SUI;
    use sui::coin::Coin;
    use sui::transfer::transfer;
    use sui::tx_context::{Self, TxContext};

    use token_bridge::bridge_state::{BridgeState};

    // mock token attestation function
    public fun attest_token<CoinType>(
        _bridge_state: &mut BridgeState,
        fee_coins: Coin<SUI>,
        ctx: &mut TxContext
    ): u64 {
        transfer(fee_coins, tx_context::sender(ctx));
        return 0 //sequence number of publish message
    }
}

// TODO - actual token attestation
//        currently blocked because there is no token metadata standard yet
//        we need to know where to get info like symbols, decimals, name, etc.
//        to do the attestation
