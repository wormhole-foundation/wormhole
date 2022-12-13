module nft_bridge::transfer_nft {
    use std::string::{String};
    use aptos_framework::aptos_coin::{AptosCoin};
    use aptos_framework::coin::{Self, Coin};
    use aptos_token::token::{Self, Token};

    use wormhole::u16::{Self, U16};
    use wormhole::external_address::{Self, ExternalAddress};

    use nft_bridge::state;
    use nft_bridge::transfer;
    use nft_bridge::uri::{Self, URI};

    use token_bridge::string32::{Self, String32};

    const E_AMOUNT_SHOULD_BE_ONE: u64 = 0;

    public entry fun transfer_nft_entry(
        sender: &signer,
        creators_address: address,
        collection: String,
        name: String,
        property_version: u64,
        recipient_chain: u64,
        recipient: vector<u8>,
        nonce: u64
    ) {
        let token_id = token::create_token_id_raw(creators_address, collection, name, property_version);
        let token = token::withdraw_token(sender, token_id, 1);
        let wormhole_fee = wormhole::state::get_message_fee();
        let wormhole_fee_coins = coin::withdraw<AptosCoin>(sender, wormhole_fee);
        transfer_nft(
            token,
            wormhole_fee_coins,
            u16::from_u64(recipient_chain),
            external_address::from_bytes(recipient),
            nonce
        );
    }

    public fun transfer_nft(
        token: Token,
        wormhole_fee_coins: Coin<AptosCoin>,
        recipient_chain: U16,
        recipient: ExternalAddress,
        nonce: u64
    ): u64 {
        // TODO(csongor): should we check that the supply of the token is 1? or
        // do anything with property_version?
        let (token_address,
             token_chain,
             symbol,
             name,
             token_id,
             uri,
        ) = lock_or_burn(token);

        let transfer = transfer::create(
            token_address,
            token_chain,
            symbol,
            name,
            token_id,
            uri,
            recipient,
            recipient_chain,
        );
        state::publish_message(
            nonce,
            transfer::encode(transfer),
            wormhole_fee_coins,
        )
    }

    #[test_only]
    public fun transfer_nft_test(
        token: Token,
    ): (
        ExternalAddress, // token_address
        U16, // token_chain
        String32, // symbol
        String32, // name
        ExternalAddress, // token_id
        URI, // URI
    ) {
        lock_or_burn(token)
    }

    /// Transfer a native (lock) or wrapped (burn) token from sender to nft_bridge.
    /// Returns the token's address and native chain
    fun lock_or_burn(token: Token): (
        ExternalAddress, // token_address
        U16, // token_chain
        String32, // symbol
        String32, // name
        ExternalAddress, // token_id
        URI, // URI
    ) {
        let nft_bridge: signer = state::nft_bridge_signer();

        // transfer coin to nft_bridge
        if (!coin::is_account_registered<AptosCoin>(@nft_bridge)) {
            coin::register<AptosCoin>(&nft_bridge);
        };

        let amount = token::get_token_amount(&token);
        assert!(amount == 1, E_AMOUNT_SHOULD_BE_ONE);

        let token_id = token::get_token_id(&token);
        let (creator, collection, name, property_version)
            = token::get_token_id_fields(&token_id);

        // First we deposit the token into the nft bridge signer account.
        // We do this irrespective of whether the token is a wrapped NFT or an
        // aptos-native NFT, even though in the former case we're going to burn the token anyway.
        //
        // The reason is that the aptos standard library doesn't expose
        // functionality to directly burn an NFT, the only two methods are
        // `burn` which allows the owner to burn a token, and `burn_by_creator`
        // which allows the creator to burn a token, given that it knows its
        // owner.
        //
        // For wrapped assets, the nft bridge is the creator, but at this point
        // in the control flow we don't know who the owner is, so we just burn directly.
        //
        // tldr; we burn wrapped tokens by first depositing them into the nft
        // bridge, due to poor design decisions in the aptos token standard
        token::deposit_token(&nft_bridge, token);

        if (state::is_wrapped_asset(&token_id)) {
            // now we burn the wrapped token to remove it from circulation
            token::burn(&nft_bridge, creator, collection, name, property_version, 1);
        } else {
            // if we're seeing this native token for the first time, store its token id
            state::set_native_asset_info(token_id);
        };

        let origin_info = state::get_origin_info(&token_id);

        let token_chain = state::get_origin_info_token_chain(&origin_info);
        let token_address = state::get_origin_info_token_address(&origin_info);
        let token_id = state::get_origin_info_token_id(&origin_info);
        let symbol = string32::from_bytes(b"");
        let token_data_id = token::create_token_data_id(creator, collection, name);
        let uri = uri::from_string(&token::get_tokendata_uri(creator, token_data_id));
        let name = string32::from_string(&name);

        return (
            token_address,
            token_chain,
            symbol,
            name,
            token_id,
            uri,
        )
    }


}

#[test_only]
module nft_bridge::transfer_nft_test {
    //TODO(csongor): test
}
