module token_bridge::native_asset {
    use sui::coin::{Self, Coin};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};

    use token_bridge::token_info::{Self, TokenInfo};

    // Needs 'deposit` and `withdraw`
    friend token_bridge::registered_tokens;

    struct NativeAsset<phantom C> has key, store {
        id: UID,
        custody: Coin<C>,
        // Even though we can look up token_chain at any time from wormhole
        // state, it can be more efficient to store it here locally so we don't
        // have to do lookups.
        token_chain: u16,
        token_address: ExternalAddress,
        decimals: u8
    }

    public fun new<C>(
        token_chain: u16,
        token_address: ExternalAddress,
        decimals: u8,
        ctx: &mut TxContext
    ): NativeAsset<C> {
        NativeAsset {
            id: object::new(ctx),
            custody: coin::zero(ctx),
            token_chain,
            token_address,
            decimals
        }
    }

    public fun token_chain<C>(self: &NativeAsset<C>): u16 {
        self.token_chain
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
            self.token_chain,
            self.token_address
        )
    }

    public(friend) fun deposit<C>(
        self: &mut NativeAsset<C>,
        some_coin: Coin<C>
    ) {
        coin::join(&mut self.custody, some_coin)
    }

    public(friend) fun withdraw<C>(
        self: &mut NativeAsset<C>,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<C> {
        coin::split(&mut self.custody, amount, ctx)
    }
}
