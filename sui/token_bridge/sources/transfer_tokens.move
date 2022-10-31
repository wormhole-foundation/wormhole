module token_bridge::transfer_tokens {
    use sui::tx_context::{TxContext};

    use wormhole::state::{State as WormholeState};

    use token_bridge::bridge_state::{BridgeState};

    // If transfer to different chain
    // - accept tokens from user, store them in token bridge

    public entry fun submit_vaa<CoinType>(
        _wormhole_state: &mut WormholeState,
        _bridge_state: &mut BridgeState,
        _vaa: vector<u8>,
        _ctx: &mut TxContext
    ) {
        transfer_tokens_internal<CoinType>();
    }

    fun transfer_tokens_internal<CoinType>(){
        //TODO
    }

}