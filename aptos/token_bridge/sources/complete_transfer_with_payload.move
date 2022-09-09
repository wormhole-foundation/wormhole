module token_bridge::complete_transfer_with_payload {
    use aptos_framework::coin::{Self, Coin};

    use token_bridge::vaa;
    use token_bridge::transfer_with_payload::{Self as transfer, TransferWithPayload};
    use token_bridge::bridge_state as state;
    use token_bridge::wrapped;

    use wormhole::u256;
    use wormhole::emitter::{Self, EmitterCapability};
    use wormhole::deserialize;

    use wormhole::cursor;

    const E_INVALID_RECIPIENT: u64 = 0;

    // TODO(csongor): document this, and create an example contract receiving
    // such a transfer
    public entry fun submit_vaa<CoinType>(
        vaa: vector<u8>,
        emitter_cap: &EmitterCapability
    ): (Coin<CoinType>, TransferWithPayload) {
        let vaa = vaa::parse_verify_and_replay_protect(vaa);
        let transfer = transfer::parse(wormhole::vaa::destroy(vaa));
        let token_chain = transfer::get_token_chain(&transfer);
        let token_address = transfer::get_token_address(&transfer);
        let origin_info = state::create_origin_info(token_address, token_chain);

        state::assert_coin_origin_info<CoinType>(origin_info);

        // Convert to u64. Aborts in case of overflow
        let amount = u256::as_u64(transfer::get_amount(&transfer));

        let recipient = transfer::get_to(&transfer);
        let cur = cursor::init(recipient);
        let recipient: u128 = u256::as_u128(deserialize::deserialize_u256(&mut cur));
        cursor::destroy_empty(cur);

        assert!(recipient == emitter::get_emitter(emitter_cap), E_INVALID_RECIPIENT);

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
