/// 32 byte hash representing an arbitrary Aptos token, to be used in VAAs to
/// refer to coins.
module token_bridge::token_hash {
    use aptos_framework::type_info;
    use std::hash;
    use std::string;

    struct TokenHash has drop, copy, store {
        // 32 bytes
        hash: vector<u8>,
    }

    public fun get_bytes(a: &TokenHash): vector<u8> {
        a.hash
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

    use std::type_info;
    use std::string;

    struct MyCoin {}

    #[test]
    public fun foo() {
        let t = type_info::type_name<MyCoin>();
        assert!(*string::bytes(&t) == b"0x4450040bc7ea55def9182559ceffc0652d88541538b30a43477364f475f4a4ed::token_hash_test::MyCoin", 0)
    }

    #[test]
    public fun test_derive() {
        let t = token_hash::derive<MyCoin>();
        let expected = x"a5839f5bd57edea609a0ea0f8a58df8bf245e24624c1675cf6fa18c569356b1b";
        assert!(token_hash::get_bytes(&t) == expected, 0);
    }
}
