module coin::unwrapped_coin {
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};

    struct UNWRAPPED_COIN has drop {}

    fun init(coin_witness: UNWRAPPED_COIN, ctx: &mut TxContext) {
        let decimals = 9;
        let symbol = x"00";
        let name = x"00";

        let (treasury_cap, coin_metadata) = coin::create_currency<CoinType>(
            coin_witness,
            decimals,
            string32::to_bytes(&symbol),
            string32::to_bytes(&name),
            x"", //empty description
            option::none<Url>(), //empty url
            ctx
        );
        transfer::transfer(
            treasury_cap,
            tx_context::sender(ctx)
        );
         transfer::transfer(
            coin_metadata,
            tx_context::sender(ctx)
        );
    }

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(UNWRAPPED_COIN {}, ctx)
    }
}
