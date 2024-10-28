// SPDX-License-Identifier: Apache 2

/// This module implements a custom type representing a fixed-size array of
/// length 20.
module wormhole::bytes20 {

    use wormhole::bytes::{Self};
    use wormhole::cursor::{Cursor};

    /// Invalid vector<u8> length to create `Bytes20`.
    const E_INVALID_BYTES20: u64 = 0;
    /// Found non-zero bytes when attempting to trim `vector<u8>`.
    const E_CANNOT_TRIM_NONZERO: u64 = 1;

    /// 20.
    const LEN: u64 = 20;

    /// Container for `vector<u8>`, which has length == 20.
    public struct Bytes20 has copy, drop, store {
        data: vector<u8>
    }

    public fun length(): u64 {
        LEN
    }

    /// Create new `Bytes20`, which checks the length of input `data`.
    public fun new(data: vector<u8>): Bytes20 {
        assert!(is_valid(&data), E_INVALID_BYTES20);
        Bytes20 { data }
    }

    /// Create new `Bytes20` of all zeros.
    public fun default(): Bytes20 {
        let mut data = vector[];
        let mut i = 0;
        while (i < LEN) {
            data.push_back(0);
            i = i + 1;
        };
        new(data)
    }

    /// Retrieve underlying `data`.
    public fun data(self: &Bytes20): vector<u8> {
        self.data
    }

    /// Either trim or pad (depending on length of the input `vector<u8>`) to 20
    /// bytes.
    public fun from_bytes(mut buf: vector<u8>): Bytes20 {
        let len = buf.length();
        if (len > LEN) {
            trim_nonzero_left(&mut buf);
            new(buf)
        } else {
            new(pad_left(&buf, false))
        }
    }

    /// Destroy `Bytes20` for its underlying data.
    public fun to_bytes(value: Bytes20): vector<u8> {
        let Bytes20 { data } = value;
        data
    }

    /// Drain 20 elements of `Cursor<u8>` to create `Bytes20`.
    public fun take(cur: &mut Cursor<u8>): Bytes20 {
        new(bytes::take_bytes(cur, LEN))
    }

    /// Validate that any of the bytes in underlying data is non-zero.
    public fun is_nonzero(self: &Bytes20): bool {
        let mut i = 0;
        while (i < LEN) {
            if (*self.data.borrow(i) > 0) {
                return true
            };
            i = i + 1;
        };

        false
    }

    /// Check that the input data is correct length.
    fun is_valid(data: &vector<u8>): bool {
        data.length() == LEN
    }

    /// For vector size less than 20, add zeros to the left.
    fun pad_left(data: &vector<u8>, data_reversed: bool): vector<u8> {
        let mut out = vector[];
        let len = data.length();
        let mut i = len;
        while (i < LEN) {
            out.push_back(0);
            i = i + 1;
        };
        if (data_reversed) {
            let mut i = 0;
            while (i < len) {
                out.push_back(
                    *data.borrow(len - i - 1)
                );
                i = i + 1;
            };
        } else {
            out.append(*data);
        };

        out
    }

    /// Trim bytes from the left if they are zero. If any of these bytes
    /// are non-zero, abort.
    fun trim_nonzero_left(data: &mut vector<u8>) {
        data.reverse();
        let (mut i, n) = (0, data.length() - LEN);
        while (i < n) {
            assert!(data.pop_back() == 0, E_CANNOT_TRIM_NONZERO);
            i = i + 1;
        };
        data.reverse();
    }
}

#[test_only]
module wormhole::bytes20_tests {

    use wormhole::bytes20::{Self};

    #[test]
    public fun new() {
        let data = x"deadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
        assert!(data.length() == 20, 0);
        let actual = bytes20::new(data);

        assert!(bytes20::data(&actual) == data, 0);
    }

    #[test]
    public fun default() {
        let actual = bytes20::default();
        let expected = x"0000000000000000000000000000000000000000";
        assert!(bytes20::data(&actual) == expected, 0);
    }

    #[test]
    public fun from_bytes() {
        let actual = bytes20::from_bytes(x"deadbeef");
        let expected = x"00000000000000000000000000000000deadbeef";
        assert!(bytes20::data(&actual) == expected, 0);
    }

    #[test]
    public fun is_nonzero() {
        let data = x"deadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
        let actual = bytes20::new(data);
        assert!(bytes20::is_nonzero(&actual), 0);

        let zeros = bytes20::default();
        assert!(!bytes20::is_nonzero(&zeros), 0);
    }

    #[test]
    #[expected_failure(abort_code = bytes20::E_INVALID_BYTES20)]
    public fun cannot_new_non_20_byte_vector() {
        let data =
            x"deadbeefdeadbeefdeadbeefdeadbeefdeadbe";
        assert!(data.length() != 20, 0);
        bytes20::new(data);
    }
}
