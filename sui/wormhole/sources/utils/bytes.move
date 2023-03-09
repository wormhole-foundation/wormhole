module wormhole::bytes {
    use std::vector::{Self};
    use std::bcs::{Self};
    use wormhole::cursor::{Self, Cursor};

    public fun push_u8(buf: &mut vector<u8>, v: u8) {
        vector::push_back<u8>(buf, v);
    }

    public fun push_u16_be(buf: &mut vector<u8>, value: u16) {
        serialize_reverse(buf, value);
    }

    public fun push_u32_be(buf: &mut vector<u8>, value: u32) {
        serialize_reverse(buf, value);
    }

    public fun push_u64_be(buf: &mut vector<u8>, value: u64) {
        serialize_reverse(buf, value);
    }

    public fun push_u128_be(buf: &mut vector<u8>, value: u128) {
        serialize_reverse(buf, value);
    }

    public fun push_u256_be(buf: &mut vector<u8>, value: u256) {
        serialize_reverse(buf, value);
    }

    public fun take_u8(cur: &mut Cursor<u8>): u8 {
        cursor::poke(cur)
    }

    public fun take_u16_be(cur: &mut Cursor<u8>): u16 {
        let res: u64 = 0;
        let i = 0;
        while (i < 2) {
            let b = cursor::poke(cur);
            res = (res << 8) + (b as u64);
            i = i + 1;
        };
        (res as u16)
    }

    public fun take_u32_be(cur: &mut Cursor<u8>): u32 {
        let res: u64 = 0;
        let i = 0;
        while (i < 4) {
            let b = cursor::poke(cur);
            res = (res << 8) + (b as u64);
            i = i + 1;
        };
        (res as u32)
    }

    public fun take_u64_be(cur: &mut Cursor<u8>): u64 {
        let res: u64 = 0;
        let i = 0;
        while (i < 8) {
            let b = cursor::poke(cur);
            res = (res << 8) + (b as u64);
            i = i + 1;
        };
        res
    }

    public fun take_u128_be(cur: &mut Cursor<u8>): u128 {
        let res: u128 = 0;
        let i = 0;
        while (i < 16) {
            let b = cursor::poke(cur);
            res = (res << 8) + (b as u128);
            i = i + 1;
        };
        res
    }

    public fun take_u256_be(cur: &mut Cursor<u8>): u256 {
        let res: u256 = 0;
        let i = 0;
        while (i < 32) {
            let b = cursor::poke(cur);
            res = (res << 8) + (b as u256);
            i = i + 1;
        };
        res
    }

    public fun take_bytes(cur: &mut Cursor<u8>, num_bytes: u64): vector<u8> {
        let result = vector::empty();
        let i = 0;
        while (i < num_bytes) {
            vector::push_back(&mut result, cursor::poke(cur));
            i = i + 1;
        };
        result
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
}

#[test_only]
module wormhole::bytes_tests {
    use std::vector::{Self};
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};

    #[test]
    fun test_push_u8(){
        let u = 0x12;
        let s = vector::empty();
        bytes::push_u8(&mut s, u);
        let cur = cursor::new(s);
        let p = bytes::take_u8(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_push_u16_be(){
        let u = 0x1234;
        let s = vector::empty();
        bytes::push_u16_be(&mut s, u);
        let cur = cursor::new(s);
        let p = bytes::take_u16_be(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_push_u32_be(){
        let u = 0x12345678;
        let s = vector::empty();
        bytes::push_u32_be(&mut s, u);
        let cur = cursor::new(s);
        let p = bytes::take_u32_be(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_push_u64_be(){
        let u = 0x1234567812345678;
        let s = vector::empty();
        bytes::push_u64_be(&mut s, u);
        let cur = cursor::new(s);
        let p = bytes::take_u64_be(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

     #[test]
    fun test_push_u128_be(){
        let u = 0x12345678123456781234567812345678;
        let s = vector::empty();
        bytes::push_u128_be(&mut s, u);
        let cur = cursor::new(s);
        let p = bytes::take_u128_be(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_push_u256_be(){
        let u =
            0x4738691759099793746170047375612500000000000000000000000000009876;
        let s = vector::empty();
        bytes::push_u256_be(&mut s, u);
        assert!(
            s == x"4738691759099793746170047375612500000000000000000000000000009876",
            0
        );
    }

    #[test]
    fun test_take_u8() {
        let cursor = cursor::new(x"99");
        let byte = bytes::take_u8(&mut cursor);
        assert!(byte==0x99, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_take_u16_be() {
        let cursor = cursor::new(x"9987");
        let u = bytes::take_u16_be(&mut cursor);
        assert!(u == 0x9987, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_take_u32_be() {
        let cursor = cursor::new(x"99876543");
        let u = bytes::take_u32_be(&mut cursor);
        assert!(u == 0x99876543, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_take_u64_be() {
        let cursor = cursor::new(x"1300000025000001");
        let u = bytes::take_u64_be(&mut cursor);
        assert!(u == 0x1300000025000001, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_take_u128_be() {
        let cursor = cursor::new(x"130209AB2500FA0113CD00AE25000001");
        let u = bytes::take_u128_be(&mut cursor);
        assert!(u == 0x130209AB2500FA0113CD00AE25000001, 0);
        cursor::destroy_empty(cursor);
    }

    #[test]
    fun test_to_bytes() {
        let cursor = cursor::new(b"hello world");
        let hello = bytes::take_bytes(&mut cursor, 5);
        bytes::take_u8(&mut cursor);
        let world = bytes::take_bytes(&mut cursor, 5);
        assert!(hello == b"hello", 0);
        assert!(world == b"world", 0);
        cursor::destroy_empty(cursor);
    }

}
