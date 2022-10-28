module token_bridge::complete_transfer {
    use sui::tx_context::{TxContext};
    //use sui::dynamic_object_field::{Self};

    use wormhole::myvaa::{VAA};
    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{ExternalAddress};

    use token_bridge::bridge_state::{Self as bridge_state, BridgeState};
    use token_bridge::myvaa::{Self as vaa};
    use token_bridge::transfer::{Self, Transfer};

    // mint wrapped tokens if incoming tokens is from foreign chain (get treasury cap for this)
    // get tokens from token bridge treasury and send to receiver if the token is native (get coin store for this)

    public entry fun submit_vaa<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        vaa: vector<u8>,
        ctx: &mut TxContext,
        fee_recipient: address
    ): Transfer {

        let vaa = vaa::parse_verify_and_replay_protect(
            wormhole_state,
            bridge_state,
            vaa,
            ctx
        );

        let transfer = transfer::parse(wormhole::vaa::destroy(vaa));
        complete_transfer<CoinType>(&transfer, fee_recipient);
        transfer
    }

    fun complete_transfer_foreign_asset<CoinType>(
        transfer: &Transfer,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        fee_recipient: address
    ) {

        let to_chain = transfer::get_to_chain(transfer);
        assert!(to_chain == wormhole::state::get_chain_id(), E_INVALID_TARGET);

        let token_chain = transfer::get_token_chain(transfer);
        let token_address = transfer::get_token_address(transfer);
        let origin_info = state::create_origin_info(token_chain, token_address);

        //state::assert_coin_origin_info<CoinType>(origin_info);

        // TODO - Get decimals from treasury cap, then do normalization
        //let decimals = coin::decimals<CoinType>();
        // let amount = normalized_amount::denormalize(transfer::get_amount(transfer), decimals);
        // let fee_amount = normalized_amount::denormalize(transfer::get_fee(transfer), decimals);

        let recipient = from_bcs::to_address(get_bytes(&transfer::get_to(transfer)));

        let recipient_coins: Coin<CoinType>;

        // get treasury cap and mint wrapped tokens to receiver
        // TODO - call bridge_state::mint to mint coins

        recipient_coins = wrapped::mint<CoinType>(amount);

        // take out fee from the recipient's coins. `extract` will revert
        // if fee > amount
        let fee_coins = coin::extract(&mut recipient_coins, fee_amount);
        coin::deposit(recipient, recipient_coins);
        coin::deposit(fee_recipient, fee_coins);
    }

}