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

        let is_wrapped_asset: bool = token_chain == wormhole_state::get_chain_id();

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
            let nft_bridge = state::nft_bridge_signer();

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
                &nft_bridge,
                collection, // token collection name
                name, // token name
                string::utf8(b""), //empty description
                1, //supply cap 1
                uri::to_string(&uri),
                signer::address_of(&creator),
                0, // royalty_points_denominator
                0, // royalty_points_numerator
                token_mut_config, // see above
                vector<String>[string::utf8(b"TOKEN_BURNABLE_BY_OWNER")],
                vector<vector<u8>>[bcs::to_bytes<bool>(&true)],
                vector<String>[string::utf8(b"bool")],
            );
            token::mint_token_to(
                &creator,
                recipient,
                // token::create_token_data_id(
                //     signer::address_of(&creator),
                //     collection,
                //     name
                // ),
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
    // TODO(csongor): test

    use std::string;
    use nft_bridge::complete_transfer;

    #[test]
    fun render_hex_test() {
        assert!(complete_transfer::render_hex(x"beefcafe") == string::utf8(b"beefcafe"), 0);
    }
}
