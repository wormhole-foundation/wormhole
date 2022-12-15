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

        // By convention, we deposit the token into the nft bridge signer account.address,
        // regardless of whether it is a wrapped or native token. (In the wrapped case, it
        // will be burned later by the creator resource account, which is stored in the NFT
        // bridge state and which can be looked up using `get_wrapped_asset_signer`).
        //
        // The standard library does not expose methods to directly burn an NFT. The only
        // two methods are `burn` which allows the owner to burn a token, and `burn_by_creator`
        // which allows the creator to burn a token. Whether a token can be burned at all, burned
        // by owner, or burned by creator is set in the property keys field when calling
        // token::create_tokendata. We only allow `burn_by_creator` to avoid an edge case whereby
        // a user burns a wrapped token and can no longer bridge it back to the origin chain.
        //
        // tldr; first deposit token into nft bridge by convention, then if it is wrapped,
        // then load the creator signer from nft bridge state and burn it using `burn_by_creator`.
        // Disallow `burn` (by owner) to avoid edge cases
        let origin_info = state::get_origin_info(&token_id);
        token::deposit_token(&nft_bridge, token);
        if (state::is_wrapped_asset(&token_id)) {
            // burn the wrapped token to remove it from circulation
            let creator_signer = state::get_wrapped_asset_signer(origin_info);
            token::burn_by_creator(&creator_signer, creator, collection, name, property_version, 1);
        } else {
            // if we're seeing this native token for the first time, store its token id
            state::set_native_asset_info(token_id);
        };
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
    use std::signer;
    use std::string::{Self};
    use std::coin::{Self};
    use std::aptos_coin::{AptosCoin};

    use aptos_token::token::{Self};

    use wormhole::external_address::{Self};
    use wormhole::u16::{Self};

    use token_bridge::string32::{Self};

    use nft_bridge::state::{Self as nft_state};
    use nft_bridge::wrapped_test::{Self};
    use nft_bridge::transfer_nft::{Self};

    // test transfer wrapped NFT to another chain
    #[test(deployer=@deployer)]
    fun test_transfer_wrapped_nft(deployer: &signer) {
        // mint 99 NFTs to deployer
        wrapped_test::test_create_wrapped_nft_collection(deployer);

        // withdraw a token from account
        let token_address = external_address::from_bytes(x"0000");
        let token_chain = u16::from_u64(14);
        let token_id = external_address::from_bytes(x"0001");
        let token_name =  string32::from_bytes(x"aa");

        let origin_info = nft_state::create_origin_info(
            token_chain,
            token_address,
            token_id,
        );
        let my_signer = nft_state::get_wrapped_asset_signer(origin_info);

        let my_token_data_id = token::create_token_data_id(
            signer::address_of(&my_signer),
            string32::to_string(&token_name),
            string::utf8(x"01"),
        );
        let my_token_id = token::create_token_id(my_token_data_id, 0);
        let my_token = token::withdraw_token(
            &my_signer,
            my_token_id,
            1,
        );

        // transfer nft
        transfer_nft::transfer_nft(
            my_token,
            coin::zero<AptosCoin>(),
            u16::from_u64(2), // to chain
            external_address::from_bytes(x"0101010101010101"),
            0
        );

        // TODO - what shall we assert here?
    }
}
