module token_bridge::wrapped_asset {
    use sui::coin::{Self, Coin, TreasuryCap};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};

    use token_bridge::token_info::{Self, TokenInfo};

    // For `burn` and `mint`
    friend token_bridge::state;

    /// WrappedAsset<CoinType> stores all the metadata about a wrapped asset
    struct WrappedAsset<phantom CoinType> has key, store {
        id: UID,
        token_chain: u16,
        token_address: ExternalAddress,
        treasury_cap: TreasuryCap<CoinType>,
        decimals: u8,
    }

    public fun new<CoinType>(
        token_chain: u16,
        token_address: ExternalAddress,
        treasury_cap: TreasuryCap<CoinType>,
        decimals: u8,
        ctx: &mut TxContext
    ): WrappedAsset<CoinType> {
        return WrappedAsset {
            id: object::new(ctx),
            token_chain,
            token_address,
            treasury_cap,
            decimals
        }
    }

    public fun token_chain<CoinType>(self: &WrappedAsset<CoinType>): u16 {
        self.token_chain
    }

    public fun token_address<CoinType>(
        self: &WrappedAsset<CoinType>
    ): ExternalAddress {
        self.token_address
    }

    public fun treasury_cap<CoinType>(
        self: &WrappedAsset<CoinType>
    ): &TreasuryCap<CoinType> {
        &self.treasury_cap
    }

    public fun decimals<CoinType>(self: &WrappedAsset<CoinType>): u8 {
        self.decimals
    }

    public fun to_token_info<CoinType>(
        self: &WrappedAsset<CoinType>
    ): TokenInfo<CoinType> {
        token_info::new(
            true, // is_wrapped
            self.token_chain,
            self.token_address
        )
    }

    public(friend) fun burn<CoinType>(
        self: &mut WrappedAsset<CoinType>,
        some_coin: Coin<CoinType>
    ): u64 {
        coin::burn(&mut self.treasury_cap, some_coin)
    }

    public(friend) fun mint<CoinType>(
        self: &mut WrappedAsset<CoinType>,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        coin::mint(&mut self.treasury_cap, amount, ctx)
    }
}
