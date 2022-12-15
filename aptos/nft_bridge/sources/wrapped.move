module nft_bridge::wrapped {
    use std::vector;
    use std::string;

    use aptos_framework::account;
    use aptos_token::token;

    use wormhole::serialize;
    use wormhole::external_address;

    use token_bridge::string32;

    use nft_bridge::state::{Self, OriginInfo};
    use nft_bridge::transfer::{Self, Transfer};

    friend nft_bridge::complete_transfer;
    friend nft_bridge::transfer_nft;

    #[test_only]
    friend nft_bridge::transfer_nft_test;
    #[test_only]
    friend nft_bridge::wrapped_test;
    #[test_only]
    friend nft_bridge::complete_transfer_test;

    const E_IS_NOT_WRAPPED_ASSET: u64 = 0;

    public(friend) fun create_wrapped_nft_collection(transfer: &Transfer) {
        let token_address = transfer::get_token_address(transfer);
        let token_chain = transfer::get_token_chain(transfer);
        let token_id = transfer::get_token_id(transfer);
        let origin_info = state::create_origin_info(token_chain, token_address, token_id);

        // if the resource account already exists, we don't need do anything
        if (!state::wrapped_asset_signer_exists(origin_info)) {
            let seed = create_seed(&origin_info);
            //create resource account
            let nft_bridge_signer = state::nft_bridge_signer();
            let (new_signer, new_cap) = account::create_resource_account(&nft_bridge_signer, seed);
            state::set_wrapped_asset_signer_capability(origin_info, new_cap);
            init_wrapped_nft(&new_signer, transfer);
        };
    }

    fun init_wrapped_nft(
        creator_signer: &signer,
        transfer: &Transfer,
    ) {
        let name = transfer::get_name(transfer);
        // let symbol = transfer::get_symbol(transfer);
        let description = string::utf8(b"Wormhole wrapped NFT"); // TODO(csongor): what should the description be
        let uri = string::utf8(b""); // TODO(csongor): what should the collection uri be?
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

        let token_address = transfer::get_token_address(transfer);
        let token_chain = transfer::get_token_chain(transfer);
        let token_id = transfer::get_token_id(transfer);
        let origin_info = state::create_origin_info(token_chain, token_address, token_id);

        // update the following two mappings in State
        // 1. (native chain, native address) => wrapped address
        // 2. wrapped address => (native chain, native address)
        state::setup_wrapped(origin_info);
    }

    // public(friend) fun mint<CoinType>(amount: u64): Coin<CoinType> acquires CoinCapabilities {
    //     assert!(state::is_wrapped_asset<CoinType>(), E_IS_NOT_WRAPPED_ASSET);
    //     assert!(exists<CoinCapabilities<CoinType>>(@nft_bridge), E_COIN_CAP_DOES_NOT_EXIST);
    //     let caps = borrow_global<CoinCapabilities<CoinType>>(@nft_bridge);
    //     let mint_cap = &caps.mint_cap;
    //     let coins = coin::mint<CoinType>(amount, mint_cap);
    //     coins
    // }

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
    use std::account::{Self};
    use std::signer;
    use std::string::{Self, String};
    use std::bcs;

    use aptos_token::token::{Self};

    use wormhole::external_address::{Self};
    use wormhole::u16::{Self};
    use wormhole::wormhole::{Self};
    use wormhole::wormhole_test::{Self};

    use token_bridge::string32::{Self};

    use nft_bridge::transfer::{Self};
    use nft_bridge::uri::{Self};
    use nft_bridge::wrapped::{Self};
    use nft_bridge::state::{Self as nft_state};

    #[test(deployer=@deployer)]
    public fun init_worm_and_nft_state(deployer: &signer){
        // init wormhole state
        wormhole_test::setup(0);
        // init nft state
        let (_nft_bridge, signer_cap) = account::create_resource_account(deployer, b"nft_bridge");
        let emitter_cap = wormhole::register_emitter();
        nft_state::init_nft_bridge_state(
            signer_cap,
            emitter_cap
        );
    }

    // test that wrapped NFT collection can be created from transfer object
    #[test(deployer=@deployer)]
    public fun test_create_wrapped_nft_collection(deployer: &signer) {
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
            external_address::from_bytes(x"0000"),
            u16::from_u64(1)
        );
        wrapped::create_wrapped_nft_collection(&t);

        let origin_info = nft_state::create_origin_info(
            token_chain,
            token_address,
            token_id,
        );

        // assert that collection was indeed created
        let my_signer = nft_state::get_wrapped_asset_signer(origin_info);
        assert!(token::check_collection_exists(signer::address_of(&my_signer), string32::to_string(&token_name)), 0);

        // set token metadata
        let token_mut_config = token::create_token_mutability_config(
            &vector[true, true, true, true, true]
        );

        let token_data_id = token::create_tokendata(
            &my_signer,
            string32::to_string(&token_name), // token collection name
            string::utf8(x"01"), // token name
            string::utf8(b"a description"),
            100,
            string::utf8(b"a uri"),
            signer::address_of(&my_signer),
            4,
            1,
            token_mut_config,
            vector<String>[string::utf8(b"TOKEN_BURNABLE_BY_CREATOR")],
            vector<vector<u8>>[bcs::to_bytes<bool>(&true)],
            vector<String>[string::utf8(b"bool")],
        );

        // test mint token using signer
        token::initialize_token_store(&my_signer);
        token::opt_in_direct_transfer(&my_signer, true);

        token::mint_token_to(
            &my_signer,
            signer::address_of(&my_signer),
            token_data_id,
            99 // mint 99 NFTs
        );
    }
}
