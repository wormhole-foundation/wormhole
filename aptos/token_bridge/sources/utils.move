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

    // pad a vector with zeros on the left so that it is 32 bytes
    public entry fun pad_left_32(input: &vector<u8>): vector<u8>{
        let len = vector::length<u8>(input);
        assert!(len <= 32, 0);
        let ret = vector::empty<u8>();
        let i = 0;
        while (i < 32 - len){
            vector::append<u8>(&mut ret, x"00");
            i = i+1;
        };
        vector::append<u8>(&mut ret, *input);
        ret
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
        use token_bridge::utils::{pad_left_32};

        struct MyCoin has key{}

        #[test]
        fun test_utils() {
            let name = type_name<MyCoin>();
            assert!(*string::bytes(&name) == b"0x4450040bc7ea55def9182559ceffc0652d88541538b30a43477364f475f4a4ed::utils_test::MyCoin", 0);
            let v = x"11";
            let pad_left_v = pad_left_32(&v);
            assert!(pad_left_v==x"0000000000000000000000000000000000000000000000000000000000000011", 0);
        }
    }
