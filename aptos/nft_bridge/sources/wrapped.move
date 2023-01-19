module nft_bridge::wrapped {
    use std::signer;
    use std::vector;
    use std::bcs;
    use std::string::{Self, String};

    use aptos_framework::account;
    use aptos_token::token;

    use wormhole::serialize;
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::string32::{Self, String32};

    use nft_bridge::state::{Self, OriginInfo};
    use nft_bridge::transfer::{Self, Transfer};
    use nft_bridge::uri::{Self, URI};
    use nft_bridge::wrapped_token_name;

    friend nft_bridge::complete_transfer;
    friend nft_bridge::transfer_nft;

    #[test_only]
    friend nft_bridge::transfer_nft_test;
    #[test_only]
    friend nft_bridge::wrapped_test;
    #[test_only]
    friend nft_bridge::complete_transfer_test;

    const E_IS_NOT_WRAPPED_ASSET: u64 = 0;

    /// Create a new collection from the transfer data if it doesn't already exist.
    /// The collection will be created into a resource account, whose signer is
    /// returned.
    public(friend) fun create_or_find_wrapped_nft_collection(transfer: &Transfer): (signer, String32) {
        let token_address = transfer::get_token_address(transfer);
        let token_chain = transfer::get_token_chain(transfer);
        let origin_info = state::create_origin_info(token_chain, token_address);

        let original_name = transfer::get_name(transfer);
        let original_symbol = transfer::get_symbol(transfer);

        let name: String32;
        let symbol: String32;
        if (state::is_unified_solana_collection(origin_info)) {
            name = string32::from_bytes(b"Wormhole Bridged Solana-NFT");
            symbol = string32::from_bytes(b"WORMSPLNFT");
            state::set_spl_cache(transfer::get_token_id(transfer), original_name, original_symbol);
        } else {
            name = original_name;
            symbol = original_symbol;
        };

        // if the resource account already exists, we don't need do anything
        if (!state::wrapped_asset_signer_exists(origin_info)) {
            let seed = create_seed(&origin_info);
            //create resource account
            let nft_bridge_signer = state::nft_bridge_signer();
            let (new_signer, new_cap) = account::create_resource_account(&nft_bridge_signer, seed);
            state::set_wrapped_asset_info(origin_info, new_cap, symbol);
            init_wrapped_nft(&new_signer, name, origin_info);
            (new_signer, name)
        } else {
            (state::get_wrapped_asset_signer(origin_info), name)
        }
    }

    fun init_wrapped_nft(
        creator_signer: &signer,
        name: String32,
        origin_info: OriginInfo,
    ) {
        let description = string::utf8(b"NFT transferred through Wormhole");

        let uri = string::utf8(b"http://portalbridge.com");
        // unbounded
        let maximum = 0;
        // allow all fields to be mutated, in case needed in the future
        let mutability_config = vector[true, true, true];

        token::create_collection(
            creator_signer,
            string32::to_string(&name),
            description,
            uri,
            maximum,
            mutability_config,
        );

        state::setup_wrapped(origin_info);
    }

    public(friend) fun mint_to(
        creator: &signer,
        recipient: address,
        collection: String,
        token_external_id: &ExternalAddress,
        uri: URI
    ) {
        // for the token name, we put the hex of the token id.
        // TODO: is there anything better we could do? maybe render as
        // decimal, as most chains use decimal numbers for token ids.
        let name = wrapped_token_name::render_hex(external_address::get_bytes(token_external_id));

        // set token data, including property keys (set token burnability to true)
        let token_mut_config = token::create_token_mutability_config(
            &vector[
                true, // TOKEN_MAX_MUTABLE
                true, // TOKEN_URI_MUTABLE
                true, // TOKEN_ROYALTY_MUTABLE_IND
                true, // TOKEN_DESCRIPTION_MUTABLE_IND
                true  // TOKEN_PROPERTY_MUTABLE_IND
            ]
        );

        // NOTE: Whether a token can be burned at all, burned by owner, or
        // burned by creator is set in the property keys field when calling
        // token::create_tokendata. We only allow `burn_by_creator` to avoid an
        // edge case whereby a user burns a wrapped token and can no longer
        // bridge it back to the origin chain.

        let token_data_id = token::create_tokendata(
            creator,
            collection, // token collection name
            name, // token name
            string::utf8(b""), //empty description
            1, //supply cap 1
            uri::to_string(&uri),
            signer::address_of(creator),
            0, // royalty_points_denominator
            0, // royalty_points_numerator
            token_mut_config, // see above
            // the following three arguments declare that
            // TOKEN_BURNABLE_BY_CREATOR (of type bool) should be set to true
            // see NOTE above
            vector<String>[string::utf8(b"TOKEN_BURNABLE_BY_CREATOR")],
            vector<vector<u8>>[bcs::to_bytes<bool>(&true)],
            vector<String>[string::utf8(b"bool")],
        );
        token::mint_token_to(
            creator,
            recipient,
            token_data_id,
            1
        );
    }

    /// Derive the generation seed for the resource account from
    /// (token chain (2 bytes) || token address (32 bytes)).
    fun create_seed(origin_info: &OriginInfo): vector<u8> {
        let token_chain = state::get_origin_info_token_chain(origin_info);
        let token_address = state::get_origin_info_token_address(origin_info);
        let seed = vector::empty<u8>();
        serialize::serialize_u16(&mut seed, token_chain);
        external_address::serialize(&mut seed, token_address);
        seed
    }

}

#[test_only]
module nft_bridge::wrapped_test {
    use std::signer;
    use std::string::String;

    use aptos_token::token;

    use wormhole::external_address;
    use wormhole::u16;

    use token_bridge::string32;

    use nft_bridge::transfer;
    use nft_bridge::uri;
    use nft_bridge::wrapped;

    /// Creates a test NFT collection
    public fun create_wrapped_nft_collection(recipient: address, collection_name: String): signer {
        let token_address = external_address::from_bytes(x"00");
        let token_chain = u16::from_u64(14);
        let token_id = external_address::from_bytes(x"01");
        let token_symbol = string32::from_bytes(b"collection symbol");
        let token_name =  string32::from_string(&collection_name);
        let uri = uri::from_bytes(b"http://netscape-navigator.it");
        let t = transfer::create(
            token_address,
            token_chain,
            token_symbol,
            token_name,
            token_id,
            uri,
            external_address::from_bytes(x"0000"),
            u16::from_u64(1) // target chain
        );
        let (creator, _) = wrapped::create_or_find_wrapped_nft_collection(&t);

        // assert that collection was indeed created
        assert!(token::check_collection_exists(signer::address_of(&creator), string32::to_string(&token_name)), 0);

        wrapped::mint_to(&creator, recipient, string32::to_string(&token_name), &token_id, uri);

        creator
    }
}
