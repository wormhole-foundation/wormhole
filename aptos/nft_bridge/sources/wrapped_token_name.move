module nft_bridge::wrapped_token_name {
    use std::vector;
    use std::string::{Self, String};

    const E_INVALID_HEX_DIGIT: u64 = 0;
    const E_INVALID_HEX_CHAR: u64 = 1;

    // TODO(csongor): rename this functions maybe

    /// Render a vector as a hex string
    public fun render_hex(bytes: vector<u8>): String {
        let res = vector::empty<u8>();
        vector::reverse(&mut bytes);

        while (!vector::is_empty(&bytes)) {
            let b = vector::pop_back(&mut bytes);
            let l = b >> 4;
            let h = b & 0xF;
            vector::push_back(&mut res, hex_digit(l));
            vector::push_back(&mut res, hex_digit(h));
        };
        string::utf8(res)
    }

    /// Returns the ASCII code for a hex digit (i.e. 0 -> '0', a -> 'a')
    fun hex_digit(d: u8): u8 {
        assert!(d < 16, E_INVALID_HEX_DIGIT);
        if (d < 10) {
           d + 48
        } else {
            d + 87
        }
    }

    public fun parse_hex(s: String): vector<u8> {
        let res = vector::empty<u8>();
        let bytes = *string::bytes(&s);

        while (!vector::is_empty(&bytes)) {
            let h = hex_char(vector::pop_back(&mut bytes));
            let l = hex_char(vector::pop_back(&mut bytes));
            let b = (l << 4) + h;
            vector::push_back(&mut res, b);
        };
        vector::reverse(&mut res);
        res
    }

    // Inverse of hex_digit
    fun hex_char(v: u8): u8 {
        if (48 <= v && v <= 57) {
            v - 48
        } else if (97 <= v && v <= 102) {
            v - 87
        } else {
            assert!(false, E_INVALID_HEX_CHAR);
            0
        }
    }

}

#[test_only]
module nft_bridge::wrapped_token_name_test {
    use std::string;
    use nft_bridge::wrapped_token_name;

    #[test]
    fun render_hex_test() {
        assert!(wrapped_token_name::render_hex(x"beefcafe") == string::utf8(b"beefcafe"), 0);
    }

    #[test]
    fun parse_hex_test() {
        assert!(wrapped_token_name::parse_hex(string::utf8(b"beefcafe")) == x"beefcafe", 0);
    }
}
