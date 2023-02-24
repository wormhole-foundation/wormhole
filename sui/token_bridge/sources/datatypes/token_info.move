module token_bridge::token_info {
    use wormhole::external_address::{ExternalAddress};

    struct TokenInfo<phantom C> has store, copy, drop {
        is_wrapped: bool,
        chain: u16,
        addr: ExternalAddress
    }

    public fun new<C>(
        is_wrapped: bool,
        chain: u16,
        addr: ExternalAddress
    ): TokenInfo<C> {
        TokenInfo {
            is_wrapped,
            chain,
            addr
        }
    }

    public fun is_wrapped<C>(self: &TokenInfo<C>): bool {
        self.is_wrapped
    }

    public fun chain<C>(self: &TokenInfo<C>): u16 {
        self.chain
    }

    public fun addr<C>(self: &TokenInfo<C>): ExternalAddress {
        self.addr
    }

    public fun equals<C>(
        self: &TokenInfo<C>,
        chain: u16,
        addr: ExternalAddress
    ): bool {
        self.chain == chain && self.addr == addr
    }
}

#[test_only]
module token_bridge::token_info_test{
    use wormhole::external_address::{Self, get_bytes, from_bytes};

    use token_bridge::token_info::{Self, equals, is_wrapped, chain, addr};

    struct MyCoinType {}

    #[test]
    fun test_create_token_info_1(){
        let addr_bytes =
            x"0000000000000000000000000000000000000000000000000000000000110011";
        let token_info = token_info::new<MyCoinType>(
            false,
            2, // chain
            external_address::from_bytes(addr_bytes)
        );

        // Assert that created TokenInfo has correct fields.
        assert!(is_wrapped<MyCoinType>(&token_info)==false, 0);
        assert!(chain<MyCoinType>(&token_info)==2, 0);
        assert!(get_bytes(&addr<MyCoinType>(&token_info))==addr_bytes, 0);
        assert!(equals<MyCoinType>(&token_info, 2, from_bytes(addr_bytes)), 0);
    }

    #[test]
    fun test_create_token_info_2(){
        let addr_bytes =
            x"2300000000000000000000000000000000000000000000000000000000110011";
        let token_info = token_info::new<MyCoinType>(
            true,
            15, // chain
            from_bytes(addr_bytes)
        );

        // Assert that created TokenInfo has correct fields.
        assert!(is_wrapped<MyCoinType>(&token_info)==true, 0);
        assert!(chain<MyCoinType>(&token_info)==15, 0);
        assert!(get_bytes(&addr<MyCoinType>(&token_info))==addr_bytes, 0);
        assert!(equals<MyCoinType>(&token_info, 15, from_bytes(addr_bytes)), 0);
    }
}
