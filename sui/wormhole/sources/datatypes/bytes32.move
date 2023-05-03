// SPDX-License-Identifier: Apache 2

/// This module implements a custom type representing a fixed-size array of
/// length 32.
module wormhole::bytes32 {
    use std::option::{Self};
    use std::string::{Self, String};
    use std::vector::{Self};
    use sui::bcs::{Self};

    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self, Cursor};

    /// Invalid vector<u8> length to create `Bytes32`.
    const E_INVALID_BYTES32: u64 = 0;
    /// Found non-zero bytes when attempting to trim `vector<u8>`.
    const E_CANNOT_TRIM_NONZERO: u64 = 1;
    /// Value of deserialized 32-byte array data overflows u64 max.
    const E_U64_OVERFLOW: u64 = 2;

    /// 32.
    const LEN: u64 = 32;

    /// Container for `vector<u8>`, which has length == 32.
    struct Bytes32 has copy, drop, store {
        data: vector<u8>,
    }

    public fun length(): u64 {
        LEN
    }

    /// Create new `Bytes32`, which checks the length of input `data`.
    public fun new(data: vector<u8>): Bytes32 {
        assert!(is_valid(&data), E_INVALID_BYTES32);
        Bytes32 { data }
    }

    /// Create new `Bytes20` of all zeros.
    public fun default(): Bytes32 {
        let data = vector::empty();
        let i = 0;
        while (i < LEN) {
            vector::push_back(&mut data, 0);
            i = i + 1;
        };
        new(data)
    }

    /// Retrieve underlying `data`.
    public fun data(self: &Bytes32): vector<u8> {
        self.data
    }

    /// Serialize `u256` as big-endian format in zero-padded `Bytes32`.
    public fun from_u256_be(value: u256): Bytes32 {
        let buf = bcs::to_bytes(&value);
        vector::reverse(&mut buf);
        new(buf)
    }

    /// Deserialize from big-endian `u256`.
    public fun to_u256_be(value: Bytes32): u256 {
        let cur = cursor::new(to_bytes(value));
        let out = bytes::take_u256_be(&mut cur);
        cursor::destroy_empty(cur);

        out
    }

    /// Serialize `u64` as big-endian format in zero-padded `Bytes32`.
    public fun from_u64_be(value: u64): Bytes32 {
        from_u256_be((value as u256))
    }

    /// Deserialize from big-endian `u64` as long as the data does not
    /// overflow.
    public fun to_u64_be(value: Bytes32): u64 {
        let num = to_u256_be(value);
        assert!(num < (1u256 << 64), E_U64_OVERFLOW);
        (num as u64)
    }

    /// Either trim or pad (depending on length of the input `vector<u8>`) to 32
    /// bytes.
    public fun from_bytes(buf: vector<u8>): Bytes32 {
        let len = vector::length(&buf);
        if (len > LEN) {
            trim_nonzero_left(&mut buf);
            new(buf)
        } else {
            new(pad_left(&buf, false))
        }
    }

    /// Destroy `Bytes32` for its underlying data.
    public fun to_bytes(value: Bytes32): vector<u8> {
        let Bytes32 { data } = value;
        data
    }

    /// Drain 32 elements of `Cursor<u8>` to create `Bytes32`.
    public fun take_bytes(cur: &mut Cursor<u8>): Bytes32 {
        new(bytes::take_bytes(cur, LEN))
    }

    /// Destroy `Bytes32` to represent its underlying data as `address`.
    public fun to_address(value: Bytes32): address {
        sui::address::from_bytes(to_bytes(value))
    }

    /// Create `Bytes32` from `address`.
    public fun from_address(addr: address): Bytes32 {
        new(sui::address::to_bytes(addr))
    }

    public fun from_utf8(str: String): Bytes32 {
        let data = *string::bytes(&str);
        let len = vector::length(&data);
        if (len > LEN) {
            // Trim from end.
            let i = len;
            while (i > LEN) {
                vector::pop_back(&mut data);
                i = i - 1;
            }
        } else {
            // Pad right to `LEN`.
            let i = len;
            while (i < LEN) {
                vector::push_back(&mut data, 0);
                i = i + 1;
            }
        };

        new(data)
    }

    /// Even if the input is valid utf8, the result might be shorter than 32
    /// bytes, because the original string might have a multi-byte utf8
    /// character at the 32 byte boundary, which, when split, results in an
    /// invalid code point, so we remove it.
    public fun to_utf8(value: Bytes32): String {
        let data = to_bytes(value);

        let utf8 = string::try_utf8(data);
        while (option::is_none(&utf8)) {
            vector::pop_back(&mut data);
            utf8 = string::try_utf8(data);
        };

        let buf = *string::bytes(&option::extract(&mut utf8));

        // Now trim zeros from the right.
        while (
            *vector::borrow(&buf, vector::length(&buf) - 1) == 0
        ) {
            vector::pop_back(&mut buf);
        };

        string::utf8(buf)
    }

    /// Validate that any of the bytes in underlying data is non-zero.
    public fun is_nonzero(self: &Bytes32): bool {
        let i = 0;
        while (i < LEN) {
            if (*vector::borrow(&self.data, i) > 0) {
                return true
            };
            i = i + 1;
        };

        false
    }

    /// Check that the input data is correct length.
    fun is_valid(data: &vector<u8>): bool {
        vector::length(data) == LEN
    }

    /// For vector size less than 32, add zeros to the left.
    fun pad_left(data: &vector<u8>, data_reversed: bool): vector<u8> {
        let out = vector::empty();
        let len = vector::length(data);
        let i = len;
        while (i < LEN) {
            vector::push_back(&mut out, 0);
            i = i + 1;
        };
        if (data_reversed) {
            let i = 0;
            while (i < len) {
                vector::push_back(
                    &mut out,
                    *vector::borrow(data, len - i - 1)
                );
                i = i + 1;
            };
        } else {
            vector::append(&mut out, *data);
        };

        out
    }

    /// Trim bytes from the left if they are zero. If any of these bytes
    /// are non-zero, abort.
    fun trim_nonzero_left(data: &mut vector<u8>) {
        vector::reverse(data);
        let (i, n) = (0, vector::length(data) - LEN);
        while (i < n) {
            assert!(vector::pop_back(data) == 0, E_CANNOT_TRIM_NONZERO);
            i = i + 1;
        };
        vector::reverse(data);
    }
}

#[test_only]
module wormhole::bytes32_tests {
    use std::vector::{Self};

    use wormhole::bytes32::{Self};

    #[test]
    public fun new() {
        let data =
            x"deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
        assert!(vector::length(&data) == 32, 0);
        let actual = bytes32::new(data);

        assert!(bytes32::data(&actual) == data, 0);
    }

    #[test]
    public fun default() {
        let actual = bytes32::default();
        let expected =
            x"0000000000000000000000000000000000000000000000000000000000000000";
        assert!(bytes32::data(&actual) == expected, 0);
    }

    #[test]
    public fun from_u256_be() {
        let actual = bytes32::from_u256_be(1 << 32);
        let expected =
            x"0000000000000000000000000000000000000000000000000000000100000000";
        assert!(bytes32::data(&actual) == expected, 0);
    }

    #[test]
    public fun to_u256_be() {
        let actual = bytes32::new(
            x"0000000000000000000000000000000000000000000000000000000100000000"
        );
        assert!(bytes32::to_u256_be(actual) == (1 << 32), 0);
    }

    #[test]
    public fun from_bytes() {
        let actual = bytes32::from_bytes(x"deadbeef");
        let expected =
            x"00000000000000000000000000000000000000000000000000000000deadbeef";
        assert!(bytes32::data(&actual) == expected, 0);
    }

    #[test]
    public fun is_nonzero() {
        let data =
            x"deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
        let actual = bytes32::new(data);
        assert!(bytes32::is_nonzero(&actual), 0);

        let zeros = bytes32::default();
        assert!(!bytes32::is_nonzero(&zeros), 0);
    }

    #[test]
    #[expected_failure(abort_code = bytes32::E_INVALID_BYTES32)]
    public fun cannot_new_non_32_byte_vector() {
        let data =
            x"deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbe";
        assert!(vector::length(&data) != 32, 0);
        bytes32::new(data);
    }
}
