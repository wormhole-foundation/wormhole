// SPDX-License-Identifier: Apache 2

/// This module implements a custom type for a 32-byte standardized address,
/// which is meant to represent an address from any other network.
module wormhole::external_address {
    use sui::object::{Self, ID};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::cursor::{Cursor};

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
        sui::address::from_bytes(to_bytes(ext))
    }

    /// Create `ExternalAddress` from `address`.
    public fun from_address(addr: address): ExternalAddress {
        new(bytes32::from_address(addr))
    }

    /// Create `ExternalAddress` from `ID`.
    public fun from_id(id: ID): ExternalAddress {
        new(bytes32::from_bytes(object::id_to_bytes(&id)))
    }

    /// Check whether underlying data is not all zeros.
    public fun is_nonzero(self: &ExternalAddress): bool {
        bytes32::is_nonzero(&self.value)
    }
}

#[test_only]
module wormhole::external_address_tests {
    use wormhole::bytes32::{Self};
    use wormhole::external_address::{Self};

    #[test]
    public fun test_bytes() {
        let data =
            bytes32::new(
                x"1234567891234567891234567891234512345678912345678912345678912345"
            );
        let addr = external_address::new(data);
        assert!(external_address::to_bytes(addr) == bytes32::to_bytes(data), 0);
    }

    #[test]
    public fun test_address() {
        let data =
            bytes32::new(
                x"0000000000000000000000000000000000000000000000000000000000001234"
            );
        let addr = external_address::new(data);
        assert!(external_address::to_address(addr) == @0x1234, 0);
        assert!(addr == external_address::from_address(@0x1234), 0);
    }
}
