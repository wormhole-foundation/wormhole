module coin::coin {
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};

    use token_bridge::create_wrapped;

    struct COIN has drop {}

    fun init(coin_witness: COIN, ctx: &mut TxContext) {
        // Step 1. Paste token attestation VAA below.
        let vaa_bytes =
            x"0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";

        let new_wrapped_coin =
            create_wrapped::create_unregistered_currency(
                vaa_bytes,
                coin_witness,
                ctx
            );
        transfer::transfer(new_wrapped_coin, tx_context::sender(ctx));
    }

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(COIN {}, ctx)
    }
}
