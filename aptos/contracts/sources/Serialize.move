module Wormhole::Serialize {
    use 0x1::vector::{Self};
    use Wormhole::u16::{Self, U16};
    use Wormhole::u32::{Self, U32};
    use Wormhole::u256::{Self, U256};

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
        serialize_u8(buf, ((v >> 56) as u8));
        serialize_u8(buf, ((v >> 48) % (2<<8) as u8));
        serialize_u8(buf, ((v >> 40) % (2<<8) as u8));
        serialize_u8(buf, ((v >> 32) % (2<<8) as u8));
        serialize_u8(buf, ((v >> 24) % (2<<8) as u8));
        serialize_u8(buf, ((v >> 16) % (2<<8) as u8));
        serialize_u8(buf, ((v >> 8) % (2<<8) as u8));
        serialize_u8(buf, ((v % (2<<8)) as u8))
    }

    public fun serialize_u128(buf: &mut vector<u8>, v: u128) {
        serialize_u64(buf, ((v >> 64) as u64));
        serialize_u64(buf, ((v % 2<<64) as u64))
    }

    public fun serialize_u256(buf: &mut vector<u8>, v: U256) {
        serialize_u64(buf, u256::get(&v, 0));
        serialize_u64(buf, u256::get(&v, 1));
        serialize_u64(buf, u256::get(&v, 2));
        serialize_u64(buf, u256::get(&v, 3));
    }

    public fun serialize_vector(buf: &mut vector<u8>, v: vector<u8>){
        vector::reverse<u8>(&mut v);
        let len = vector::length<u8>(&mut v);
        while ({
            spec {
                invariant len >  0;
            };
            len > 0
        }) {
            let byte = vector::pop_back(&mut v);
            vector::push_back(buf, byte);
        }
    }
}

#[test_only]
module Wormhole::TestSerialize{
    use Wormhole::Serialize;
    use Wormhole::Deserialize;
    use Wormhole::cursor::{Self};
    use 0x1::vector;

    #[test]
    fun test_one(){
        let x = vector::empty();
        vector::push_back<u8>(&mut x, 0x12);
        vector::push_back<u8>(&mut x, 0x34);
        vector::push_back<u8>(&mut x, 0x56);
        vector::push_back<u8>(&mut x, 0x78);
        vector::push_back<u8>(&mut x, 0x12);
        vector::push_back<u8>(&mut x, 0x34);
        vector::push_back<u8>(&mut x, 0x56);
        vector::push_back<u8>(&mut x, 0x78);
        let cur = cursor::init(x);
        let u = Deserialize::deserialize_u64(&mut cur);
        assert!(u==0x1234567812345678, 0);
        cursor::destroy_empty(cur);

        // serialize then deserialize test
        let s = vector::empty();
        Serialize::serialize_u64(&mut s, u);
        let cur = cursor::init(s);
        let p = Deserialize::deserialize_u64(&mut cur);
        cursor::destroy_empty(cur);
        assert!(p==u, 0);
    }
}
