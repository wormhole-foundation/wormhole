module wormhole::bytes32 {
    use std::vector::{Self};
    use sui::bcs::{Self};

    use wormhole::bytes::{Self};
    use wormhole::bytes20::{Self};
    use wormhole::cursor::{Cursor};

    // Errors.
    const E_INVALID_BYTES32: u64 = 0;
    const E_INVALID_U64_BE: u64 = 1;

    // Of course LEN == 32.
    const LEN: u64 = 32;

    struct Bytes32 has copy, drop, store {
        data: vector<u8>,
    }

    public fun new(data: vector<u8>): Bytes32 {
        assert!(is_valid(&data), E_INVALID_BYTES32);
        Bytes32 {
            data
        }
    }

    public fun default(): Bytes32 {
        let data = vector::empty();
        let i = 0;
        while (i < LEN) {
            vector::push_back(&mut data, 0u8);
            i = i + 1;
        };
        new(data)
    }

    public fun data(self: &Bytes32): vector<u8> {
        self.data
    }

    public fun from_u64_be(value: u64): Bytes32 {
        let buf = pad_left(&bcs::to_bytes(&value), true);
        new(buf)
    }

    public fun to_u64_be(value: Bytes32): u64 {
        let Bytes32 { data } = value;
        vector::reverse(&mut data);

        let i = 0;
        while (i < 24) {
            assert!(vector::pop_back(&mut data) == 0, E_INVALID_U64_BE);
            i = i + 1;
        };
        bcs::peel_u64(&mut bcs::new(data))
    }

    public fun from_bytes(buf: vector<u8>): Bytes32 {
        new(pad_left(&buf, false))
    }

    public fun to_bytes(value: Bytes32): vector<u8> {
        let Bytes32 { data } = value;
        data
    }

    public fun deserialize(cur: &mut Cursor<u8>): Bytes32 {
        new(bytes::to_bytes(cur, 32))
    }

    public fun from_address(addr: address): Bytes32 {
        new(pad_left(&bytes20::data(&bytes20::from_address(addr)), false))
    }

    public fun to_address(value: Bytes32): address {
        let Bytes32 { data } = value;
        bytes20::to_address(bytes20::from_bytes(data))
    }

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

    fun is_valid(data: &vector<u8>): bool {
        vector::length(data) == LEN
    }

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
    public fun from_u64_be() {
        let actual = bytes32::from_u64_be(1u64 << 32);
        let expected =
            x"0000000000000000000000000000000000000000000000000000000100000000";
        assert!(bytes32::data(&actual) == expected, 0);
    }

    #[test]
    public fun to_u64_be() {
        let actual = bytes32::new(
            x"0000000000000000000000000000000000000000000000000000000100000000"
        );
        assert!(bytes32::to_u64_be(actual) == (1u64 << 32), 0);
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
