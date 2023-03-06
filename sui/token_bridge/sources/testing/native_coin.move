#[test_only]
module token_bridge::native_coin_witness {
    use std::option::{Self};
    use sui::tx_context::{TxContext};
    use sui::coin::{Self};
    use sui::transfer::{Self};

    struct NATIVE_COIN_WITNESS has drop {}

    /// This module creates a Sui-native token for testing purposes,
    /// for example in complete_transfer, where we create a native coin,
    /// mint some and deposit in the token bridge, then complete transfer
    /// and ultimately transfer a portion of those native coins to a recipient.
    fun init(coin_witness: NATIVE_COIN_WITNESS, ctx: &mut TxContext) {
        let (treasury_cap, coin_metadata) = coin::create_currency<NATIVE_COIN_WITNESS>(
            coin_witness,
            10,
            x"00",
            x"11",
            x"22",
            option::none(),
            ctx
        );
        transfer::share_object(coin_metadata);
        transfer::share_object(treasury_cap);
    }

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(NATIVE_COIN_WITNESS {}, ctx)
    }
}
