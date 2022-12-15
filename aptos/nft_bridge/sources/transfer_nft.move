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
        token::deposit_token(&nft_bridge, token);
        let origin_info = state::get_origin_info(&token_id);
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
    use std::string::{Self, String};
    use std::coin::{Self};
    use std::bcs::{Self};
    use std::aptos_coin::{AptosCoin};
    use std::account::{Self};

    use aptos_token::token::{Self, TokenId};

    use wormhole::external_address::{Self};
    use wormhole::u16::{Self};

    use token_bridge::string32::{Self};

    use nft_bridge::state::{Self as nft_state};
    use nft_bridge::wrapped_test::{Self};
    use nft_bridge::transfer_nft::{Self};

    // test transfer wrapped NFT to another chain
    #[test(deployer=@deployer)]
    public fun test_transfer_wrapped_nft(deployer: &signer): TokenId {
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
            u16::from_u64(13), // to chain
            external_address::from_bytes(x"277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b"),
            0
        );
        // TODO - what shall we assert here?
        my_token_id
    }

    // test transfer native NFT to another chain
    // this function is called in complete_transfer::complete_transfer_test
    #[test(deployer=@deployer, recipient=@0x123456)]
    public fun test_transfer_native_nft(deployer: &signer, recipient: &signer): TokenId {
        wrapped_test::init_worm_and_nft_state(deployer);
        account::create_account_for_test(signer::address_of(recipient));

        // create new aptos-native nft collection unseen by the nft bridge
        let mutate_setting = vector[
                true, // TOKEN_MAX_MUTABLE
                true, // TOKEN_URI_MUTABLE
                true, // TOKEN_ROYALTY_MUTABLE_IND
                true, // TOKEN_DESCRIPTION_MUTABLE_IND
                true  // TOKEN_PROPERTY_MUTABLE_IND
        ];
        let token_mut_config = token::create_token_mutability_config(
            &mutate_setting
        );
        let collection_name = string::utf8(b"beef vault");
        token::create_collection(
            deployer,
            collection_name, // collection name
            string::utf8(b"beeeeef"), //description
            string::utf8(b"beef.com"), //uri
            10,
            mutate_setting
        );
        let token_name = string::utf8(b"beef token 1");
        let token_uri = string::utf8(b"beef.com/token_1");

        let token_data_id = token::create_tokendata(
            deployer,
            collection_name, // token collection name
            token_name, // token name
            string::utf8(b"beeff"), // description
            1, //supply cap 1
            token_uri,
            signer::address_of(deployer),
            0, // royalty_points_denominator
            0, // royalty_points_numerator
            token_mut_config, // see above
            vector<String>[string::utf8(b"TOKEN_BURNABLE_BY_CREATOR")],
            vector<vector<u8>>[bcs::to_bytes<bool>(&true)],
            vector<String>[string::utf8(b"bool")],
        );

        // have recipient register for a token store and enable direct deposit
        token::initialize_token_store(recipient);
        token::opt_in_direct_transfer(recipient, true);

        token::mint_token_to(
            deployer,
            signer::address_of(recipient),
            token_data_id,
            1
        );

        // withdraw a token from account
        let my_token_data_id = token::create_token_data_id(
            signer::address_of(deployer),
            collection_name,
            token_name,
        );
        let my_token_id = token::create_token_id(my_token_data_id, 0);
        let my_token = token::withdraw_token(
            recipient,
            my_token_id,
            1,
        );

        // transfer
        transfer_nft::transfer_nft(
            my_token, // the aptos-native NFT
            coin::zero<AptosCoin>(),
            u16::from_u64(22), // Aptos chain ID
            external_address::from_bytes(x"277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b"),
            0
        );
        my_token_id
    }
}
