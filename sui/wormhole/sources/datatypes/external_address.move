// SPDX-License-Identifier: Apache 2

/// This module implements a custom type for a 32-byte standardized address,
/// which is meant to represent an address from any other network.
module wormhole::external_address {
    use wormhole::cursor::{Cursor};
    use wormhole::bytes32::{Self, Bytes32};

    /// Underlying data is all zeros.
    const E_ZERO_ADDRESS: u64 = 0;

    /// Container for `Bytes32`.
    struct ExternalAddress has copy, drop, store {
        value: Bytes32,
    }

    /// Create `ExternalAddress`.
    public fun new(value: Bytes32): ExternalAddress {
        ExternalAddress { value }
    }

    /// Create `ExternalAddress` of all zeros.`
    public fun default(): ExternalAddress {
        new(bytes32::default())
    }

    /// Create `ExternalAddress` ensuring that not all bytes are zero.
    public fun new_nonzero(value: Bytes32): ExternalAddress {
        assert!(bytes32::is_nonzero(&value), E_ZERO_ADDRESS);
        new(value)
    }

    /// Create `ExternalAddress` with `vector<u8>` of length == 32.
    public fun from_bytes(buf: vector<u8>): ExternalAddress {
        new(bytes32::new(buf))
    }

    /// Create `ExternalAddress` with `vector<u8>` of length == 32 ensuring that
    /// not all bytes are zero.
    public fun from_nonzero_bytes(buf: vector<u8>): ExternalAddress {
        new_nonzero(bytes32::new(buf))
    }

    /// Destroy `ExternalAddress` for underlying bytes as `vector<u8>`.
    public fun to_bytes(ext: ExternalAddress): vector<u8> {
        bytes32::to_bytes(to_bytes32(ext))
    }

    /// Destroy 'ExternalAddress` for underlying data.
    public fun to_bytes32(ext: ExternalAddress): Bytes32 {
        let ExternalAddress { value } = ext;
        value
    }

    /// Drain 32 elements of `Cursor<u8>` to create `ExternalAddress`.
    public fun take_bytes(cur: &mut Cursor<u8>): ExternalAddress {
        new(bytes32::take_bytes(cur))
    }

    /// Drain 32 elements of `Cursor<u8>` to create `ExternalAddress` ensuring
    /// that not all bytes are zero.
    public fun take_nonzero(cur: &mut Cursor<u8>): ExternalAddress {
        new_nonzero(bytes32::take_bytes(cur))
    }

    /// Destroy `ExternalAddress` to represent its underlying data as `address`.
    public fun to_address(ext: ExternalAddress): address {
        bytes32::to_address(to_bytes32(ext))
    }

    /// Create `ExternalAddress` from `address`.
    public fun from_address(addr: address): ExternalAddress {
        new(bytes32::from_address(addr))
    }

    /// Check whether underlying data is not all zeros.
    public fun is_nonzero(self: &ExternalAddress): bool {
        bytes32::is_nonzero(&self.value)
    }

    #[test_only]
    public fun from_any_bytes(buf: vector<u8>): ExternalAddress {
        new(bytes32::from_bytes(buf))
    }
}

#[test_only]
module wormhole::external_address_tests {
    use wormhole::bytes20::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::external_address::{Self};

    #[test]
    public fun test_left_pad_length_32_vector() {
        let v = x"1234567891234567891234567891234512345678912345678912345678912345"; //32 bytes
        let res = external_address::from_bytes(v);
        let bytes = external_address::to_bytes(res);
        assert!(bytes == v, 0);
    }

    #[test]
    #[expected_failure(abort_code = bytes32::E_INVALID_BYTES32)]
    public fun test_left_pad_vector_too_long() {
        let v = x"123456789123456789123456789123451234567891234567891234567891234500"; //33 bytes
        external_address::from_bytes(v);
    }

    #[test]
    #[expected_failure(abort_code = bytes20::E_CANNOT_TRIM_NONZERO)]
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
