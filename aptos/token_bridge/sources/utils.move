module token_bridge::utils {
    use 0x1::hash::{sha3_256};
    use 0x1::type_info::{type_name};
    use 0x1::bcs::{to_bytes};
    use 0x1::vector::{Self};

    //use wormhole::u256::{Self, U256};

    public entry fun hash_type_info<info>(): vector<u8>{
        let type_name = type_name<info>();
        // TODO: let's use keccak256 here?
        let res = sha3_256(to_bytes(&type_name));
        assert!(vector::length(&res)==32, 0);
        res
    }

    //TODO - finish and test normalized and denormalize functions

    // public entry fun normalize_amount(amount: U256, decimals: u8): U256 {
    //     if (decimals > 8) {
    //         amount = u256::div(amount, 10 ** (decimals - 8));
    //     };
    //     amount
    // }

    // public entry fun denormalize_amount(amount: U256, decimals: u8): U256{
    //     if (decimals > 8) {
    //         amount = u256::mul(amount, 10 ** (decimals - 8));
    //     };
    //     amount
    // }
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
