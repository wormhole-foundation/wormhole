module token_bridge::native_asset {
    use sui::coin::{Self, Coin};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{chain_id};

    use token_bridge::token_info::{Self, TokenInfo};

    // Needs 'deposit` and `withdraw`
    friend token_bridge::registered_tokens;

    struct NativeAsset<phantom C> has key, store {
        id: UID,
        custody: Coin<C>,
        token_address: ExternalAddress,
        decimals: u8
    }

    public fun new<C>(
        token_address: ExternalAddress,
        decimals: u8,
        ctx: &mut TxContext
    ): NativeAsset<C> {
        NativeAsset {
            id: object::new(ctx),
            custody: coin::zero(ctx),
            token_address,
            decimals
        }
    }

    public fun token_address<C>(
        self: &NativeAsset<C>
    ): ExternalAddress {
        self.token_address
    }

    public fun decimals<C>(self: &NativeAsset<C>): u8 {
        self.decimals
    }

    public fun balance<C>(self: &NativeAsset<C>): u64 {
        coin::value(&self.custody)
    }

    public fun to_token_info<C>(self: &NativeAsset<C>): TokenInfo<C> {
        token_info::new(
            false, // is_wrapped
            chain_id(),
            self.token_address
        )
    }

    public(friend) fun deposit<C>(
        self: &mut NativeAsset<C>,
        depositable: Coin<C>
    ) {
        coin::join(&mut self.custody, depositable)
    }

    public(friend) fun withdraw<C>(
        self: &mut NativeAsset<C>,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<C> {
        coin::split(&mut self.custody, amount, ctx)
    }
}
