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
    use sui::test_scenario::{Self, Scenario, next_tx};

    use wormhole::external_address::{Self};

    use token_bridge::token_info::{Self};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    struct MyCoinType {}

    #[test]
    fun test_create_token_info_1(){
        let test = scenario();
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            let addr_bytes = x"0000000000000000000000000000000000000000000000000000000000110011";
            let token_info = token_info::new<MyCoinType>(false, 2, external_address::from_bytes(addr_bytes));
            assert!(token_info::is_wrapped<MyCoinType>(&token_info)==false, 0);
            assert!(token_info::chain<MyCoinType>(&token_info)==2, 0);
            assert!(external_address::get_bytes(&token_info::addr<MyCoinType>(&token_info))==addr_bytes, 0);
            assert!(token_info::equals<MyCoinType>(&token_info, 2, external_address::from_bytes(addr_bytes)), 0);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_create_token_info_2(){
        let test = scenario();
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            let addr_bytes = x"2300000000000000000000000000000000000000000000000000000000110011";
            let token_info = token_info::new<MyCoinType>(true, 155, external_address::from_bytes(addr_bytes));
            assert!(token_info::is_wrapped<MyCoinType>(&token_info)==true, 0);
            assert!(token_info::chain<MyCoinType>(&token_info)==155, 0);
            assert!(external_address::get_bytes(&token_info::addr<MyCoinType>(&token_info))==addr_bytes, 0);
            assert!(token_info::equals<MyCoinType>(&token_info, 155, external_address::from_bytes(addr_bytes)), 0);
        };
        test_scenario::end(test);
    }
}
