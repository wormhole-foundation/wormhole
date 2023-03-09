module wormhole::bytes20 {
    use std::vector::{Self};

    // Errors.
    const E_INVALID_BYTES20: u64 = 0;
    const E_INVALID_FROM_BYTES: u64 = 1;

    /// 20.
    const LEN: u64 = 20;

    struct Bytes20 has copy, drop, store {
        data: vector<u8>
    }

    public fun new(data: vector<u8>): Bytes20 {
        assert!(is_valid(&data), E_INVALID_BYTES20);
        Bytes20 { data }
    }

    public fun default(): Bytes20 {
        let data = vector::empty();
        let i = 0;
        while (i < LEN) {
            vector::push_back(&mut data, 0);
            i = i + 1;
        };
        new(data)
    }

    public fun from_bytes(buf: vector<u8>): Bytes20 {
        let len = vector::length(&buf);
        if (len > LEN) {
            trim_nonzero_left(&mut buf);
            new(buf)
        } else {
            new(pad_left(&buf, false))
        }
    }

    public fun to_bytes(value: Bytes20): vector<u8> {
        let Bytes20 { data } = value;
        data
    }

    public fun data(self: &Bytes20): vector<u8> {
        self.data
    }

    public fun to_address(value: Bytes20): address {
        let Bytes20 { data } = value;
        sui::address::from_bytes(data)
    }

    public fun from_address(addr: address): Bytes20 {
        new(sui::address::to_bytes(addr))
    }

    public fun is_nonzero(self: &Bytes20): bool {
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

    /// Trim bytes from the left if they are zero. If any of these bytes
    /// are non-zero, abort.
    fun trim_nonzero_left(data: &mut vector<u8>) {
        vector::reverse(data);
        let (i, n) = (0, vector::length(data) - LEN);
        while (i < n) {
            assert!(vector::pop_back(data) == 0, E_INVALID_FROM_BYTES);
            i = i + 1;
        };
        vector::reverse(data);
    }
}

#[test_only]
module wormhole::bytes20_test {
    use std::vector::{Self};

    use wormhole::bytes20::{Self};

    #[test]
    public fun new() {
        let data = x"deadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
        assert!(vector::length(&data) == 20, 0);
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
        assert!(vector::length(&data) != 20, 0);
        bytes20::new(data);
    }
}
