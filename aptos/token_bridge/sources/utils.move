module token_bridge::utils {
    use 0x1::vector;

    use wormhole::u256::{Self, U256, from_u64};

    const E_VECTOR_TOO_LONG: u64 = 0;

    // pad a vector with zeros on the left so that it is 32 bytes
    public entry fun pad_left_32(input: &vector<u8>): vector<u8>{
        let len = vector::length<u8>(input);
        assert!(len <= 32, E_VECTOR_TOO_LONG);
        let ret = vector::empty<u8>();
        let zeros_remaining = 32 - len;
        while (zeros_remaining > 0){
            vector::push_back<u8>(&mut ret, 0);
            zeros_remaining = zeros_remaining - 1;
        };
        vector::append<u8>(&mut ret, *input);
        ret
    }

    public entry fun normalize_amount(amount: U256, decimals: u8): U256 {
         if (decimals > 8) {
            let n = decimals - 8;
            while (n > 0){
                amount = u256::div(amount, from_u64(10));
                n = n - 1;
            }
         };
         amount
    }

    public entry fun denormalize_amount(amount: U256, decimals: u8): U256{
         if (decimals > 8) {
            let n = decimals - 8;
            while (n > 0){
                amount = u256::mul(amount, from_u64(10));
                n = n - 1;
            }
         };
         amount
    }
}

#[test_only]
module token_bridge::utils_test {
    use aptos_std::type_info::{type_name};
    use std::string;
    use token_bridge::utils::{pad_left_32, normalize_amount, denormalize_amount};
    use wormhole::u256::{from_u64};

    struct MyCoin {}

    #[test]
    fun test_type_name() {
        let name = type_name<MyCoin>();
        assert!(*string::bytes(&name) == b"0x4450040bc7ea55def9182559ceffc0652d88541538b30a43477364f475f4a4ed::utils_test::MyCoin", 0);
    }

    #[test]
    fun test_pad_left_short() {
        let v = x"11";
        let pad_left_v = pad_left_32(&v);
        assert!(pad_left_v==x"0000000000000000000000000000000000000000000000000000000000000011", 0);
    }

    #[test]
    fun test_pad_left_exact() {
        let v = x"5555555555555555555555555555555555555555555555555555555555555555";
        let pad_left_v = pad_left_32(&v);
        assert!(pad_left_v==x"5555555555555555555555555555555555555555555555555555555555555555", 0);
    }

    #[test]
    #[expected_failure(abort_code = 0)]
    fun test_pad_left_long() {
        let v = x"665555555555555555555555555555555555555555555555555555555555555555";
        pad_left_32(&v);
    }

    #[test]
    fun test_normalize_denormalize_amount() {
        let a = from_u64(12345678910111);
        let b = normalize_amount(a, 9);
        let c = denormalize_amount(b, 9);
        assert!(c==from_u64(12345678910110), 0);

        let x = from_u64(12345678910111);
        let y = normalize_amount(x, 5);
        let z = denormalize_amount(y, 5);
        assert!(z==x, 0);
    }
}
