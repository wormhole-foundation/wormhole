module token_bridge::complete_transfer_with_payload {
    use aptos_framework::coin::{Self, Coin};

    use token_bridge::vaa;
    use token_bridge::transfer_with_payload::{Self as transfer, TransferWithPayload};
    use token_bridge::state;
    use token_bridge::wrapped;
    use token_bridge::normalized_amount;

    use wormhole::emitter::{Self, EmitterCapability};

    const E_INVALID_RECIPIENT: u64 = 0;
    const E_INVALID_TARGET: u64 = 1;

    // TODO(csongor): document this, and create an example contract receiving
    // such a transfer
    public fun submit_vaa<CoinType>(
        vaa: vector<u8>,
        emitter_cap: &EmitterCapability
    ): (Coin<CoinType>, TransferWithPayload) {
        let vaa = vaa::parse_verify_and_replay_protect(vaa);
        let transfer = transfer::parse(wormhole::vaa::destroy(vaa));

        let to_chain = transfer::get_to_chain(&transfer);
        assert!(to_chain == wormhole::state::get_chain_id(), E_INVALID_TARGET);

        let token_chain = transfer::get_token_chain(&transfer);
        let token_address = transfer::get_token_address(&transfer);
        let origin_info = state::create_origin_info(token_chain, token_address);

        state::assert_coin_origin_info<CoinType>(origin_info);

        let decimals = coin::decimals<CoinType>();

        let amount = normalized_amount::denormalize(transfer::get_amount(&transfer), decimals);

        // transfers with payload can only be redeemed by the recipient.
        let recipient = transfer::get_to(&transfer);
        assert!(
            recipient == emitter::get_external_address(emitter_cap),
            E_INVALID_RECIPIENT
        );

        let recipient_coins: Coin<CoinType>;

        if (state::is_wrapped_asset<CoinType>()) {
            recipient_coins = wrapped::mint<CoinType>(amount);
        } else {
            let token_bridge = state::token_bridge_signer();
            recipient_coins = coin::withdraw<CoinType>(&token_bridge, amount);
        };

        (recipient_coins, transfer)
    }
}

#[test_only]
module token_bridge::complete_transfer_with_payload_test {
}
