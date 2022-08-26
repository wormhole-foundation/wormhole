module Wormhole::Uints{
    use 0x1::vector;

    struct U16 has key, store, copy, drop{
        inner: vector<u8>,
    }

    struct U32 has key, store, copy, drop{
        inner: vector<u8>,
    }

    struct U256 has key, store, copy, drop{
        inner: vector<u8>,
    }

    public fun get_bytes_array_u16(x: U16): vector<u8>{
        x.inner
    }

    public fun get_bytes_array_u32(x: U32): vector<u8>{
        x.inner
    }
    
    public fun get_bytes_array_u256(x: U256): vector<u8>{
        x.inner
    }

    public fun into_u16(x: vector<u8>): U16{
        assert!(vector::length<u8>(&x)==2, 0);
        U16 {inner: x}
    }
    public fun into_u32(x: vector<u8>): U32{
        assert!(vector::length<u8>(&x)==4, 0);
        U32 {inner: x}
    }
    public fun into_u256(x: vector<u8>): U256{
        assert!(vector::length<u8>(&x)==32, 0);
        U256 {inner: x}
    }

    public fun zero_u16(): U16{
        let x = vector::empty();
        let i=0;
        loop { 
            if (i==2){
                break
            };
            vector::push_back(&mut x, 0x00);
        };
        into_u16(x)
    }

    public fun zero_u32(): U32{
        let x = vector::empty();
        let i=0;
        loop { 
            if (i==4){
                break
            };
            vector::push_back(&mut x, 0x00);
        };
        into_u32(x)
    }

    public fun zero_u256(): U256{
        let x = vector::empty();
        let i=0;
        loop { 
            if (i==4){
                break
            };
            vector::push_back(&mut x, 0x00);
        };
        into_u256(x)
    }

    // TODO: addition and comparison ops
    // U32 addition

}