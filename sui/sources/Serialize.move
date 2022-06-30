module Wormhole::Serialize {
    use Sui::Vector;

    public fun serialize_u8(buf: &mut vector<u8>, v: u8) {
        Vector::push_back(&mut buf, v);
    }

    public fun serialize_u16(buf: &mut vector<u8>, v: u16) {
        serialize_u8(buf, (v >> 8) as u8);
        serialize_u8(buf, v as u8);
    }

    public fun serialize_u32(buf: &mut vector<u8>, v: u32) {
        serialize_u8(buf, (v >> 24) as u8);
        serialize_u8(buf, (v >> 16) as u8);
        serialize_u8(buf, (v >> 8) as u8);
        serialize_u8(buf, v as u8);
    }

    public fun serialize_u64(buf: &mut vector<u8>, v: u64) {
        serialize_u8(buf, (v >> 56) as u8);
        serialize_u8(buf, (v >> 48) as u8);
        serialize_u8(buf, (v >> 40) as u8);
        serialize_u8(buf, (v >> 32) as u8);
        serialize_u8(buf, (v >> 24) as u8);
        serialize_u8(buf, (v >> 16) as u8);
        serialize_u8(buf, (v >> 8) as u8);
        serialize_u8(buf, v as u8);
    }

    public fun serialize_u128(buf: &mut vector<u8>, v: u128) {
        serialize_u64(buf, (v >> 64) as u64);
        serialize_u64(buf, v as u64);
    }
}

