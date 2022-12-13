/// The `uri` module defines the `URI` type which represents UTF8 encoded
/// strings that are at most 200 characters long, used for representing the URI
/// field of an NFT. Since these strings are not fixed length, their binary
/// encoding includes a length byte prefix.
module nft_bridge::uri {

    use std::string::{Self, String};
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
        assert!(vector::length(&b) <= MAX_LENGTH, E_URI_TOO_LONG);
        URI { string: string::utf8(b) }
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


}

#[test_only]
module nft_bridge::uri_test {
    use nft_bridge::uri;

    #[test]
    public fun test_uri_from_bytes_valid() {
        let utf8 = b"hello world";
        uri::from_bytes(utf8);
    }

    // The input string here is not a valid utf8 bytestring
    #[test]
    #[expected_failure(abort_code = 1, location = std::string)]
    public fun test_uri_from_bytes_invalid() {
        let not_utf8 = x"afafafaf";
        uri::from_bytes(not_utf8);
    }

    // The string is longer than 200 characters, in which case we abort.
    #[test]
    #[expected_failure(abort_code = 0, location = nft_bridge::uri)]
    public fun test_uri_too_long() {
        let too_long = std::string::utf8(b"this string is very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very very long");
        uri::from_string(&too_long);
    }

    #[test]
    public fun test_serialize_roundtrip() {
        let uri = uri::from_bytes(b"hello world");
        let vec = std::vector::empty();
        uri::serialize(&mut vec, uri);
        let c = wormhole::cursor::init(vec);
        let uri2 = uri::deserialize(&mut c);
        wormhole::cursor::destroy_empty(c);
        assert!(uri == uri2, 0);
    }
}
