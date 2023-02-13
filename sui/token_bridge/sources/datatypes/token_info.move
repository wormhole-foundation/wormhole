module token_bridge::token_info {
    use wormhole::external_address::{ExternalAddress};

    struct TokenInfo<phantom C> has store, copy, drop {
        chain: u16,
        addr: ExternalAddress
    }

    public fun new<C>(
        chain: u16,
        addr: ExternalAddress
    ): TokenInfo<C> {
        TokenInfo {
            chain,
            addr
        }
    }

    public fun chain<C>(self: &TokenInfo<C>): u16 {
        self.chain
    }

    public fun addr<C>(self: &TokenInfo<C>): ExternalAddress {
        self.addr
    }
}
