module token_bridge::transfer_tokens {
    use sui::sui::SUI;
    use sui::tx_context::{TxContext};
    use sui::coin::{Self, Coin};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::ExternalAddress;
    use wormhole::myu16::{U16};

    use token_bridge::bridge_state::{Self, BridgeState};
    use token_bridge::transfer_result::{Self, TransferResult};
    use token_bridge::transfer::{Self};
    use token_bridge::normalized_amount::{Self};

    //TODO - should this return a sequence number?
    public fun transfer_tokens<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        token_chain: U16,
        token_address: ExternalAddress,
        wormhole_fee_coins: Coin<SUI>,
        recipient_chain: U16,
        recipient: ExternalAddress,
        relayer_fee: u64,
        nonce: u64,
        ctx: &mut TxContext
    ) {
        let result = transfer_tokens_internal<CoinType>(
            wormhole_state,
            bridge_state,
            coins,
            token_chain,
            token_address,
            relayer_fee,
            ctx
        );
        let (token_chain, token_address, normalized_amount, normalized_relayer_fee)
            = transfer_result::destroy(result);
        let transfer = transfer::create(
            normalized_amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            normalized_relayer_fee,
        );
        bridge_state::publish_message(
            wormhole_state,
            nonce,
            transfer::encode(transfer),
            wormhole_fee_coins,
            ctx
        )
    }

    public fun transfer_tokens_with_payload<CoinType>(){
        //TODO
    }

    fun transfer_tokens_internal<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        // TODO - how to encode whether coins is native or wrapped?
        //        if native, then store in token bridge, if wrapped, then burn
        //        right now, we pass in the token_chain and token_address to
        //        tell whether it is native or wrapped
        coins: Coin<CoinType>,
        token_chain: U16,
        token_address: ExternalAddress,
        _relayer_fee: u64,
        ctx: &mut TxContext
    ): TransferResult {
        let origin_info = bridge_state::create_origin_info(token_chain, token_address);
        let this_chain = wormhole::state::get_chain_id(wormhole_state);
        let amount = coin::value<CoinType>(&coins);
        if (token_chain == this_chain) {
            // token is native, so store token in bridge
            bridge_state::deposit<CoinType>(
                bridge_state,
                coins,
                origin_info,
                ctx,
            )
        } else{
            // token is wrapped, so burn it
            bridge_state::burn<CoinType>(
                bridge_state,
                coins,
                origin_info,
            )
        };

        //TODO - figure out how to get normalization decimals for token and relayer fee amounts
        let normalized_amount = normalized_amount::normalize(amount, 1);
        let normalized_relayer_fee = normalized_amount::normalize(_relayer_fee, 1);

        let transfer_result: TransferResult = transfer_result::create(
            token_chain,
            token_address,
            normalized_amount,
            normalized_relayer_fee,
        );
        transfer_result
    }
}
