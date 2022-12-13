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
    const E_FUNGIBLE_TOKEN: u64 = 1;

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
        let wormhole_fee_coins: Coin<AptosCoin>;
        if (wormhole_fee > 0) {
            wormhole_fee_coins = coin::withdraw<AptosCoin>(sender, wormhole_fee);
        } else {
            wormhole_fee_coins = coin::zero<AptosCoin>();
        };
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
            transfer::encode(&transfer),
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
        // NOTE: the way aptos tokens are designed, it is possible to mint
        // multiple copies of a token. See the README for an explanation
        let amount = token::get_token_amount(&token);
        assert!(amount == 1, E_AMOUNT_SHOULD_BE_ONE);

        let token_id = token::get_token_id(&token);
        let (creator, collection, name, property_version)
            = token::get_token_id_fields(&token_id);

        // We deposit the token into the nft bridge signer account.address,
        // regardless of whether it is a wrapped or native token. In the native
        // token case, this just means custodying the tokens as expected. In the
        // wrapped case, we do this because there are only two ways to burn a
        // token: either by the owner, or the creator: `burn` and
        // `burn_by_creator` respectively. In both cases, the owner of the token
        // must be known, so we deposit to the nft bridge first.
        //
        // tldr; first deposit token into nft bridge by convention, then if it is wrapped,
        // then load the creator signer from nft bridge state and burn it using `burn_by_creator`.
        // Disallow `burn` (by owner) to avoid edge cases
        let nft_bridge: signer = state::nft_bridge_signer();
        token::deposit_token(&nft_bridge, token);
        let (origin_info, external_token_id) = state::get_origin_info(&token_id);

        // We need to grab the URI *before* burning the NFT, otherwise its
        // tokendata will no longer be available
        let token_data_id = token::create_token_data_id(creator, collection, name);
        let uri = uri::from_string(&token::get_tokendata_uri(creator, token_data_id));

        // The symbol field will be set to empty for aptos native NFTs (as they
        // do not have an equivalent field). For wrapped assets, it's simply
        // preserved from the original metadata.
        let symbol: String32;
        let external_name: String32;

        if (state::is_wrapped_asset(&token_id)) {
            (external_name, symbol) = state::get_wrapped_asset_name_and_symbol(
                origin_info,
                collection,
                external_token_id
            );
            // burn the wrapped token to remove it from circulation
            let creator_signer = state::get_wrapped_asset_signer(origin_info);
            token::burn_by_creator(
                &creator_signer,
                std::signer::address_of(&nft_bridge),
                collection,
                name,
                property_version,
                1
            );
        } else {
            symbol = string32::from_bytes(b"");
            external_name = string32::from_string(&collection);
            // NOTE: The way Aptos Tokens are designed, it is possible to mint
            // multiple copies of a token, if its property_version is 0, and the
            // tokendata's `maximum` value is greater than 1.  We could check
            // that `maximum` is 1, but the `maximum` field can be mutated by
            // the collection's creator by calling
            // `token::mutate_tokendata_maximum`, so that check would not
            // suffice.  We could additionally check that the mutability config
            // of the maximum field is set to immutable, which would ensure that
            // if it's 1, then it stays 1. However, I expect most collections to
            // just leave that field as `true` even if there's no intention of
            // mutating that field.
            //
            // Instead, we just ensure that the NFT bridge can hold a maximum of
            // 1 copy of this token. That is, even if the token has a supply
            // larger than 1, only 1 of the tokens can be bridged out at any
            // given time, so they effectively behave as NFTs outside of Aptos,
            // even when they're tecnhically fungible on aptos.
            assert!(token::balance_of(@nft_bridge, token_id) == 1, E_FUNGIBLE_TOKEN);
            // if we're seeing this native token for the first time, store its token id
            state::set_native_asset_info(token_id);
        };
        let token_chain = state::get_origin_info_token_chain(&origin_info);
        let token_address = state::get_origin_info_token_address(&origin_info);

        return (
            token_address,
            token_chain,
            symbol,
            external_name,
            external_token_id,
            uri,
        )
    }
}

#[test_only]
module nft_bridge::transfer_nft_test {
    use std::signer;
    use std::string::{Self, String};

    use aptos_framework::account;
    use aptos_token::token;

    use nft_bridge::wrapped_test;
    use nft_bridge::transfer_nft;

    // test transfer wrapped NFT to another chain
    #[test(deployer = @deployer, sender = @0x123456)]
    public fun test_transfer_wrapped_nft(deployer: &signer, sender: &signer) {
        wormhole::wormhole_test::setup(0);
        nft_bridge::nft_bridge::init_test(deployer);

        aptos_framework::aptos_account::create_account(signer::address_of(sender));
        token::opt_in_direct_transfer(sender, true);

        // ------ Create wrapped collection

        let collection_name = string::utf8(b"collection name");

        let creator = wrapped_test::create_wrapped_nft_collection(
            signer::address_of(sender),
            collection_name
        );

        // ------ Construct token_id for follow-up queries

        // NOTE: the x"01" comes from the
        // `wrapped_test::create_wrapped_nft_collection` function, and
        // (currently) we render it as hex
        let token_name = string::utf8(b"0000000000000000000000000000000000000000000000000000000000000001");

        let token_id = token::create_token_id_raw(
            signer::address_of(&creator),
            collection_name,
            token_name,
            0 // property_version
        );

        // ------ Check that `sender` now owns the token

        assert!(token::balance_of(signer::address_of(sender), token_id) == 1, 0);
        assert!(token::check_tokendata_exists(signer::address_of(&creator), collection_name, token_name), 0);

        // ------ Transfer the tokens
        transfer_nft::transfer_nft_entry(
            sender,
            signer::address_of(&creator),
            string::utf8(b"collection name"), // collection
            token_name,
            0, // property_version
            3, // recipient chain
            x"0000000000000000000000000000000000000000000000000000000000FAFAFA",
            0
        );

        // ------ Check that `sender` no longer owns the token, and that it no longer exists

        assert!(token::balance_of(signer::address_of(sender), token_id) == 0, 0);
        assert!(!token::check_tokendata_exists(signer::address_of(&creator), collection_name, token_name), 0);
    }

    // test transfer native NFT to another chain
    // this function is called in complete_transfer::complete_transfer_test
    #[test(deployer = @deployer, creator = @0x654321, sender = @0x123456)]
    public fun test_transfer_native_nft(deployer: &signer, creator: address, sender: &signer) {
        // ------ Setup
        wormhole::wormhole_test::setup(0);
        nft_bridge::nft_bridge::init_test(deployer);

        aptos_framework::aptos_account::create_account(signer::address_of(sender));
        token::opt_in_direct_transfer(sender, true);

        let creator = account::create_account_for_test(creator);

        // ------ Create a collection under `creator`

        let collection_name = string::utf8(b"my test collection");
        create_collection(&creator, collection_name);

        // ------ Mint two token to `sender` from the collection we just created

        let token_name = string::utf8(b"my test token");
        mint_token_to(
            &creator,
            signer::address_of(sender),
            collection_name,
            token_name,
            2
        );


        // ------ Construct token_id for follow-up queries

        let token_id = token::create_token_id_raw(
            signer::address_of(&creator),
            collection_name,
            token_name,
            0 // property_version
        );

        // ------ Check that `sender` now owns the token

        assert!(token::balance_of(signer::address_of(sender), token_id) == 2, 0);

        // ------ Transfer the tokens
        transfer_nft::transfer_nft_entry(
            sender,
            signer::address_of(&creator),
            collection_name,
            token_name,
            0, // property_version
            3, // recipient chain
            x"0000000000000000000000000000000000000000000000000000000000FAFAFA",
            0
        );

        // ------ Check that `sender` no longer owns the token, but `nft_bridge` does

        assert!(token::balance_of(signer::address_of(sender), token_id) == 1, 0);
        assert!(token::balance_of(@nft_bridge, token_id) == 1, 0);
    }

    // This test case checks that we handle 'fungible' NFTs correctly (see NOTE in `lock_or_burn`)
    #[test(deployer=@deployer, creator = @0x654321, sender = @0x123456)]
    #[expected_failure(abort_code = 1, location = nft_bridge::transfer_nft)]
    public fun test_transfer_native_nft_fungible(deployer: &signer, creator: &signer, sender: &signer) {
        // We first invoke `test_transfer_native_nft_fungible` which will
        // deposit one of the tokens into the NFT bridge.
        // Then we will attempt to transfer another one, which should fail
        test_transfer_native_nft(deployer, signer::address_of(creator), sender);

        let collection_name = string::utf8(b"my test collection");
        let token_name = string::utf8(b"my test token");

        let token_id = token::create_token_id_raw(
            signer::address_of(creator),
            collection_name,
            token_name,
            0 // property_version
        );

        // ------ Check that `sender` and `nft_bridge` both own one token

        assert!(token::balance_of(signer::address_of(sender), token_id) == 1, 0);
        assert!(token::balance_of(@nft_bridge, token_id) == 1, 0);

        // ------ Attempt to transfer the token
        transfer_nft::transfer_nft_entry(
            sender,
            signer::address_of(creator),
            collection_name,
            token_name,
            0, // property_version
            3, // recipient chain
            x"0000000000000000000000000000000000000000000000000000000000FAFAFA",
            0
        );
    }

    // ------------ Helper functions --------------------

    /// Create a collection with a given name
    public fun create_collection(creator: &signer, collection_name: String) {
        token::create_collection(
            creator,
            collection_name, // collection name
            string::utf8(b"beeeeef"), //description
            string::utf8(b"beef.com"), //uri
            0,
            vector[true, true, true]
        );
    }

    public fun mint_token_to(
        creator: &signer,
        recipient: address,
        collection_name: String,
        token_name: String,
        amount: u64
    ) {
        let token_data_id = token::create_tokendata(
            creator,
            collection_name,
            token_name,
            string::utf8(b"some description"),
            amount,
            string::utf8(b"some uri"),
            signer::address_of(creator), // royalty payee
            0, // royalty_points_denominator
            0, // royalty_points_numerator
            token::create_token_mutability_config(&vector[true, true, true, true, true]), // allow all fields to be mutated
            vector[],
            vector[],
            vector[],
        );

        token::mint_token_to(
            creator,
            recipient,
            token_data_id,
            amount
        );
    }

}
