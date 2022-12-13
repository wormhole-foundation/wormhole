/// The `uri` module defines the `URI` type which represents UTF8 encoded
/// strings that are at most 200 characters long, used for representing the URI
/// field of an NFT. Since these strings are not fixed length, their binary
/// encoding includes a length byte prefix.
module nft_bridge::uri {

    use std::string::{Self, String};
    use std::option;
    use std::vector;

    use wormhole::cursor::Cursor;
    use wormhole::deserialize;
    use wormhole::serialize;

    const MAX_LENGTH: u64 = 200;

    const E_URI_TOO_LONG: u64 = 0;

    /// A `URI` holds a ut8 string which is guaranteed to be at most 200 characters long
    struct URI has copy, drop, store {
       string: String
    }

    spec URI {
        invariant string::length(string) <= MAX_LENGTH;
    }

    /// Truncates a string to a URI.
    /// Does not abort.
    public fun from_string(s: &String): URI {
        from_bytes(*string::bytes(s))
    }

    /// Truncates a byte vector to a URI.
    /// Does not abort.
    public fun from_bytes(b: vector<u8>): URI {
        URI { string: take(b, MAX_LENGTH) }
    }

    /// Converts `URI` to `String`
    public fun to_string(u: &URI): String {
        u.string
    }

    /// Converts `String32` to a byte vector of length 32.
    public fun to_bytes(u: &URI): vector<u8> {
        *string::bytes(&u.string)
    }

    public fun deserialize(cur: &mut Cursor<u8>): URI {
        let len = (deserialize::deserialize_u8(cur) as u64);
        assert!(len <= MAX_LENGTH, E_URI_TOO_LONG);
        let bytes = deserialize::deserialize_vector(cur, len);
        from_bytes(bytes)
    }

    public fun serialize(buf: &mut vector<u8>, e: URI) {
        let bytes = to_bytes(&e);
        serialize::serialize_u8(buf, (vector::length(&bytes) as u8));
        serialize::serialize_vector(buf, to_bytes(&e))
    }


    /// Internal function to take the first `n` bytes of a byte sequence and
    /// convert to a utf8 `String`.
    /// Takes the longest prefix that's valid utf8 and maximum `n` bytes.
    ///
    /// Even if the input is valid utf8, the result might be shorter than `n`
    /// bytes, because the original string might have a multi-byte utf8
    /// character at the `n` byte boundary, which, when split, results in an
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

}

// TODO(csongor): tests
