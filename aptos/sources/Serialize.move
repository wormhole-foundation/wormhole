module Wormhole::Serialize {
    
    use Std::Vector;

    public fun serialize_u8(buf: &mut vector<u8>, v: u8) {
        Vector::push_back<u8>(buf, v);
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

    public fun serialize_vector(buf: &mut vector<u8>, v: vector<u8>){
        Vector::reverse<u8>(&mut v); 
        let len = Vector::length<u8>(&mut v);
        while ({
            spec { 
                invariant len >  0;
            };
            len > 0
        }) {
            let byte = Vector::pop_back(&mut v);
            Vector::push_back(buf, byte);
        }
    }

}

#[test_only]
module Wormhole::TestSerialize{
    use Wormhole::Serialize;
    use Wormhole::Deserialize;
    use Std::Vector;

    #[test]
    fun test_one(){
        let x = Vector::empty();
        Vector::push_back<u8>(&mut x, 0x12);
        Vector::push_back<u8>(&mut x, 0x34);
        Vector::push_back<u8>(&mut x, 0x56);
        Vector::push_back<u8>(&mut x, 0x78);
        Vector::push_back<u8>(&mut x, 0x12);
        Vector::push_back<u8>(&mut x, 0x34);
        Vector::push_back<u8>(&mut x, 0x56);
        Vector::push_back<u8>(&mut x, 0x78);
        let (u, _) = Deserialize::deserialize_u64(x);
        assert!(u==0x1234567812345678, 0);
        
        // serialize then deserialize test
        let s = Vector::empty();
        Serialize::serialize_u64(&mut s, u);
        let (p, _) = Deserialize::deserialize_u64(s);
        assert!(p==u, 0);
    }
}
