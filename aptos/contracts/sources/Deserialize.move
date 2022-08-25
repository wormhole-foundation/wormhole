
module Wormhole::Deserialize {
    use 0x1::vector::{Self};
    use Wormhole::cursor::{Self, Cursor};

    public fun deserialize_u8(cur: &mut Cursor<u8>): u8 {
        cursor::poke(cur)
    }

    public fun deserialize_u64(cur: &mut Cursor<u8>): u64 {
        let res: u64 = 0;
        let i = 0;
        while (i < 8) {
            let b = cursor::poke(cur);
            res = (res << 8) | (b as u64);
            i = i + 1;
        };
        res
    }

    public fun deserialize_u128(cur: &mut Cursor<u8>): u128 {
        let res: u128 = 0;
        let i = 0;
        while (i < 16) {
            let b = cursor::poke(cur);
            res = (res << 8) | (b as u128);
            i = i + 1;
        };
        res
    }

    public fun deserialize_vector(cur: &mut Cursor<u8>, len: u64): vector<u8> {
        let result = vector::empty();
        while ({
            spec {
                invariant len >= 0;
                invariant len <  vector::length(bytes);
            };
            len > 0
        }) {
            vector::push_back(&mut result, cursor::poke(cur));
            len = len - 1;
        };
        result
    }
}

#[test_only]
module Wormhole::TestDeserialize{
    use 0x1::vector::{push_back, empty};//, //length};
    use Wormhole::Deserialize::{deserialize_u8, deserialize_u64, deserialize_vector};
    use Wormhole::cursor::{Self};

    #[test]
    fun test_one(){
        // test deserialize u8 vector
        let x = empty();
        push_back(&mut x, 0x99);
        let cur = cursor::init(x);
        let byte = deserialize_u8(&mut cur);
        assert!(byte==0x99, 0);
        cursor::destroy_empty(cur);

        // deserialize u64 vector
        let v = empty();
        push_back(&mut v, 0x13);
        push_back(&mut v, 0x00);
        push_back(&mut v, 0x00);
        push_back(&mut v, 0x00);
        push_back(&mut v, 0x25);
        push_back(&mut v, 0x00);
        push_back(&mut v, 0x00);
        push_back(&mut v, 0x01);
        let cur = cursor::init(v);
        let u = deserialize_u64(&mut cur);
        assert!(u==(0x1300000025000001 as u64), 0);
        let p = deserialize_vector(&mut cur, 8);
        let q = deserialize_u64(&mut cur);
        cursor::destroy_empty(cur);
        assert!(q==(u as u64), 0);
    }
}
