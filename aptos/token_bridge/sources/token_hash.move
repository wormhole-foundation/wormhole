/// 32 byte hash representing an arbitrary Aptos token, to be used in VAAs to
/// refer to coins.
module token_bridge::token_hash {
    use aptos_framework::type_info;
    use std::hash;
    use std::string;

    use wormhole::external_address::{Self, ExternalAddress};

    struct TokenHash has drop, copy, store {
        // 32 bytes
        hash: vector<u8>,
    }

    public fun get_external_address(a: &TokenHash): ExternalAddress {
        external_address::from_bytes(a.hash)
    }

    /// Get the 32 token address of an arbitary CoinType
    public fun derive<CoinType>(): TokenHash {
        let type_name = type_info::type_name<CoinType>();
        let hash = hash::sha3_256(*string::bytes(&type_name));
        TokenHash { hash }
    }

}

#[test_only]
module token_bridge::token_hash_test {
    use token_bridge::token_hash;
    use wormhole::external_address;
    use wrapped_coin::coin;

    use std::type_info;
    use std::string;

    struct MyCoin {}

    #[test]
    public fun test_type_name() {
        let t = type_info::type_name<MyCoin>();
        assert!(*string::bytes(&t) == b"0x84a5f374d29fc77e370014dce4fd6a55b58ad608de8074b0be5571701724da31::token_hash_test::MyCoin", 0)
    }

    #[test]
    public fun test_derive() {
        let t = token_hash::derive<MyCoin>();
        let expected = x"4f69c5d0be57aee780277b1179e4833d61f0563869145e971d24a4e49fcd9302";
        assert!(token_hash::get_external_address(&t) == external_address::from_bytes(expected), 0);
    }

    #[test]
    public fun test_type_name_T() {
        let t = type_info::type_name<coin::T>();
        assert!(*string::bytes(&t) == b"0xf4f53cc591e5190eddbc43940746e2b5deea6e0e1562b2bba765d488504842c7::coin::T", 0)
    }

    #[test]
    public fun test_derive_T() {
        let t = token_hash::derive<coin::T>();
        let expected = x"f0dcbf26a2d59b2196630ed6d5fb5c5bc4fd33996c9f31f19d29389d0c8e7ec2";
        assert!(token_hash::get_external_address(&t) == external_address::from_bytes(expected), 0);
    }
}
