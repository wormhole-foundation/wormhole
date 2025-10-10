pub const fn validate_bs58(s: &str) -> &str {
    let b = s.as_bytes();
    if b.len() != 44 {
        panic!("invalid pubkey length (need to be 44 characters)");
    }

    let mut i = 0;
    while i < b.len() {
        let c = b[i];

        #[allow(non_snake_case)]
        let is_A_to_Z = c >= b'A' && c <= b'Z';
        #[allow(non_snake_case)]
        let is_O_or_I = c == b'O' || c == b'I';
        let is_a_to_z = c >= b'a' && c <= b'z';
        let is_l = c == b'l';
        let is_1_to_9 = c >= b'1' && c <= b'9';

        if (is_A_to_Z || is_a_to_z || is_1_to_9) && !is_O_or_I && !is_l {
            i += 1;
        } else {
            panic!("invalid character in base58 string")
        }
    }

    s
}

#[macro_export]
macro_rules! env_pubkey {
    ($name: literal) => {
        const_crypto::bs58::decode_pubkey(crate::validate_bs58(env!($name)))
    };
}
