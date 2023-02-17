module token_bridge::wrapped_asset {
    use sui::coin::{Self, Coin, TreasuryCap};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};

    use token_bridge::token_info::{Self, TokenInfo};

    // For `burn` and `mint`
    friend token_bridge::registered_tokens;

    /// WrappedAsset<C> stores all the metadata about a wrapped asset
    struct WrappedAsset<phantom C> has store {
        token_chain: u16,
        token_address: ExternalAddress,
        treasury_cap: TreasuryCap<C>,
        decimals: u8,
    }

    public fun new<C>(
        token_chain: u16,
        token_address: ExternalAddress,
        treasury_cap: TreasuryCap<C>,
        decimals: u8,
    ): WrappedAsset<C> {
        return WrappedAsset {
            token_chain,
            token_address,
            treasury_cap,
            decimals
        }
    }

    public fun token_chain<C>(self: &WrappedAsset<C>): u16 {
        self.token_chain
    }

    public fun token_address<C>(self: &WrappedAsset<C>): ExternalAddress {
        self.token_address
    }

    public fun treasury_cap<C>(self: &WrappedAsset<C>): &TreasuryCap<C> {
        &self.treasury_cap
    }

    public fun decimals<C>(self: &WrappedAsset<C>): u8 {
        self.decimals
    }

    public fun to_token_info<C>(self: &WrappedAsset<C>): TokenInfo<C> {
        token_info::new(
            true, // is_wrapped
            self.token_chain,
            self.token_address
        )
    }

    public(friend) fun burn<C>(
        self: &mut WrappedAsset<C>,
        burnable: Coin<C>
    ): u64 {
        coin::burn(&mut self.treasury_cap, burnable)
    }

    public(friend) fun mint<C>(
        self: &mut WrappedAsset<C>,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<C> {
        coin::mint(&mut self.treasury_cap, amount, ctx)
    }
}
