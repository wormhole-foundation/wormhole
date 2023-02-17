module wormhole::bytes {
    use std::vector::{Self};
    use std::bcs::{Self};
    use wormhole::cursor::{Self, Cursor};

    public fun serialize_u8(buf: &mut vector<u8>, v: u8) {
        vector::push_back<u8>(buf, v);
    }

    public fun serialize_u16_be(buf: &mut vector<u8>, value: u16) {
        serialize_reverse(buf, value);
    }

    public fun serialize_u32_be(buf: &mut vector<u8>, value: u32) {
        serialize_reverse(buf, value);
    }

    public fun serialize_u64_be(buf: &mut vector<u8>, value: u64) {
        serialize_reverse(buf, value);
    }

    public fun serialize_u128_be(buf: &mut vector<u8>, value: u128) {
        serialize_reverse(buf, value);
    }

    public fun serialize_u256_be(buf: &mut vector<u8>, value: u256) {
        serialize_reverse(buf, value);
    }

    fun serialize_reverse<T: drop>(buf: &mut vector<u8>, v: T) {
        let v = bcs::to_bytes(&v);
        let len = vector::length(&v);
        let i = 0;
        while (i < len) {
            vector::push_back(buf, *vector::borrow(&v, len - i - 1));
            i = i + 1;
        };
    }

    public fun from_bytes(buf: &mut vector<u8>, other: vector<u8>){
        vector::append(buf, other)
    }

    public fun deserialize_u8(cur: &mut Cursor<u8>): u8 {
        cursor::poke(cur)
    }

    public fun deserialize_u16_be(cur: &mut Cursor<u8>): u16 {
        let res: u64 = 0;
        let i = 0;
        while (i < 2) {
            let b = cursor::poke(cur);
            res = (res << 8) + (b as u64);
            i = i + 1;
        };
        (res as u16)
    }

    public fun deserialize_u32_be(cur: &mut Cursor<u8>): u32 {
        let res: u64 = 0;
        let i = 0;
        while (i < 4) {
            let b = cursor::poke(cur);
            res = (res << 8) + (b as u64);
            i = i + 1;
        };
        (res as u32)
    }

    public fun deserialize_u64_be(cur: &mut Cursor<u8>): u64 {
        let res: u64 = 0;
        let i = 0;
        while (i < 8) {
            let b = cursor::poke(cur);
            res = (res << 8) + (b as u64);
            i = i + 1;
        };
        res
    }

    public fun deserialize_u128_be(cur: &mut Cursor<u8>): u128 {
        let res: u128 = 0;
        let i = 0;
        while (i < 16) {
            let b = cursor::poke(cur);
            res = (res << 8) + (b as u128);
            i = i + 1;
        };
        res
    }

    public fun deserialize_u256_be(cur: &mut Cursor<u8>): u256 {
        let res: u256 = 0;
        let i = 0;
        while (i < 32) {
            let b = cursor::poke(cur);
            res = (res << 8) + (b as u256);
            i = i + 1;
        };
        res
    }

    public fun to_bytes(cur: &mut Cursor<u8>, num_bytes: u64): vector<u8> {
        let result = vector::empty();
        let i = 0;
        while (i < num_bytes) {
            vector::push_back(&mut result, cursor::poke(cur));
            i = i + 1;
        };
        result
    }
}

#[test_only]
module wormhole::test_bytes {
    use std::vector::{Self};
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};

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
        let u = 0x12345678;
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
        let u =
            0x4738691759099793746170047375612500000000000000000000000000009876;
        let s = vector::empty();
        bytes::serialize_u256_be(&mut s, u);
        assert!(
            s == x"4738691759099793746170047375612500000000000000000000000000009876",
            0
        );
    }

    #[test]
    fun test_from_bytes(){
        let x = vector::empty();
        let y = vector::empty();
        vector::push_back(&mut x, 0x12);
        vector::push_back(&mut x, 0x34);
        vector::push_back(&mut x, 0x56);
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
        assert!(u == 0x99876543, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_deserialize_u64_be() {
        let cursor = cursor::new(x"1300000025000001");
        let u = bytes::deserialize_u64_be(&mut cursor);
        assert!(u == 0x1300000025000001, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_deserialize_u128_be() {
        let cursor = cursor::new(x"130209AB2500FA0113CD00AE25000001");
        let u = bytes::deserialize_u128_be(&mut cursor);
        assert!(u == 0x130209AB2500FA0113CD00AE25000001, 0);
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
