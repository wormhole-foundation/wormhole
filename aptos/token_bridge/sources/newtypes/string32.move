/// The `string32` module defines the `String32` type which represents UTF8
/// encoded strings that are guaranteed to be 32 bytes long, with 0 padding on
/// the right.
module token_bridge::string32 {

    use std::string::{Self, String};
    use std::option;
    use std::vector;

    use wormhole::cursor::Cursor;
    use wormhole::deserialize;
    use wormhole::serialize;

    const E_STRING_TOO_LONG: u64 = 0;

    /// A `String32` holds a ut8 string which is guaranteed to be 32 bytes long.
    struct String32 has copy, drop, store {
       string: String
    }

    spec String32 {
        invariant string::length(string) == 32;
    }

    /// Right-pads a `String` to a `String32` with 0 bytes.
    /// Aborts if the string is longer than 32 bytes.
    public fun right_pad(s: &String): String32 {
        let length = string::length(s);
        assert!(length <= 32, E_STRING_TOO_LONG);
        let string = *string::bytes(s);
        let zeros = 32 - length;
        while ({
            spec {
                invariant zeros + vector::length(string) == 32;
            };
            zeros > 0
        }) {
            vector::push_back(&mut string, 0);
            zeros = zeros - 1;
        };
        String32 { string: string::utf8(string) }
    }

    /// Internal function to take the first 32 bytes of a byte sequence and
    /// convert to a utf8 `String`.
    /// Takes the longest prefix that's valid utf8 and maximum 32 bytes.
    ///
    /// Even if the input is valid utf8, the result might be shorter than 32
    /// bytes, because the original string might have a multi-byte utf8
    /// character at the 32 byte boundary, which, when split, results in an
    /// invalid code point, so we remove it.
    fun take(bytes: vector<u8>, n: u64): String {
        while (vector::length(&bytes) > n) {
            vector::pop_back(&mut bytes);
        };

        let utf8 = string::try_utf8(bytes);
        while (option::is_none(&utf8)) {
            vector::pop_back(&mut bytes);
            utf8 = string::try_utf8(bytes);
        };
        option::extract(&mut utf8)
    }

    /// Takes the first `n` bytes of a `String`.
    ///
    /// Even if the input string is longer than `n`, the resulting string might
    /// be shorter because the original string might have a multi-byte utf8
    /// character at the byte boundary, which, when split, results in an invalid
    /// code point, so we remove it.
    public fun take_utf8(str: String, n: u64): String {
        take(*string::bytes(&str), n)
    }

    /// Truncates or right-pads a `String` to a `String32`.
    /// Does not abort.
    public fun from_string(s: &String): String32 {
        right_pad(&take(*string::bytes(s), 32))
    }

    /// Truncates or right-pads a byte vector to a `String32`.
    /// Does not abort.
    public fun from_bytes(b: vector<u8>): String32 {
        right_pad(&take(b, 32))
    }

    /// Converts `String32` to `String`, removing trailing 0s.
    public fun to_string(s: &String32): String {
        let String32 { string } = s;
        let bytes = *string::bytes(string);
        // keep dropping the last character while it's 0
        while (!vector::is_empty(&bytes) &&
               *vector::borrow(&bytes, vector::length(&bytes) - 1) == 0
        ) {
            vector::pop_back(&mut bytes);
        };
        string::utf8(bytes)
    }

    /// Converts `String32` to a byte vector of length 32.
    public fun to_bytes(s: &String32): vector<u8> {
        *string::bytes(&s.string)
    }

    public fun deserialize(cur: &mut Cursor<u8>): String32 {
        let bytes = deserialize::deserialize_vector(cur, 32);
        from_bytes(bytes)
    }

    public fun serialize(buf: &mut vector<u8>, e: String32) {
        serialize::serialize_vector(buf, to_bytes(&e))
    }

}

#[test_only]
module token_bridge::string32_test {
    use std::string;
    use std::vector;
    use token_bridge::string32;

    #[test]
    public fun test_right_pad() {
        let result = string32::right_pad(&string::utf8(b"hello"));
        assert!(string32::to_string(&result) == string::utf8(b"hello"), 0)
    }

    #[test]
    #[expected_failure(abort_code = 0, location = token_bridge::string32)]
    public fun test_right_pad_fail() {
        let too_long = string::utf8(b"this string is very very very very very very very very very very very very very very very long");
        string32::right_pad(&too_long);
    }

    #[test]
    public fun test_from_string_short() {
        let result = string32::from_string(&string::utf8(b"hello"));
        assert!(string32::to_string(&result) == string::utf8(b"hello"), 0)
    }

    #[test]
    public fun test_from_string_long() {
        let long = string32::from_string(&string::utf8(b"this string is very very very very very very very very very very very very very very very long"));
        assert!(string32::to_string(&long) == string::utf8(b"this string is very very very ve"), 0)
    }

    #[test]
    public fun test_from_string_weird_utf8() {
        let string = b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
        assert!(vector::length(&string) == 31, 0);
        // append the samaritan letter Alaf, a 3-byte utf8 character the move
        // parser only allows ascii characters unfortunately (the character
        // looks nice)
        vector::append(&mut string, x"e0a080");
        // it's valid utf8
        let string = string::utf8(string);
        // string length is bytes, not characters
        assert!(string::length(&string) == 34, 0);
        let padded = string32::from_string(&string);
        // notice that the e0 byte got dropped at the end
        assert!(string32::to_string(&padded) == string::utf8(b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0)
    }

    #[test]
    public fun test_from_bytes_invalid_utf8() {
        // invalid utf8
        let bytes = x"e0a0";
        let result = string::utf8(b"");
        assert!(string32::to_string(&string32::from_bytes(bytes)) == result, 0)
    }
}
