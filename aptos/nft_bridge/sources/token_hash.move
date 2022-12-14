/// A pair of 32 byte hashes representing an arbitrary Aptos NFT, to be used in
/// VAAs to refer to NFTs.
module nft_bridge::token_hash {
    use aptos_token::token::{Self, TokenId};
    use std::bcs;
    use std::hash;
    use std::string;
    use std::vector;

    use wormhole::serialize;
    use wormhole::external_address::{Self, ExternalAddress};

    /// Hash of (creator || collection name), which uniquely identifies a
    /// collection on Aptos
    struct CollectionHash has drop, copy, store {
        // 32 bytes
        hash: vector<u8>,
    }

    /// Hash of (creator || collection name || token name || property version), which
    /// uniquely identifies a token on Aptos
    struct TokenHash has drop, copy, store {
        // 32 bytes
        hash: vector<u8>,
    }

    public fun get_collection_external_address(a: &CollectionHash): ExternalAddress {
        external_address::from_bytes(a.hash)
    }

    public fun get_token_external_address(a: &TokenHash): ExternalAddress {
        external_address::from_bytes(a.hash)
    }

    public fun from_external_address(a: ExternalAddress): TokenHash {
       TokenHash { hash: external_address::get_bytes(&a) }
    }

    public fun derive(token_id: &TokenId): (CollectionHash, TokenHash) {
        let ser = vector::empty<u8>();
        let (creator, collection, name, property_version) = token::get_token_id_fields(token_id);
        let creator_bytes = bcs::to_bytes(&creator);
        serialize::serialize_vector(&mut ser, creator_bytes);
        serialize::serialize_vector(&mut ser, *string::bytes(&collection));
        let collection_hash = hash::sha3_256(ser);

        serialize::serialize_vector(&mut ser, *string::bytes(&name));
        serialize::serialize_u64(&mut ser, property_version);
        let token_hash = hash::sha3_256(ser);

        (CollectionHash { hash: collection_hash }, TokenHash { hash: token_hash })
    }

}

#[test_only]
module nft_bridge::token_hash_test {
    // TODO(csongor): test
}
