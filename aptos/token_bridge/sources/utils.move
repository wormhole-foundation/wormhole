module token_bridge::utils {
    use 0x1::hash::{sha3_256};
    use 0x1::type_info::{type_name};
    use 0x1::bcs::{to_bytes};
    use 0x1::vector::{Self};

    public entry fun hash_type_info<info>(): vector<u8>{
        let type_name = type_name<info>();
        // TODO: let's use keccak256 here?
        let res = sha3_256(to_bytes(&type_name));
        assert!(vector::length(&res)==32, 0);
        res
    }
}

    #[test_only]
    module token_bridge::utils_test {
        use aptos_std::type_info::{type_name};
        use std::string;

        struct MyCoin has key{}

        #[test]
        fun test_utils() {
            let name = type_name<MyCoin>();
            assert!(*string::bytes(&name) == b"0x4450040bc7ea55def9182559ceffc0652d88541538b30a43477364f475f4a4ed::utils_test::MyCoin", 0);
        }
    }
