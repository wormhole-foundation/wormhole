module wormhole::u16 {

    const MAX_U16: u64 = (1 << 16) - 1;

    const E_OVERFLOW: u64 = 0x0;

    struct U16 has key, store, copy, drop {
        number: u64
    }

    fun check_overflow(u: &U16) {
        assert!(u.number <= MAX_U16, E_OVERFLOW)
    }

    public fun from_u64(number: u64): U16 {
        let u = U16 { number };
        check_overflow(&u);
        u
    }

    public fun to_u64(u: U16): u64 {
        u.number
    }

    public fun split_u8(number: U16): (u8, u8) {
        let U16 { number } = number;
        let v0: u8 = ((number >> 8) % (0xFF + 1) as u8);
        let v1: u8 = (number        % (0xFF + 1) as u8);
        (v0, v1)
    }

    #[test]
    public fun test_split_u8() {
        let u = from_u64(0x1234);
        let (v0, v1) = split_u8(u);
        assert!(v0 == 0x12, 0);
        assert!(v1 == 0x34, 0);
    }
}
