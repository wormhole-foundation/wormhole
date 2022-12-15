module nft_bridge::complete_transfer {
    use std::signer;
    use std::vector;
    use std::bcs;
    use std::string::{Self, String};
    use aptos_std::from_bcs;
    use aptos_token::token;

    use nft_bridge::vaa;
    use nft_bridge::transfer::{Self, Transfer};
    use nft_bridge::state;
    use nft_bridge::wrapped;
    use nft_bridge::token_hash;
    use nft_bridge::uri;

    use token_bridge::string32;

    use wormhole::state as wormhole_state;
    use wormhole::external_address;

    const E_INVALID_TARGET: u64 = 0;

    public fun submit_vaa(vaa: vector<u8>): Transfer {
        let vaa = vaa::parse_verify_and_replay_protect(vaa);
        let transfer = transfer::parse(wormhole::vaa::destroy(vaa));
        complete_transfer(&transfer);
        transfer
    }

    public entry fun submit_vaa_entry(vaa: vector<u8>) {
        submit_vaa(vaa);
    }

    /// Submits the complete transfer VAA and registers the NFT for the fee
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
        let token_address = transfer::get_token_address(transfer);
        let token_id = transfer::get_token_id(transfer);
        let uri = transfer::get_uri(transfer);
        let origin_info = state::create_origin_info(token_chain, token_address, token_id);

        let recipient = from_bcs::to_address(external_address::get_bytes(&transfer::get_to(transfer)));

        let is_wrapped_asset: bool = token_chain != wormhole_state::get_chain_id();
        if (is_wrapped_asset) {
            wrapped::create_wrapped_nft_collection(transfer);
            let collection = string32::to_string(&transfer::get_name(transfer));
            // for the token name, we put the hex of the token id.
            // TODO: is there anything better we could do? maybe render as
            // decimal, as most chains use decimal numbers for token ids.
            let name = render_hex(external_address::get_bytes(&token_id));
            let creator = state::get_wrapped_asset_signer(origin_info);
            // set token data, including property keys (set token burnability to true)
            //token:address_of
            //let nft_bridge = state::nft_bridge_signer();
            let creator_signer = state::get_wrapped_asset_signer(origin_info);

            let token_mut_config = token::create_token_mutability_config(
                &vector[
                    true, // TOKEN_MAX_MUTABLE
                    true, // TOKEN_URI_MUTABLE
                    true, // TOKEN_ROYALTY_MUTABLE_IND
                    true, // TOKEN_DESCRIPTION_MUTABLE_IND
                    true  // TOKEN_PROPERTY_MUTABLE_IND
                ]
            );
            let token_data_id = token::create_tokendata(
                &creator_signer,
                collection, // token collection name
                name, // token name
                string::utf8(b""), //empty description
                1, //supply cap 1
                uri::to_string(&uri),
                signer::address_of(&creator),
                0, // royalty_points_denominator
                0, // royalty_points_numerator
                token_mut_config, // see above
                vector<String>[string::utf8(b"TOKEN_BURNABLE_BY_CREATOR")],
                vector<vector<u8>>[bcs::to_bytes<bool>(&true)],
                vector<String>[string::utf8(b"bool")],
            );
            token::mint_token_to(
                &creator,
                recipient,
                token_data_id,
                1
            );
        } else {
            // not native, we must have seen this token before
            let nft_bridge = state::nft_bridge_signer();
            let token_hash = token_hash::from_external_address(token_id);
            let token_id = state::get_native_asset_info(token_hash);

            token::transfer(&nft_bridge, token_id, recipient, 1);
        };
    }

    /// Render a vector as a hex string
    public fun render_hex(bytes: vector<u8>): String {
        let res = vector::empty<u8>();
        vector::reverse(&mut bytes);

        while (!vector::is_empty(&bytes)) {
            let b = vector::pop_back(&mut bytes);
            let l = ((b >> 4) as u8);
            let h = ((b & 0xF) as u8);
            vector::push_back(&mut res, hex_digit(l));
            vector::push_back(&mut res, hex_digit(h));
        };
        string::utf8(res)
    }

    fun hex_digit(d: u8): u8 {
        assert!(d < 16, 0);
        if (d < 10) {
           d + 48
        } else {
            d + 87
        }
    }

}

#[test_only]
module nft_bridge::complete_transfer_test {
    use aptos_token::token::{Self};

    use std::string::{Self};

    use wormhole::external_address::{Self};
    use wormhole::u16::{Self};

    use token_bridge::string32::{Self};

    use nft_bridge::transfer::{Self};
    use nft_bridge::uri::{Self};
    use nft_bridge::wrapped_test::{init_worm_and_nft_state};
    use nft_bridge::complete_transfer::{Self};
    use nft_bridge::transfer_nft_test::{test_transfer_native_nft};
    use nft_bridge::token_hash::{Self};

    #[test]
    fun render_hex_test() {
        assert!(complete_transfer::render_hex(x"beefcafe") == string::utf8(b"beefcafe"), 0);
    }

    // test that complete_transfer for wrapped token works
    #[test(deployer=@deployer)]
    public fun test_complete_transfer_wrapped_asset(deployer: &signer) {
        init_worm_and_nft_state(deployer);
        let token_address = external_address::from_bytes(x"0000");
        let token_chain = u16::from_u64(14);
        let token_id = external_address::from_bytes(x"0001");
        let token_symbol = string32::from_bytes(x"aa");
        let token_name =  string32::from_bytes(x"aa");
        let t = transfer::create(
            token_address, // token address
            token_chain, // token chain
            token_symbol, // symbol
            token_name, // name
            token_id, // token id
            uri::from_bytes(x"0000aa"),
            external_address::from_bytes(x"277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b"), // to
            u16::from_u64(22) // to chain
        );
        // have recipient (in our case deployer) register for a token store and enable direct deposit
        token::initialize_token_store(deployer);
        token::opt_in_direct_transfer(deployer, true);

        // complete transfer using transfer object above
        complete_transfer::test(&t);
    }

    // create native aptos nft, use wormhole to transfer it to a native address (the nft gets
    // locked in nft_bridge), and finally call complete_transfer to complete the loop and transfer
    // nft to intended recipient
    #[test(deployer=@deployer, recipient=@0x123456)]
    public fun test_complete_transfer_native_token_full_loop(deployer: &signer, recipient: &signer) {
        // Transfer Aptos-native NFT to a recipient on Aptos
        // In effect, this gives the nft to nft_bridge, where it is locked up until complete_transfer is called
        let token_id = test_transfer_native_nft(deployer, recipient); // id of token that was transferred

        // do some hacking and get what the token id should be
        let (_collection_hash, token_hash) = token_hash::derive(&token_id);
        let token_hash_bytes = token_hash::get_token_hash_bytes(&token_hash);

        // token info to be passed into complete_transfer
        // in a production setting, this would be parsed from a valid transfer nft VAA
        let token_address = external_address::from_bytes(x"0000"); // arbitrarily set for now
        let token_chain = u16::from_u64(22);
        let token_id = external_address::from_bytes(token_hash_bytes); // arbitrarily set for now
        let token_symbol = string32::from_bytes(x"bb");
        let token_name = string32::from_bytes(b"beef token 1");
        let token_uri = b"beef.com/token_1";

        let t = transfer::create(
            token_address, // token address
            token_chain, // token chain
            token_symbol, // symbol
            token_name, // name
            token_id, // token id
            uri::from_bytes(token_uri), // uri
            external_address::from_bytes(x"277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b"), // to
            u16::from_u64(22) // to chain
        );

        // have recipient (in our case deployer) register for a token store and enable direct deposit
        token::initialize_token_store(deployer);
        token::opt_in_direct_transfer(deployer, true);

        // complete transfer using transfer object above
        complete_transfer::test(&t);
    }

    // TODO - failure test cases
}