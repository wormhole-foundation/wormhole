
module Wormhole::Deserialize {
    //use Std::Option;
    use 0x1::vector::{Self};

    public fun deserialize_u8(bytes: vector<u8>): (u8, vector<u8>){
        assert!(vector::length<u8>(&mut bytes)==1, 0);
        let byte = vector::pop_back(&mut bytes);
        (byte, bytes)
    }

    public fun deserialize_u64(bytes: vector<u8>): (u64, vector<u8>){
        let res = (0 as u64);
        let i = 0; 
        vector::reverse<u8>(&mut bytes);
        loop { 
            if (i==8){
                break
            };
            let cur = vector::pop_back<u8>(&mut bytes);
            res = res | (cur as u64) << (56 - i * 8);
            i=i+1;
        };
        (res, bytes)
    }

    public fun deserialize_vector(bytes: vector<u8>, len: u64): (vector<u8>, vector<u8>) {
        let result = vector::empty();
        while ({ 
            spec { 
                invariant len >= 0;
                invariant len <  vector::length(bytes);
            }; 
            len > 0 
        }) { 
            let byte = vector::pop_back(&mut bytes);
            vector::push_back(&mut result, byte);
            len = len - 1;
        };
        vector::reverse<u8>(&mut result);
        (result, bytes)
    }
}

#[test_only]
module Wormhole::TestDeserialize{
    use 0x1::vector::{push_back, empty};//, //length};
    use Wormhole::Deserialize::{deserialize_u8, deserialize_u64, deserialize_vector};

    #[test]
    fun test_one(){
        // test deserialize u8 vector
        let x = empty(); 
        push_back(&mut x, 0x99);
        let (byte, _) = deserialize_u8(x);
        assert!(byte==0x99, 0);

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
        let (u, _) = deserialize_u64(v);
        assert!(u==(0x1300000025000001 as u64), 0);
        let (p, _) = deserialize_vector(v, 8);
        let (q, _) = deserialize_u64(p);
        assert!(q==(u as u64), 0);
    }
}