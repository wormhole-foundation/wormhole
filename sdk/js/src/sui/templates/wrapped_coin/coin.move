module wrapped_coin::coin {
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use token_bridge::create_wrapped::{Self};

    struct COIN has drop {}

    const VAA: vector<u8> = x"{{VAA_BYTES}}";

    fun init(witness: COIN, ctx: &mut TxContext) {
        transfer::public_transfer(
            create_wrapped::prepare_registration(
                witness,
                VAA,
                ctx
            ),
            tx_context::sender(ctx)
        );
    }
}
