/// 32 byte, left-padded address representing an arbitrary address, to be used in VAAs to
/// refer to addresses.
module wormhole::external_address {
    use std::vector::{Self};

    use sui::address::{Self};

    use wormhole::cursor::Cursor;
    use wormhole::bytes::{Self};

    const E_VECTOR_TOO_LONG: u64 = 0;
    const E_INVALID_EXTERNAL_ADDRESS: u64 = 1;

    struct ExternalAddress has drop, copy, store {
        external_address: vector<u8>,
    }

    public fun get_bytes(e: &ExternalAddress): vector<u8> {
        e.external_address
    }

    public fun pad_left_32(input: &vector<u8>): vector<u8>{
        let len = vector::length<u8>(input);
        assert!(len <= 32, E_VECTOR_TOO_LONG);
        let ret = vector::empty<u8>();
        let zeros_remaining = 32 - len;
        while (zeros_remaining > 0){
            vector::push_back<u8>(&mut ret, 0);
            zeros_remaining = zeros_remaining - 1;
        };
        vector::append<u8>(&mut ret, *input);
        ret
    }

    public fun left_pad(s: &vector<u8>): ExternalAddress {
        let padded_vector = pad_left_32(s);
        ExternalAddress { external_address: padded_vector}
    }

    public fun from_bytes(bytes: vector<u8>): ExternalAddress {
        left_pad(&bytes)
    }

    public fun deserialize(cur: &mut Cursor<u8>): ExternalAddress {
        let bytes = bytes::to_bytes(cur, 32);
        from_bytes(bytes)
    }

    public fun serialize(buf: &mut vector<u8>, e: ExternalAddress) {
        bytes::from_bytes(buf, e.external_address)
    }

    /// Convert an `ExternalAddress` to a native Sui address.
    ///
    /// Sui addresses are 20 bytes, while external addresses are represented as
    /// 32 bytes, left-padded with 0s. This function thus takes the last 20
    /// bytes of an external address, and reverts if the first 12 bytes contain
    /// non-0 bytes.
    public fun to_address(e: &ExternalAddress): address {
        let vec = e.external_address;
        // we reverse the vector and drop the last 12 bytes
        vector::reverse(&mut vec);
        let bytes_to_drop = 12;
        while (bytes_to_drop > 0) {
            let last_byte = vector::pop_back(&mut vec);
            // ensure no junk in the first 12 bytes
            assert!(last_byte == 0, E_INVALID_EXTERNAL_ADDRESS);
            bytes_to_drop = bytes_to_drop - 1;
        };
        // reverse back to original order
        vector::reverse(&mut vec);
        address::from_bytes(vec)
    }

}

#[test_only]
module wormhole::external_address_test {
    use wormhole::external_address;
    use std::vector::{Self};

    // test get_bytes and left_pad
    #[test]
    public fun test_left_pad() {
        let v = x"123456789123456789123456789123451234567891234567891234"; // less than 32 bytes
        let res = external_address::left_pad(&v);
        let bytes = external_address::get_bytes(&res);
        let m = x"0000000000";
        vector::append(&mut m, v);
        assert!(bytes == m, 0);
    }

    #[test]
    public fun test_left_pad_length_32_vector() {
        let v = x"1234567891234567891234567891234512345678912345678912345678912345"; //32 bytes
        let res = external_address::left_pad(&v);
        let bytes = external_address::get_bytes(&res);
        assert!(bytes == v, 0);
    }

    #[test]
    #[expected_failure(abort_code = 0, location=wormhole::external_address)]
    public fun test_left_pad_vector_too_long() {
        let v = x"123456789123456789123456789123451234567891234567891234567891234500"; //33 bytes
        let res = external_address::left_pad(&v);
        let bytes = external_address::get_bytes(&res);
        assert!(bytes == v, 0);
    }

    #[test]
    public fun test_from_bytes() {
        let v = x"1234";
        let ea = external_address::from_bytes(v);
        let bytes = external_address::get_bytes(&ea);
        let w = x"000000000000000000000000000000000000000000000000000000000000";
        vector::append(&mut w, v);
        assert!(bytes == w, 0);
    }

    #[test]
    #[expected_failure(abort_code = 0, location=wormhole::external_address)]
    public fun test_from_bytes_over_32_bytes() {
        let v = x"00000000000000000000000000000000000000000000000000000000000000001234";
        let ea = external_address::from_bytes(v);
        let _bytes = external_address::get_bytes(&ea);
    }

    #[test]
    fun test_pad_left_short() {
        let v = x"11";
        let pad_left_v = external_address::pad_left_32(&v);
        assert!(pad_left_v == x"0000000000000000000000000000000000000000000000000000000000000011", 0);
    }

    #[test]
    fun test_pad_left_exact() {
        let v = x"5555555555555555555555555555555555555555555555555555555555555555";
        let pad_left_v = external_address::pad_left_32(&v);
        assert!(pad_left_v == x"5555555555555555555555555555555555555555555555555555555555555555", 0);
    }

    #[test]
    #[expected_failure(abort_code = 0, location=wormhole::external_address)]
    fun test_pad_left_long() {
        let v = x"665555555555555555555555555555555555555555555555555555555555555555";
        external_address::pad_left_32(&v);
    }

    #[test]
    #[expected_failure(abort_code = 1, location=wormhole::external_address)]
    public fun test_to_address_too_long() {
        // non-0 bytes in first 12 bytes
        let v = x"0000010000000000000000000000000000000000000000000000000000001234";
        let res = external_address::from_bytes(v);
        let _address = external_address::to_address(&res);
    }

    #[test]
    public fun test_to_address() {
        let v = x"0000000000000000000000000000000000000000000000000000000000001234";
        let res = external_address::from_bytes(v);
        let address = external_address::to_address(&res);
        assert!(address == @0x1234, 0);
    }
}
