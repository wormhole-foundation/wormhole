module token_bridge::utils {
    use wormhole::u256::{Self, U256, from_u64};

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
    use token_bridge::utils::{normalize_amount, denormalize_amount};
    use wormhole::u256::{from_u64};

    struct MyCoin {}

    #[test]
    fun test_type_name() {
        let name = type_name<MyCoin>();
        assert!(*string::bytes(&name) == b"0x4450040bc7ea55def9182559ceffc0652d88541538b30a43477364f475f4a4ed::utils_test::MyCoin", 0);
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
