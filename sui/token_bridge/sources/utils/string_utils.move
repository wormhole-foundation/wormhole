module token_bridge::string_utils {
    use std::ascii::{Self};
    use std::string::{Self, String};
    use std::vector::{Self};

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

    /// Converts a String32 to an ascii string if possible, otherwise errors
    /// out at `ascii::string(bytes)`. For input strings that contain non-ascii
    /// characters, we will swap the non-ascii character with `?`.
    ///
    /// Note that while the Sui spec limits symbols to only use ascii
    /// characters, the token bridge spec does allow utf8 symbols.
    public fun to_ascii(s: &String): ascii::String {
        let buf = *string::bytes(s);
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
}
