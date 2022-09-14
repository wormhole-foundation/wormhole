module wormhole::serialize {
    use 0x1::vector::{Self};
    use wormhole::u16::{Self, U16};
    use wormhole::u32::{Self, U32};
    use wormhole::u256::{Self, U256};

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
        serialize_u8(buf, (v >> 56 & 0xFF as u8));
        serialize_u8(buf, (v >> 48 & 0xFF as u8));
        serialize_u8(buf, (v >> 40 & 0xFF as u8));
        serialize_u8(buf, (v >> 32 & 0xFF as u8));
        serialize_u8(buf, (v >> 24 & 0xFF as u8));
        serialize_u8(buf, (v >> 16 & 0xFF as u8));
        serialize_u8(buf, (v >> 8  & 0xFF as u8));
        serialize_u8(buf, (v       & 0xFF as u8))
    }

    public fun serialize_u128(buf: &mut vector<u8>, v: u128) {
        serialize_u64(buf, (v >> 64 & 0xFFFFFFFFFFFFFFFF as u64));
        serialize_u64(buf, (v       & 0xFFFFFFFFFFFFFFFF as u64));
    }

    public fun serialize_u256(buf: &mut vector<u8>, v: U256) {
        serialize_u64(buf, u256::get(&v, 3));
        serialize_u64(buf, u256::get(&v, 2));
        serialize_u64(buf, u256::get(&v, 1));
        serialize_u64(buf, u256::get(&v, 0));
    }

    public fun serialize_vector(buf: &mut vector<u8>, v: vector<u8>){
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
    use 0x1::vector;

    #[test]
    fun test_serialize_u8(){
        let u = 0x12;
        let s = vector::empty();
        serialize::serialize_u8(&mut s, u);
        let cur = cursor::init(s);
        let p = deserialize::deserialize_u8(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u16(){
        let u = u16::from_u64((0x1234 as u64));
        let s = vector::empty();
        serialize::serialize_u16(&mut s, u);
        let cur = cursor::init(s);
        let p = deserialize::deserialize_u16(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u32(){
        let u = u32::from_u64((0x12345678 as u64));
        let s = vector::empty();
        serialize::serialize_u32(&mut s, u);
        let cur = cursor::init(s);
        let p = deserialize::deserialize_u32(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u64(){
        let u = 0x1234567812345678;
        let s = vector::empty();
        serialize::serialize_u64(&mut s, u);
        let cur = cursor::init(s);
        let p = deserialize::deserialize_u64(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

     #[test]
    fun test_serialize_u128(){
        let u = 0x12345678123456781234567812345678;
        let s = vector::empty();
        serialize::serialize_u128(&mut s, u);
        let cur = cursor::init(s);
        let p = deserialize::deserialize_u128(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }

    #[test]
    fun test_serialize_u256(){
        let u = u256::add(u256::shl(u256::from_u128(0x12345678123456781234567812345678), 128), u256::from_u128(0x9876));
        let s = vector::empty();
        serialize::serialize_u256(&mut s, u);
        let exp = x"1234567812345678123456781234567800000000000000000000000000009876";
        assert!(s == exp, 0);
    }

    #[test]
    fun test_serialize_vector(){
        let x = vector::empty<u8>();
        let y = vector::empty<u8>();
        vector::push_back<u8>(&mut x, 0x12);
        vector::push_back<u8>(&mut x, 0x34);
        vector::push_back<u8>(&mut x, 0x56);
        serialize::serialize_vector(&mut y, x);
        assert!(y == x"123456", 0);
    }
}
