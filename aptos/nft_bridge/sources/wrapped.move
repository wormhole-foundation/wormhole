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
        }
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
    // TODO(csongor): test
}
