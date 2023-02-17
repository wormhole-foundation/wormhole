/// 32 byte, left-padded address representing an arbitrary address, to be used in VAAs to
/// refer to addresses.
module wormhole::external_address {
    use wormhole::cursor::{Cursor};
    use wormhole::bytes::{Self};
    use wormhole::bytes32::{Self, Bytes32};

    const E_VECTOR_TOO_LONG: u64 = 0;
    const E_INVALID_EXTERNAL_ADDRESS: u64 = 1;
    const E_ZERO_ADDRESS: u64 = 2;

    struct ExternalAddress has drop, copy, store {
        value: Bytes32,
    }

    public fun new(value: Bytes32): ExternalAddress {
        ExternalAddress { value }
    }

    public fun default(): ExternalAddress {
        new(bytes32::default())
    }

    public fun new_nonzero(value: Bytes32): ExternalAddress {
        assert!(bytes32::is_nonzero(&value), E_ZERO_ADDRESS);
        new(value)
    }

    public fun from_bytes(buf: vector<u8>): ExternalAddress {
        new(bytes32::from_bytes(buf))
    }

    public fun from_nonzero_bytes(buf: vector<u8>): ExternalAddress{
        new_nonzero(bytes32::from_bytes(buf))
    }

    public fun to_bytes(ext: ExternalAddress): vector<u8> {
        bytes32::to_bytes(ext.value)
    }

    public fun to_bytes32(ext: ExternalAddress): Bytes32 {
        let ExternalAddress { value } = ext;
        value
    }

    public fun from_u64_be(value: u64): ExternalAddress {
        new(bytes32::from_u64_be(value))
    }

    public fun to_u64_be(ext: ExternalAddress): u64 {
        bytes32::to_u64_be(ext.value)
    }

    public fun deserialize(cur: &mut Cursor<u8>): ExternalAddress {
        let bytes = bytes::to_bytes(cur, 32);
        from_bytes(bytes)
    }

    public fun serialize(buf: &mut vector<u8>, ext: ExternalAddress) {
        bytes::from_bytes(buf, bytes32::to_bytes(ext.value))
    }

    /// Convert an `ExternalAddress` to a native Sui address.
    ///
    /// Sui addresses are 20 bytes, while external addresses are represented as
    /// 32 bytes, left-padded with 0s. This function thus takes the last 20
    /// bytes of an external address, and reverts if the first 12 bytes contain
    /// non-0 bytes.
    public fun to_address(ext: ExternalAddress): address {
        bytes32::to_address(ext.value)
    }

}

#[test_only]
module wormhole::external_address_test {
    use std::vector::{Self};
    use wormhole::bytes20::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::external_address::{Self};

    // test get_bytes and left_pad
    #[test]
    public fun test_left_pad() {
        let v = x"123456789123456789123456789123451234567891234567891234"; // less than 32 bytes
        let res = external_address::from_bytes(v);
        let bytes = external_address::to_bytes(res);
        let m = x"0000000000";
        vector::append(&mut m, v);
        assert!(bytes == m, 0);
    }

    #[test]
    public fun test_left_pad_length_32_vector() {
        let v = x"1234567891234567891234567891234512345678912345678912345678912345"; //32 bytes
        let res = external_address::from_bytes(v);
        let bytes = external_address::to_bytes(res);
        assert!(bytes == v, 0);
    }

    #[test]
    #[expected_failure(
        abort_code = bytes32::E_INVALID_BYTES32,
        location = bytes32
    )]
    public fun test_left_pad_vector_too_long() {
        let v = x"123456789123456789123456789123451234567891234567891234567891234500"; //33 bytes
        external_address::from_bytes(v);
    }

    #[test]
    #[expected_failure(
        abort_code = bytes20::E_INVALID_FROM_BYTES,
        location = bytes20
    )]
    public fun test_to_address_too_long() {
        // non-0 bytes in first 12 bytes
        let v = x"0000010000000000000000000000000000000000000000000000000000001234";
        let res = external_address::from_bytes(v);
        let _address = external_address::to_address(res);
    }

    #[test]
    public fun test_to_address() {
        let v = x"0000000000000000000000000000000000000000000000000000000000001234";
        let res = external_address::from_bytes(v);
        let address = external_address::to_address(res);
        assert!(address == @0x1234, 0);
    }
}
