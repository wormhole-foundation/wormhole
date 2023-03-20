/// The `string32` module defines the `String32` type which represents UTF8
/// encoded strings that are guaranteed to be 32 bytes long, with 0 padding on
/// the right.
module token_bridge::string32 {
    use std::ascii::{Self};
    use std::option::{Self};
    use std::string::{Self, String};
    use std::vector::{Self};

    use wormhole::bytes::{Self};
    use wormhole::cursor::{Cursor};

    const E_STRING_TOO_LONG: u64 = 0;

    const QUESTION_MARK: u8 = 63;
    // Recall that UTF-8 characters have variable-length encoding and can have
    // 1, 2, 3, or 4 bytes.
    // The first byte of the 2, 3, and 4-byte UTF-8 characters have a special
    // form indicating how many more bytes follow in the same character
    // representation. Specifically, it can have the forms
    //  - 110xxxxx // 11000000 is 192 (base 10)
    //  - 1110xxxx // 11100000 is 224 (base 10)
    //  - or 11110xxx // 11110000 is 240 (base 10)
    //
    // We can tell the length the a hex UTF-8 character in bytes by looking
    // at the first byte and counting the leading 1's, or alternatively
    // seeing whether it falls in the range
    // [11000000, 11100000) or [11100000, 11110000) or [11110000, 11111111],
    //
    // The following constants demarcate those ranges and are used in the
    // string32::to_ascii function.
    const UTF8_LENGTH_2_FIRST_BYTE_LOWER_BOUND: u8 = 192;
    const UTF8_LENGTH_3_FIRST_BYTE_LOWER_BOUND: u8 = 224;
    const UTF8_LENGTH_4_FIRST_BYTE_LOWER_BOUND: u8 = 240;

    /// A `String32` holds a ut8 string which is guaranteed to be 32 bytes long.
    struct String32 has copy, drop, store {
       data: String
    }

    spec String32 {
        invariant string::length(string) == 32;
    }

    /// Right-pads a `String` to a `String32` with 0 bytes.
    /// Aborts if the string is longer than 32 bytes.
    public fun right_pad(s: &String): String32 {
        let length = string::length(s);
        assert!(length <= 32, E_STRING_TOO_LONG);
        let buf = *string::bytes(s);
        let zeros = 32 - length;
        while ({
            spec {
                invariant zeros + vector::length(buf) == 32;
            };
            zeros > 0
        }) {
            vector::push_back(&mut buf, 0);
            zeros = zeros - 1;
        };
        String32 { data: string::utf8(buf) }
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
    public fun take_utf8(data: String, n: u64): String {
        take(*string::bytes(&data), n)
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
    public fun to_utf8(s: &String32): String {
        let String32 { data } = s;
        let buf = *string::bytes(data);
        // Keep dropping the last character while it's 0.
        while (
            !vector::is_empty(&buf) &&
            *vector::borrow(&buf, vector::length(&buf) - 1) == 0
        ) {
            vector::pop_back(&mut buf);
        };
        string::utf8(buf)
    }

    /// Converts a String32 to an ascii string if possible, otherwise errors
    /// out at `ascii::string(bytes)`. For input strings that contain non-ascii
    /// characters, we will swap the non-ascii character with `?`.
    ///
    /// Note that while the Sui spec limits symbols to only use ascii
    /// characters, the token bridge spec does allow utf8 symbols.
    public fun to_ascii(s: &String32): ascii::String {
        let String32 { data } = s;
        let buf = *string::bytes(data);
        // keep dropping the last character while it's 0
        while (
            !vector::is_empty(&buf) &&
            *vector::borrow(&buf, vector::length(&buf) - 1) == 0
        ) {
            vector::pop_back(&mut buf);
        };

        // Run through `buf` to convert any non-ascii character to `?`.
        let asciified = vector::empty();
        let (i, n) = (0, vector::length(&buf));
        while (i < n) {
            let b = *vector::borrow(&buf, i);
            // If it is a valid ascii character, keep it.
            if (ascii::is_valid_char(b)) {
                vector::push_back(&mut asciified, b);
                i = i + 1;
            } else {
                // Since UTF-8 characters have variable-length encoding (they are
                // represented using 1-4 bytes, unlike ASCII characters, which
                // are represented using 1 byte), we don't want to transform
                // every byte in a UTF-8 string that does not represent an ASCII
                // character to the question mark symbol "?". This would result
                // in having too many "?" symbols.
                //
                // Instead, we want a single "?" for each character. Note that
                // the 1-byte UTF-8 characters correspond to valid ASCII
                // characters and have the form 0xxxxxxx.
                // The 2, 3, and 4-byte UTF-8 characters have first byte equal
                // to:
                //  - 110xxxxx // 192
                //  - 1110xxxx // 224
                //  - or 11110xxx // 240
                //
                // and remaining bytes of the form:
                // - 10xxxxxx
                //
                // To ensure a one-to-one mapping of a multi-byte UTF-8 character
                // to a "?", we detect the first byte of a new UTF-8 character
                // in a multi-byte representation by checking if it is
                // >= 11000000 (base 2) or 192 (base 10) and convert it to a "?"
                // and skip the remaining bytes in the same representation.
                //
                //
                // Reference: https://en.wikipedia.org/wiki/UTF-8
                if (b >= UTF8_LENGTH_2_FIRST_BYTE_LOWER_BOUND){
                    vector::push_back(&mut asciified, QUESTION_MARK);
                    if (b >= UTF8_LENGTH_4_FIRST_BYTE_LOWER_BOUND){
                        // The UTF-8 char has a 4-byte hex representation.
                        i = i + 4;
                    } else if (b >= UTF8_LENGTH_3_FIRST_BYTE_LOWER_BOUND){
                        // The UTF-8 char has a 3-byte hex representation.
                        i = i + 3;
                    } else {
                        // The UTF-8 char has a 2-byte hex representation.
                        i = i + 2;
                    }
                }
            };
        };
        ascii::string(asciified)
    }

    /// Converts `String32` to a byte vector of length 32.
    public fun to_bytes(s: &String32): vector<u8> {
        *string::bytes(&s.data)
    }

    public fun deserialize(cur: &mut Cursor<u8>): String32 {
        let bytes = bytes::take_bytes(cur, 32);
        from_bytes(bytes)
    }

    public fun serialize(buf: &mut vector<u8>, e: String32) {
        vector::append(buf, to_bytes(&e))
    }

}

#[test_only]
module token_bridge::string32_test {
    use std::string;
    //use std::ascii;
    use std::vector;
    use token_bridge::string32;

    #[test]
    public fun test_right_pad() {
        let result = string32::right_pad(&string::utf8(b"hello"));
        assert!(string32::to_utf8(&result) == string::utf8(b"hello"), 0)
    }

    #[test]
    #[expected_failure(abort_code = string32::E_STRING_TOO_LONG)]
    public fun test_right_pad_fail() {
        let too_long = string::utf8(
            b"this string is very very very very very very very very very very very very very very very long");
        string32::right_pad(&too_long);
    }

    #[test]
    public fun test_from_string_short() {
        let result = string32::from_string(&string::utf8(b"hello"));
        assert!(string32::to_utf8(&result) == string::utf8(b"hello"), 0)
    }

    #[test]
    public fun test_from_string_long() {
        let long = string32::from_string(&string::utf8(
            b"this string is very very very very very very very very very very very very very very very long"));
        assert!(string32::to_utf8(&long) == string::utf8(
            b"this string is very very very ve"), 0)
    }

    #[test]
    public fun test_from_string_weird_utf8() {
        let string = b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
        assert!(vector::length(&string) == 31, 0);
        // Append the samaritan letter Alaf, a 3-byte utf8 character the move
        // parser only allows ascii characters unfortunately (the character
        // looks nice).
        vector::append(&mut string, x"e0a080");
        // It's valid utf8.
        let string = string::utf8(string);
        // String length is bytes, not characters.
        assert!(string::length(&string) == 34, 0);
        let padded = string32::from_string(&string);
        // Notice that the e0 byte got dropped at the end.
        assert!(string32::to_utf8(&padded) ==
            string::utf8(b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0)
    }

    #[test]
    public fun test_from_bytes_invalid_utf8() {
        // invalid utf8
        let bytes = x"e0a0";
        let result = string::utf8(b"");
        assert!(string32::to_utf8(&string32::from_bytes(bytes)) == result, 0)
    }

    #[test]
    /// In this test, we check that string32::to_ascii replaces non-ASCII
    /// characters in a utf8 hex bytestring with "?", and leaves valid ASCII
    /// characters untouched.
    /// Note that that UTF-8 characters are often represented using multiple
    /// bytes. We use this test as an opportunity to see if to_ascii works on
    /// multi-byte UTF-8 characters and UTF-8 strings containing them.
    public fun test_to_ascii() {
        use std::ascii::Self;
        // UTF-8 character with 2-byte representation.
        let utf8_bytes = x"C2B0";
        let stringified = string32::from_bytes(utf8_bytes);
        let ascii_string = string32::to_ascii(&stringified);
        // The 2-byte hex UTF-8 character is transformed to a singular "?".
        assert!(ascii::into_bytes(ascii_string)==b"?", 0);

        // UTF-8 character with 4-byte representation.
        utf8_bytes = x"F0908D88";
        let stringified = string32::from_bytes(utf8_bytes);
        let ascii_string = string32::to_ascii(&stringified);
        // The 4-byte hex UTF-8 character is transformed to a singular "?".
        assert!(ascii::into_bytes(ascii_string)==b"?", 0);

        // UTF-8 characters with variable number of bytes.
        utf8_bytes = x"C2B0F0908D88C2B0F0908D88F0908D88C2B0";
        let stringified = string32::from_bytes(utf8_bytes);
        let ascii_string = string32::to_ascii(&stringified);
        // The 4-byte hex UTF-8 character is transformed into "??????".
        assert!(ascii::into_bytes(ascii_string)==b"??????", 0);

        // UTF-8 characters with variable number of bytes and valid ASCII
        // "~" characters mixed in.
        utf8_bytes = x"C2B07EF0908D88C2B0F0908D887EF0908D887EC2B0";
        let stringified = string32::from_bytes(utf8_bytes);
        let ascii_string = string32::to_ascii(&stringified);
        // The 4-byte hex UTF-8 character is transformed into "?~???~?~?".
        // Note that the valid ASCII characters remain untouched.
        assert!(ascii::into_bytes(ascii_string)==b"?~???~?~?", 0);
    }
}
