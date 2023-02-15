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
