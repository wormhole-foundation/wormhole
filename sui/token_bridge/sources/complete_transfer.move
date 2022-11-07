module token_bridge::complete_transfer {
    use sui::tx_context::{TxContext};
    use sui::transfer::{Self as transfer_object};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self};

    use token_bridge::bridge_state::{Self as bridge_state, BridgeState};
    use token_bridge::vaa::{Self};
    use token_bridge::transfer::{Self, Transfer};
    use token_bridge::normalized_amount::{denormalize};

    const E_INVALID_TARGET: u64 = 0;

    public entry fun submit_vaa<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        vaa: vector<u8>,
        _fee_recipient: address,
        ctx: &mut TxContext
    ) {

        let vaa = vaa::parse_verify_and_replay_protect(
            wormhole_state,
            bridge_state,
            vaa,
            ctx
        );

        let transfer = transfer::parse(wormhole::myvaa::destroy(vaa));

        complete_transfer<CoinType>(
            &transfer,
            wormhole_state,
            bridge_state,
            _fee_recipient,
            ctx
        );
    }

    fun complete_transfer<CoinType>(
        transfer: &Transfer,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        _fee_recipient: address,
        ctx: &mut TxContext
    ) {
        let to_chain = transfer::get_to_chain(transfer);
        let this_chain = wormhole::state::get_chain_id(wormhole_state);
        assert!(to_chain == this_chain, E_INVALID_TARGET);

        let token_chain = transfer::get_token_chain(transfer);
        let token_address = transfer::get_token_address(transfer);
        let origin_info = bridge_state::create_origin_info(token_chain, token_address);

        let recipient = external_address::to_address(external_address::get_bytes(&transfer::get_to(transfer)));

        // TODO - figure out actual number of decimal places to denormalize by
        //        Where to store or how to find out #decimals for coin?
        let amount = denormalize(transfer::get_amount(transfer), 0);

        let recipient_coins;
        if (token_chain==this_chain){
            recipient_coins = bridge_state::withdraw<CoinType>(
                bridge_state,
                amount,
                origin_info,
                ctx
            )
        } else{
            recipient_coins = bridge_state::mint<CoinType>(
                bridge_state,
                amount,
                origin_info,
                ctx
            );
        };
        transfer_object::transfer(recipient_coins, recipient);
    }
    //TODO: send fee to fee_recipient
}