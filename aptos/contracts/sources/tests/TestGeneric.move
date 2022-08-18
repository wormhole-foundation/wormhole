module Wormhole::TestGeneric{
    use 0x1::coin::{Self, MintCapability, BurnCapability};
    use 0x1::signer::{Self};
    use 0x1::string::{Self, utf8};

    struct MyCoinType has key, drop { type: u8}
    
    struct Caps<phantom CoinType> has key {
        mint: MintCapability<CoinType>,
        burn: BurnCapability<CoinType>,
    }   

    #[test(admin=@0x123)]
    public entry fun test(admin: &signer){
        let t1 = MyCoinType {type: 1};
        let t2 = MyCoinType {type: 2};

        let (a, b) = coin::initialize<t1>(admin, utf8(b"coin1"), utf8(b"symbol1"), 12, false);
        let (c, d) = coin::initialize<t2>(admin, utf8(b"coin2"), utf8(b"symbol2"), 13, false);
        move_to(admin, Caps<t1> { mint: a, burn: b });
        move_to(admin, Caps<t2> { mint: c, burn: d });

    }

        // account: &signer,
        // name: string::String,
        // symbol: string::String,
        // decimals: u64,
        // monitor_supply: bool,

}