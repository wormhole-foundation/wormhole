module nft_bridge::complete_transfer {
    use aptos_std::from_bcs;
    use aptos_token::token;

    use nft_bridge::vaa;
    use nft_bridge::transfer::{Self, Transfer};
    use nft_bridge::state;
    use nft_bridge::wrapped;
    use nft_bridge::token_hash;

    use token_bridge::string32;

    use wormhole::state as wormhole_state;
    use wormhole::external_address;

    const E_INVALID_TARGET: u64 = 0;
    const E_INVALID_TOKEN_ADDRESS: u64 = 1;

    public fun submit_vaa(vaa: vector<u8>): Transfer {
        let vaa = vaa::parse_verify_and_replay_protect(vaa);
        let transfer = transfer::parse(wormhole::vaa::destroy(vaa));
        complete_transfer(&transfer);
        transfer
    }

    public entry fun submit_vaa_entry(vaa: vector<u8>) {
        submit_vaa(vaa);
    }

    /// Submits the complete transfer VAA and registers the NFT for the
    /// recipient if not already registered.
    public entry fun submit_vaa_and_register_entry(recipient: &signer, vaa: vector<u8>) {
        token::opt_in_direct_transfer(recipient, true);
        submit_vaa(vaa);
    }

    #[test_only]
    public fun test(transfer: &Transfer) {
        complete_transfer(transfer)
    }

    fun complete_transfer(transfer: &Transfer) {
        let to_chain = transfer::get_to_chain(transfer);
        assert!(to_chain == wormhole::state::get_chain_id(), E_INVALID_TARGET);

        let token_chain = transfer::get_token_chain(transfer);
        let token_id = transfer::get_token_id(transfer);
        let uri = transfer::get_uri(transfer);

        let recipient = from_bcs::to_address(external_address::get_bytes(&transfer::get_to(transfer)));

        let is_wrapped_asset: bool = token_chain != wormhole_state::get_chain_id();
        if (is_wrapped_asset) {
            let (creator, collection) = wrapped::create_or_find_wrapped_nft_collection(transfer);
            wrapped::mint_to(&creator, recipient, string32::to_string(&collection), &token_id, uri);
        } else {
            // native, we must have seen this token before (on the way out)
            let nft_bridge = state::nft_bridge_signer();

            // Since the way we derive external ids for tokens (see
            // token_hash.move) guarantees that the ids are globally unique, we
            // only need the token_id field of the transfer VAA to identify the
            // token in question, the token_address is not necessary...
            let token_hash = token_hash::from_external_address(token_id);
            let token_id = state::get_native_asset_info(token_hash);

            // ...nevertheless, as a sanity check, we derive the collection hash
            // and ensure that it comes from collection that token_address claims.
            let (collection_hash, _) = token_hash::derive(&token_id);
            let collection_hash = token_hash::get_collection_external_address(&collection_hash);
            assert!(collection_hash == transfer::get_token_address(transfer), E_INVALID_TOKEN_ADDRESS);

            token::transfer(&nft_bridge, token_id, recipient, 1);
        };
    }

}

#[test_only]
module nft_bridge::complete_transfer_test {
    use std::signer;
    use std::string::{Self, String};
    use std::bcs;

    use aptos_framework::account;
    use aptos_token::token;

    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::u16;

    use token_bridge::string32;

    use nft_bridge::transfer;
    use nft_bridge::uri;
    use nft_bridge::complete_transfer;
    use nft_bridge::state;
    use nft_bridge::transfer_nft;
    use nft_bridge::token_hash;
    use nft_bridge::transfer_nft_test;
    use nft_bridge::wrapped_token_name;

    // ------ Wrapped asset tests

    // Test that complete_transfer for wrapped token works
    #[test(deployer = @deployer, recipient = @0x1234)]
    public fun test_complete_transfer_wrapped_asset(deployer: &signer, recipient: address) {
        wormhole::wormhole_test::setup(0);
        nft_bridge::nft_bridge::init_test(deployer);

        let recipient_signer = aptos_framework::account::create_account_for_test(recipient);
        complete_transfer_wrapped_helper(&recipient_signer, external_address::from_bytes(x"01"));
    }

    // Test that the same token can't be transferred in twice without being
    // transferred out first
    #[test(deployer = @deployer, recipient1 = @0x1234, recipient2 = @0x5678)]
    #[expected_failure(abort_code = 524297, location = aptos_token::token)]
    public fun test_complete_transfer_wrapped_asset_twice(
        deployer: &signer,
        recipient1: address,
        recipient2: address
    ) {
        wormhole::wormhole_test::setup(0);
        nft_bridge::nft_bridge::init_test(deployer);

        let recipient_signer1 = aptos_framework::account::create_account_for_test(recipient1);
        let recipient_signer2 = aptos_framework::account::create_account_for_test(recipient2);
        complete_transfer_wrapped_helper(&recipient_signer1, external_address::from_bytes(x"01"));
        complete_transfer_wrapped_helper(&recipient_signer2, external_address::from_bytes(x"01"));
    }

    // Test that transferring a token in, then out, then back in again works
    #[test(deployer = @deployer, recipient = @0x1234)]
    public fun test_complete_transfer_wrapped_there_and_back(
        deployer: &signer,
        recipient: address,
    ) {
        wormhole::wormhole_test::setup(0);
        nft_bridge::nft_bridge::init_test(deployer);

        let recipient_signer = aptos_framework::account::create_account_for_test(recipient);

        let token_id = external_address::from_bytes(x"01");

        // step 1) transfer in
        complete_transfer_wrapped_helper(&recipient_signer, token_id);

        let token_address = external_address::from_bytes(x"09");
        let token_chain = u16::from_u64(14);
        let origin_info = state::create_origin_info(token_chain, token_address);
        let creator = state::get_wrapped_asset_signer(origin_info);

        let expected_token_name = string::utf8(b"0000000000000000000000000000000000000000000000000000000000000001");

        // step 2) transfer out
        transfer_nft::transfer_nft_entry(
            &recipient_signer,
            signer::address_of(&creator),
            string::utf8(b"my name"),
            expected_token_name,
            0, // property_version
            1, // recipient chain (doesn't matter)
            x"0000000000000000000000000000000000000000000000000000000000012345", // recipient (doesn't matter)
            0 // nonce (doesn't matter)
        );

        // step 3) transfer in again
        complete_transfer_wrapped_helper(&recipient_signer, token_id);
    }

    // Test that transferring a token in, then out, preserves the metadata
    #[test(deployer = @deployer, recipient = @0x1234)]
    public fun test_complete_transfer_wrapped_preserves_metadata(
        deployer: &signer,
        recipient: address,
    ) {
        wormhole::wormhole_test::setup(0);
        nft_bridge::nft_bridge::init_test(deployer);

        let recipient_signer = aptos_framework::account::create_account_for_test(recipient);

        let token_id = external_address::from_bytes(x"01");

        // step 1) transfer in
        complete_transfer_wrapped_helper(&recipient_signer, token_id);

        let token_address = external_address::from_bytes(x"09");
        let token_chain = u16::from_u64(14);
        let origin_info = state::create_origin_info(token_chain, token_address);
        let creator = state::get_wrapped_asset_signer(origin_info);

        let expected_token_name = string::utf8(b"0000000000000000000000000000000000000000000000000000000000000001");

        let token_id = token::create_token_id_raw(
            signer::address_of(&creator),
            string::utf8(b"my name"),
            expected_token_name,
            0, // property_version
        );
        let token = token::withdraw_token(&recipient_signer, token_id, 1);

        let (token_address,
             token_chain,
             symbol,
             name,
             token_id,
             uri,
        ) = transfer_nft::transfer_nft_test(token);

        assert!(token_address == external_address::from_bytes(x"09"), 0);
        assert!(token_chain == u16::from_u64(14), 0);
        assert!(symbol == string32::from_bytes(b"my symbol"), 0);
        assert!(name == string32::from_bytes(b"my name"), 0);
        assert!(token_id == external_address::from_bytes(x"01"), 0);
        assert!(uri == uri::from_bytes(b"http://google.com"), 0);
    }

    // Test that multiple tokens can be minted from the same collection
    #[test(deployer=@deployer, recipient=@0x1234)]
    public fun test_complete_transfer_wrapped_asset_multiple_tokens(deployer: &signer, recipient: address) {
        wormhole::wormhole_test::setup(0);
        nft_bridge::nft_bridge::init_test(deployer);

        let recipient = aptos_framework::account::create_account_for_test(recipient);
        token::opt_in_direct_transfer(&recipient, true);

        let token_id1 = external_address::from_bytes(x"01");
        let token_id2 = external_address::from_bytes(x"02");

        complete_transfer_wrapped_helper(&recipient, token_id1);
        complete_transfer_wrapped_helper(&recipient, token_id2);
    }


    /// Helper function that transfer a token into the recipient address
    fun complete_transfer_wrapped_helper(
        recipient_signer: &signer,
        token_id: ExternalAddress
    ) {
        let recipient = signer::address_of(recipient_signer);
        let recipient_external = external_address::left_pad(&bcs::to_bytes(&recipient));

        token::opt_in_direct_transfer(recipient_signer, true);

        let token_address = external_address::from_bytes(x"09");
        let token_chain = u16::from_u64(14);
        let token_symbol = string32::from_bytes(b"my symbol");
        let token_name =  string32::from_bytes(b"my name");
        let token_uri = uri::from_bytes(b"http://google.com");
        let t = transfer::create(
            token_address,
            token_chain,
            token_symbol,
            token_name,
            token_id,
            token_uri,
            recipient_external,
            u16::from_u64(22) // to chain
        );

        // complete transfer using transfer object above
        complete_transfer::test(&t);

        let origin_info = state::create_origin_info(token_chain, token_address);
        let creator = state::get_wrapped_asset_signer(origin_info);

        let expected_collection_name = string::utf8(b"my name");
        let expected_token_name = wrapped_token_name::render_hex(external_address::get_bytes(&token_id));
        let token_id = token::create_token_id_raw(
            signer::address_of(&creator),
            expected_collection_name,
            expected_token_name,
            0
        );

        assert!(token::balance_of(recipient, token_id) == 1, 0);
    }

    // ------ Native asset tests

    #[test(
        deployer = @deployer,
        creator = @0x654321,
        first_user = @0x123456,
        second_user = @0x121212
    )]
    fun complete_transfer_native_test(
        deployer: &signer,
        creator: address,
        first_user: &signer,
        second_user: &signer
    ) {
        let collection_name = string::utf8(b"my test collection");
        let token_name = string::utf8(b"my test token");
        complete_transfer_native_helper(
            deployer,
            creator,
            first_user,
            second_user,
            collection_name,
            token_name,
            true,
            22
        );
    }

    #[test(
        deployer = @deployer,
        creator = @0x654321,
        first_user = @0x123456,
        second_user = @0x121212
    )]
    #[expected_failure(abort_code = 1, location = nft_bridge::complete_transfer)]
    fun complete_transfer_native_test_incorrect_token_address(
        deployer: &signer,
        creator: address,
        first_user: &signer,
        second_user: &signer
    ) {
        let collection_name = string::utf8(b"my test collection");
        let token_name = string::utf8(b"my test token");
        complete_transfer_native_helper(
            deployer,
            creator,
            first_user,
            second_user,
            collection_name,
            token_name,
            false,
            22
        );
    }

    #[test(
        deployer = @deployer,
        creator = @0x654321,
        first_user = @0x123456,
        second_user = @0x121212
    )]
    #[expected_failure(abort_code = 0, location = nft_bridge::complete_transfer)]
    fun complete_transfer_native_test_incorrect_target_chain(
        deployer: &signer,
        creator: address,
        first_user: &signer,
        second_user: &signer
    ) {
        let collection_name = string::utf8(b"my test collection");
        let token_name = string::utf8(b"my test token");
        complete_transfer_native_helper(
            deployer,
            creator,
            first_user,
            second_user,
            collection_name,
            token_name,
            true,
            21
        );
    }

    /// Helper function for performing a variety of tests that all follow the
    /// same flow.
    ///
    /// This function performs the setup, then creates a collection under
    /// `creator`, then mints a token to `first_user`, then transfers out that
    /// token.
    /// Finally, it transfers the token back in to `second_user`.
    fun complete_transfer_native_helper(
        deployer: &signer,
        creator: address,
        first_user: &signer,
        second_user: &signer,
        collection_name: String,
        token_name: String,
        // if this flag is `true`, the 'token_address' field in the incoming
        // transfer will be one matching the collection hash of the token,
        // otherwise an arbitary address
        use_correct_token_address: bool,
        // this flag determines the target chain of the incoming transfer
        // (only 22 should be accepted)
        target_chain: u64
    ) {
        // ------ Setup
        wormhole::wormhole_test::setup(0);
        nft_bridge::nft_bridge::init_test(deployer);

        aptos_framework::aptos_account::create_account(signer::address_of(first_user));
        token::opt_in_direct_transfer(first_user, true);

        aptos_framework::aptos_account::create_account(signer::address_of(second_user));
        token::opt_in_direct_transfer(second_user, true);

        let creator = account::create_account_for_test(creator);

        // ------ Create a collection under `creator`

        transfer_nft_test::create_collection(&creator, collection_name);

        // ------ Mint a token to `first_user` from the collection we just created

        transfer_nft_test::mint_token_to(
            &creator,
            signer::address_of(first_user),
            collection_name,
            token_name,
            1
        );

        // ------ Construct token_id for follow-up queries

        let token_id = token::create_token_id_raw(
            signer::address_of(&creator),
            collection_name,
            token_name,
            0 // property_version
        );

        let (collection_hash, token_hash) = token_hash::derive(&token_id);

        // ------ Transfer the tokens
        transfer_nft::transfer_nft_entry(
            first_user,
            signer::address_of(&creator),
            collection_name,
            token_name,
            0, // property_version
            3, // recipient chain
            x"0000000000000000000000000000000000000000000000000000000000FAFAFA",
            0
        );

        // ------ Check that `nft_bridge` now holds the token

        assert!(token::balance_of(@nft_bridge, token_id) == 1, 0);

        let expected_token_address: ExternalAddress;
        if (use_correct_token_address) {
            expected_token_address = token_hash::get_collection_external_address(&collection_hash);
        } else {
            expected_token_address = external_address::from_bytes(x"0123");
        };

        let t = transfer::create(
            expected_token_address,
            u16::from_u64(22),
            string32::from_bytes(b"symbol (ignored)"),
            string32::from_bytes(b"name (ignored)"),
            token_hash::get_token_external_address(&token_hash),
            uri::from_bytes(b"uri (ignored)"),
            external_address::from_bytes(std::bcs::to_bytes(&signer::address_of(second_user))),
            u16::from_u64(target_chain)
        );

        // complete transfer using transfer object above
        complete_transfer::test(&t);

        // ------ Check that `second_user` now holds the token
        assert!(token::balance_of(signer::address_of(second_user), token_id) == 1, 0);
    }
}
