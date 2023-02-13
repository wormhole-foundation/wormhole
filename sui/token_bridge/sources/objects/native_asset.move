module token_bridge::native_asset {
    use sui::coin::{Self, Coin};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};

    // Needs 'deposit` and `withdraw`
    friend token_bridge::state;

    struct NativeAsset<phantom CoinType> has key, store {
        id: UID,
        custody: Coin<CoinType>,
        // Even though we can look up token_chain at any time from wormhole
        // state, it can be more efficient to store it here locally so we don't
        // have to do lookups.
        token_chain: u16,
        token_address: ExternalAddress,
        decimals: u8
    }

    public fun new<CoinType>(
        token_chain: u16,
        token_address: ExternalAddress,
        decimals: u8,
        ctx: &mut TxContext
    ): NativeAsset<CoinType> {
        NativeAsset {
            id: object::new(ctx),
            custody: coin::zero(ctx),
            token_chain,
            token_address,
            decimals
        }
    }

    public fun token_chain<CoinType>(self: &NativeAsset<CoinType>): u16 {
        self.token_chain
    }

    public fun token_address<CoinType>(
        self: &NativeAsset<CoinType>
    ): ExternalAddress {
        self.token_address
    }
    
    public fun decimals<CoinType>(self: &NativeAsset<CoinType>): u8 {
        self.decimals
    }

    public(friend) fun deposit<CoinType>(
        self: &mut NativeAsset<CoinType>,
        some_coin: Coin<CoinType>
    ) {
        coin::join(&mut self.custody, some_coin)
    }

    public(friend) fun withdraw<CoinType>(
        self: &mut NativeAsset<CoinType>,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        coin::split(&mut self.custody, amount, ctx)
    }

    public fun balance<CoinType>(
        self: &NativeAsset<CoinType>
    ): u64 {
        coin::value(&self.custody)
    }
}
