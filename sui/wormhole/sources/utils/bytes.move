module wormhole::bytes {
    use std::vector::{Self};
    use wormhole::cursor::{Self, Cursor};
    use wormhole::myu32::{Self as u32, U32};
    use wormhole::myu256::{Self as u256, U256};

    // we reuse the native bcs serialiser -- it uses little-endian encoding, and
    // we need big-endian, so the results are reversed
    use std::bcs::{Self};

    public fun serialize_u8(buf: &mut vector<u8>, v: u8) {
        vector::push_back<u8>(buf, v);
    }

    public fun serialize_u16_be(buf: &mut vector<u8>, v: u16) {
        let v = bcs::to_bytes(&v);
        let len = vector::length(&v);
        let i = 0;
        while (i < len) {
            vector::push_back(buf, *vector::borrow(&v, len - i - 1));
            i = i + 1;
        };
    }

    public fun serialize_u32_be(buf: &mut vector<u8>, v: U32) {
        let (v0, v1, v2, v3) = u32::split_u8(v);
        serialize_u8(buf, v0);
        serialize_u8(buf, v1);
        serialize_u8(buf, v2);
        serialize_u8(buf, v3);
    }

    public fun serialize_u64_be(buf: &mut vector<u8>, v: u64) {
        let v = bcs::to_bytes(&v);
        vector::reverse(&mut v);
        vector::append(buf, v);
    }

    public fun serialize_u128_be(buf: &mut vector<u8>, v: u128) {
        let v = bcs::to_bytes(&v);
        vector::reverse(&mut v);
        vector::append(buf, v);
    }

    public fun serialize_u256_be(buf: &mut vector<u8>, v: U256) {
        let v = bcs::to_bytes(&v);
        vector::reverse(&mut v);
        vector::append(buf, v);
    }

    public fun from_bytes(buf: &mut vector<u8>, v: vector<u8>){
        vector::append(buf, v)
    }
    
    public fun deserialize_u8(cursor: &mut Cursor<u8>): u8 {
        cursor::poke(cursor)
    }

    public fun deserialize_u16_be(cursor: &mut Cursor<u8>): u16 {
        let res: u64 = 0;
        let i = 0;
        while (i < 2) {
            let b = cursor::poke(cursor);
            res = (res << 8) + (b as u64);
            i = i + 1;
        };
        (res as u16)
    }

    public fun deserialize_u32_be(cursor: &mut Cursor<u8>): U32 {
        let res: u64 = 0;
        let i = 0;
        while (i < 4) {
            let b = cursor::poke(cursor);
            res = (res << 8) + (b as u64);
            i = i + 1;
        };
        u32::from_u64(res)
    }

    public fun deserialize_u64_be(cursor: &mut Cursor<u8>): u64 {
        let res: u64 = 0;
        let i = 0;
        while (i < 8) {
            let b = cursor::poke(cursor);
            res = (res << 8) + (b as u64);
            i = i + 1;
        };
        res
    }

    public fun deserialize_u128_be(cursor: &mut Cursor<u8>): u128 {
        let res: u128 = 0;
        let i = 0;
        while (i < 16) {
            let b = cursor::poke(cursor);
            res = (res << 8) + (b as u128);
            i = i + 1;
        };
        res
    }

    public fun deserialize_u256_be(cursor: &mut Cursor<u8>): U256 {
        let v0 = deserialize_u128_be(cursor);
        let v1 = deserialize_u128_be(cursor);
        u256::add(u256::shl(u256::from_u128(v0), 128), u256::from_u128(v1))
    }

    public fun to_bytes(cursor: &mut Cursor<u8>, len: u64): vector<u8> {
        let result = vector::empty();
        while (len > 0) {
            vector::push_back(&mut result, cursor::poke(cursor));
            len = len - 1;
        };
        result
    }
}

#[test_only]
module wormhole::test_bytes {
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::myu32::{Self as u32};
    use wormhole::myu256::{Self as u256};
    use 0x1::vector;

    #[test]
    fun test_serialize_u8(){
        let u = 0x12;
        let s = vector::empty();
        bytes::serialize_u8(&mut s, u);
        let cur = cursor::new(s);
        let p = bytes::deserialize_u8(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u16_be(){
        let u = 0x1234;
        let s = vector::empty();
        bytes::serialize_u16_be(&mut s, u);
        let cur = cursor::new(s);
        let p = bytes::deserialize_u16_be(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u32_be(){
        let u = u32::from_u64((0x12345678 as u64));
        let s = vector::empty();
        bytes::serialize_u32_be(&mut s, u);
        let cur = cursor::new(s);
        let p = bytes::deserialize_u32_be(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u64_be(){
        let u = 0x1234567812345678;
        let s = vector::empty();
        bytes::serialize_u64_be(&mut s, u);
        let cur = cursor::new(s);
        let p = bytes::deserialize_u64_be(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

     #[test]
    fun test_serialize_u128_be(){
        let u = 0x12345678123456781234567812345678;
        let s = vector::empty();
        bytes::serialize_u128_be(&mut s, u);
        let cur = cursor::new(s);
        let p = bytes::deserialize_u128_be(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u256_be(){
        let u = u256::add(u256::shl(u256::from_u128(0x47386917590997937461700473756125), 128), u256::from_u128(0x9876));
        let s = vector::empty();
        bytes::serialize_u256_be(&mut s, u);
        let exp = x"4738691759099793746170047375612500000000000000000000000000009876";
        assert!(s == exp, 0);
    }

    #[test]
    fun test_from_bytes(){
        let x = vector::empty<u8>();
        let y = vector::empty<u8>();
        vector::push_back<u8>(&mut x, 0x12);
        vector::push_back<u8>(&mut x, 0x34);
        vector::push_back<u8>(&mut x, 0x56);
        bytes::from_bytes(&mut y, x);
        assert!(y == x"123456", 0);
    }

    #[test]
    fun test_deserialize_u8() {
        let cursor = cursor::new(x"99");
        let byte = bytes::deserialize_u8(&mut cursor);
        assert!(byte==0x99, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_deserialize_u16_be() {
        let cursor = cursor::new(x"9987");
        let u = bytes::deserialize_u16_be(&mut cursor);
        assert!(u == 0x9987, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_deserialize_u32_be() {
        let cursor = cursor::new(x"99876543");
        let u = bytes::deserialize_u32_be(&mut cursor);
        assert!(u == u32::from_u64(0x99876543), 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_deserialize_u64_be() {
        let cursor = cursor::new(x"1300000025000001");
        let u = bytes::deserialize_u64_be(&mut cursor);
        assert!(u==0x1300000025000001, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_deserialize_u128_be() {
        let cursor = cursor::new(x"130209AB2500FA0113CD00AE25000001");
        let u = bytes::deserialize_u128_be(&mut cursor);
        assert!(u==0x130209AB2500FA0113CD00AE25000001, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_to_bytes() {
        let cursor = cursor::new(b"hello world");
        let hello = bytes::to_bytes(&mut cursor, 5);
        bytes::deserialize_u8(&mut cursor);
        let world = bytes::to_bytes(&mut cursor, 5);
        assert!(hello == b"hello", 0);
        assert!(world == b"world", 0);
        cursor::destroy_empty(cursor);
    }

}
