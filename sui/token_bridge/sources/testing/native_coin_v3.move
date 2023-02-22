#[test_only]
module token_bridge::native_coin_witness_v3 {
    use std::option::{Self};
    use sui::tx_context::{TxContext, sender};
    use sui::coin::{Self};
    use sui::transfer::{Self};

    struct NATIVE_COIN_WITNESS_V3 has drop {}

    // This module creates a Sui-native token for testing purposes,
    // and subsequently transfers the token cap and coin meta to the
    // sender of the transaction
    fun init(coin_witness: NATIVE_COIN_WITNESS_V3, ctx: &mut TxContext) {
        let (treasury_cap, coin_metadata) = coin::create_currency<NATIVE_COIN_WITNESS_V3>(
            coin_witness,
            6,
            x"001234",
            x"221111",
            x"4444",
            option::none(),
            ctx
        );
        transfer::transfer(coin_metadata, sender(ctx));
        transfer::transfer(treasury_cap, sender(ctx));
    }

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(NATIVE_COIN_WITNESS_V3 {}, ctx)
    }
}
