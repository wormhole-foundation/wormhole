module wrapped_coin::coin {
    use sui::object::{Self};
    use sui::package::{Self};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use token_bridge::create_wrapped::{Self};

    struct COIN has drop {}

    fun init(witness: COIN, ctx: &mut TxContext) {
        use token_bridge::version_control::{V__0_1_1};

        transfer::public_transfer(
            create_wrapped::prepare_registration<COIN, V__0_1_1>(
                witness,
                {{DECIMALS}},
                ctx
            ),
            tx_context::sender(ctx)
        );
    }
}
