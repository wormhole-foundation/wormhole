module wormhole::serialize {
    use std::vector;
    use wormhole::u16::{Self, U16};
    use wormhole::u32::{Self, U32};
    use wormhole::u256::U256;

    // we reuse the native bcs serialiser -- it uses little-endian encoding, and
    // we need big-endian, so the results are reversed
    use std::bcs;

    public fun serialize_u8(buf: &mut vector<u8>, v: u8) {
        vector::push_back<u8>(buf, v);
    }

    public fun serialize_u16(buf: &mut vector<u8>, v: U16) {
        let (v0, v1) = u16::split_u8(v);
        serialize_u8(buf, v0);
        serialize_u8(buf, v1);
    }

    public fun serialize_u32(buf: &mut vector<u8>, v: U32) {
        let (v0, v1, v2, v3) = u32::split_u8(v);
        serialize_u8(buf, v0);
        serialize_u8(buf, v1);
        serialize_u8(buf, v2);
        serialize_u8(buf, v3);
    }

    public fun serialize_u64(buf: &mut vector<u8>, v: u64) {
        let v = bcs::to_bytes(&v);
        vector::reverse(&mut v);
        vector::append(buf, v);
    }

    public fun serialize_u128(buf: &mut vector<u8>, v: u128) {
        let v = bcs::to_bytes(&v);
        vector::reverse(&mut v);
        vector::append(buf, v);
    }

    public fun serialize_u256(buf: &mut vector<u8>, v: U256) {
        let v = bcs::to_bytes(&v);
        vector::reverse(&mut v);
        vector::append(buf, v);
    }

    public fun serialize_vector(buf: &mut vector<u8>, v: vector<u8>) {
        vector::append(buf, v)
    }
}

#[test_only]
module wormhole::test_serialize {
    use wormhole::serialize;
    use wormhole::deserialize;
    use wormhole::cursor;
    use wormhole::u32;
    use wormhole::u16;
    use wormhole::u256;
    use std::vector;

    #[test]
    fun test_serialize_u8() {
        let u = 0x12;
        let s = vector::empty();
        serialize::serialize_u8(&mut s, u);
        let cur = cursor::init(s);
        let p = deserialize::deserialize_u8(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u16() {
        let u = u16::from_u64((0x1234 as u64));
        let s = vector::empty();
        serialize::serialize_u16(&mut s, u);
        let cur = cursor::init(s);
        let p = deserialize::deserialize_u16(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u32() {
        let u = u32::from_u64((0x12345678 as u64));
        let s = vector::empty();
        serialize::serialize_u32(&mut s, u);
        let cur = cursor::init(s);
        let p = deserialize::deserialize_u32(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u64() {
        let u = 0x1234567812345678;
        let s = vector::empty();
        serialize::serialize_u64(&mut s, u);
        let cur = cursor::init(s);
        let p = deserialize::deserialize_u64(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

     #[test]
    fun test_serialize_u128() {
        let u = 0x12345678123456781234567812345678;
        let s = vector::empty();
        serialize::serialize_u128(&mut s, u);
        let cur = cursor::init(s);
        let p = deserialize::deserialize_u128(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u256() {
        let u = u256::add(u256::shl(u256::from_u128(0x47386917590997937461700473756125), 128), u256::from_u128(0x9876));
        let s = vector::empty();
        serialize::serialize_u256(&mut s, u);
        let exp = x"4738691759099793746170047375612500000000000000000000000000009876";
        assert!(s == exp, 0);
    }

    #[test]
    fun test_serialize_vector() {
        let x = vector::empty<u8>();
        let y = vector::empty<u8>();
        vector::push_back<u8>(&mut x, 0x12);
        vector::push_back<u8>(&mut x, 0x34);
        vector::push_back<u8>(&mut x, 0x56);
        serialize::serialize_vector(&mut y, x);
        assert!(y == x"123456", 0);
    }
}
