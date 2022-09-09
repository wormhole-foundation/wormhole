module token_bridge::complete_transfer {
    use aptos_std::from_bcs;
    use aptos_framework::coin::{Self, Coin};

    use token_bridge::vaa;
    use token_bridge::transfer::{Self, Transfer};
    use token_bridge::bridge_state as state;
    use token_bridge::wrapped;

    use wormhole::u256;

    public entry fun submit_vaa<CoinType>(vaa: vector<u8>, fee_recipient: address): Transfer {
        let vaa = vaa::parse_verify_and_replay_protect(vaa);
        let transfer = transfer::parse(wormhole::vaa::destroy(vaa));
        let token_chain = transfer::get_token_chain(&transfer);
        let token_address = transfer::get_token_address(&transfer);
        let origin_info = state::create_origin_info(token_address, token_chain);

        state::assert_coin_origin_info<CoinType>(origin_info);

        // Convert to u64. Aborts in case of overflow
        let amount = u256::as_u64(transfer::get_amount(&transfer));
        let fee_amount = u256::as_u64(transfer::get_fee(&transfer));

        let recipient = from_bcs::to_address(transfer::get_to(&transfer));

        let recipient_coins: Coin<CoinType>;

        if (state::is_wrapped_asset<CoinType>()) {
            recipient_coins = wrapped::mint<CoinType>(amount);
        } else {
            let token_bridge = state::token_bridge_signer();
            recipient_coins = coin::withdraw<CoinType>(&token_bridge, amount);
        };

        // take out fee from the recipient's coins. `extract` will revert
        // if fee > amount
        let fee_coins = coin::extract(&mut recipient_coins, fee_amount);
        coin::deposit(recipient, recipient_coins);
        coin::deposit(fee_recipient, fee_coins);

        transfer
    }

}

#[test_only]
module token_bridge::complete_transfer_test {

}
