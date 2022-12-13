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

    /// Hash of (creator || hash(collection name)), which uniquely identifies a
    /// collection on Aptos
    struct CollectionHash has drop, copy, store {
        // 32 bytes
        hash: vector<u8>,
    }

    /// Hash of (creator || hash(collection name) || hash(token name) || property version), which
    /// uniquely identifies a token on Aptos
    struct TokenHash has drop, copy, store {
        // 32 bytes
        hash: vector<u8>,
    }

    #[test_only]
    public fun get_collection_hash_bytes(x: &CollectionHash): vector<u8>{
        return x.hash
    }

    #[test_only]
    public fun get_token_hash_bytes(x: &TokenHash): vector<u8>{
        return x.hash
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
        // we hash all variable length fields (that is, collection and name)
        let (creator, collection, name, property_version) = token::get_token_id_fields(token_id);
        let creator_bytes = bcs::to_bytes(&creator);
        serialize::serialize_vector(&mut ser, creator_bytes);
        serialize::serialize_vector(&mut ser, hash::sha3_256(*string::bytes(&collection)));
        let collection_hash = hash::sha3_256(ser);

        serialize::serialize_vector(&mut ser, hash::sha3_256(*string::bytes(&name)));
        serialize::serialize_u64(&mut ser, property_version);
        let token_hash = hash::sha3_256(ser);

        (CollectionHash { hash: collection_hash }, TokenHash { hash: token_hash })
    }

}

#[test_only]
module nft_bridge::token_hash_test {
    use std::string;

    use aptos_token::token;
    use nft_bridge::token_hash;

    #[test(creator = @0x1234)]
    public fun test_derive(creator: address) {
        let token_id = token::create_token_id_raw(
            creator,
            string::utf8(b"my collection"),
            string::utf8(b"my token"),
            0
        );
        let (collection_hash, token_hash) = token_hash::derive(&token_id);
        let collection_hash = token_hash::get_collection_hash_bytes(&collection_hash);
        let token_hash = token_hash::get_token_hash_bytes(&token_hash);
        assert!(collection_hash == x"18905beccb7e5a0f17d22e6773bd94886535fa39f4c28841a752c97a52c5eb46", 0);
        assert!(token_hash == x"54ea5951232ad17f3dbb133964eb0463605e0e35dacd856a1881090d7f0218fe", 0);
    }

    // this test ensures that variable length fields can't be reshuffled to
    // cause a collision
    #[test(creator = @0x1234)]
    public fun test_derive_no_rearrange(creator: address) {
        let token_id_1 = token::create_token_id_raw(
            creator,
            string::utf8(b"my collection"),
            string::utf8(b"my token"),
            0
        );

        let token_id_2 = token::create_token_id_raw(
            creator,
            string::utf8(b"my collectionmy"),
            string::utf8(b" token"),
            0
        );
        let (collection_hash_1, token_hash_1) = token_hash::derive(&token_id_1);
        let (collection_hash_2, token_hash_2) = token_hash::derive(&token_id_2);

        let collection_hash_1 = token_hash::get_collection_hash_bytes(&collection_hash_1);
        let token_hash_1 = token_hash::get_token_hash_bytes(&token_hash_1);
        let collection_hash_2 = token_hash::get_collection_hash_bytes(&collection_hash_2);
        let token_hash_2 = token_hash::get_token_hash_bytes(&token_hash_2);

        assert!(collection_hash_1 != collection_hash_2, 0);
        assert!(token_hash_1 != token_hash_2, 0);
    }
}
