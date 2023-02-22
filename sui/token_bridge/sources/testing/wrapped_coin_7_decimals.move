#[test_only]
module token_bridge::wrapped_coin_7_decimals {
    use std::option::{Self};
    use sui::tx_context::{TxContext, sender};
    use sui::coin::{Self};
    use sui::transfer::{Self};

    struct WRAPPED_COIN_7_DECIMALS has drop {}

    // This module creates a Sui-native token for testing purposes,
    // and subsequently transfers the token cap and coin meta to the
    // sender of the transaction
    fun init(coin_witness: WRAPPED_COIN_7_DECIMALS, ctx: &mut TxContext) {
        let (treasury_cap, coin_metadata) = coin::create_currency<WRAPPED_COIN_7_DECIMALS>(
            coin_witness,
            7,
            b"7777", // symbol
            b"seven decimals wow", // name
            b"I have seven decimals", // description
            option::none(),
            ctx
        );
        transfer::transfer(coin_metadata, sender(ctx));
        transfer::transfer(treasury_cap, sender(ctx));
    }

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(WRAPPED_COIN_7_DECIMALS {}, ctx)
    }
}
