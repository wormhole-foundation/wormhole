module wormhole::u32 {

    const MAX_U32: u64 = (1 << 32) - 1;

    const E_OVERFLOW: u64 = 0x0;

    struct U32 has key, store, copy, drop {
        number: u64
    }

    fun check_overflow(u: &U32) {
        assert!(u.number <= MAX_U32, E_OVERFLOW)
    }

    public fun from_u64(number: u64): U32 {
        let u = U32 { number };
        check_overflow(&u);
        u
    }

    public fun to_u64(u: U32): u64 {
        u.number
    }

    public fun split_u8(number: U32): (u8, u8, u8, u8) {
        let U32 { number } = number;
        let v0: u8 = ((number >> 24) % (0xFF + 1) as u8);
        let v1: u8 = ((number >> 16) % (0xFF + 1) as u8);
        let v2: u8 = ((number >> 8)  % (0xFF + 1) as u8);
        let v3: u8 = (number         % (0xFF + 1) as u8);
        (v0, v1, v2, v3)
    }

    #[test]
    public fun test_split_u8() {
        let u = from_u64(0x12345678);
        let (v0, v1, v2, v3) = split_u8(u);
        assert!(v0 == 0x12, 0);
        assert!(v1 == 0x34, 0);
        assert!(v2 == 0x56, 0);
        assert!(v3 == 0x78, 0);
    }
}
