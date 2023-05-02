module coins::coin_8 {
    use std::option::{Self};
    use sui::coin::{Self, TreasuryCap, CoinMetadata};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    /// The type identifier of coin. The coin will have a type
    /// tag of kind: `Coin<package_object::coin_8::COIN_8>`
    /// Make sure that the name of the type matches the module's name.
    struct COIN_8 has drop {}

    /// Module initializer is called once on module publish. A treasury
    /// cap is sent to the publisher, who then controls minting and burning
    fun init(witness: COIN_8, ctx: &mut TxContext) {
        let (treasury, metadata) = create_coin(witness, ctx);
        transfer::public_freeze_object(metadata);
        transfer::public_transfer(treasury, tx_context::sender(ctx));
    }

    fun create_coin(
        witness: COIN_8,
        ctx: &mut TxContext
    ): (TreasuryCap<COIN_8>, CoinMetadata<COIN_8>) {
        coin::create_currency(
            witness,
            8, // decimals
            b"COIN_8", // symbol
            b"8-Decimal Coin", // name
            b"", // description
            option::none(), // icon_url
            ctx
        )
    }

    #[test_only]
    public fun create_coin_test_only(
        ctx: &mut TxContext
    ): (TreasuryCap<COIN_8>, CoinMetadata<COIN_8>) {
        create_coin(COIN_8 {}, ctx)
    }

    #[test_only]
    public fun init_test_only(ctx: &mut TxContext) {
        init(COIN_8 {}, ctx)
    }
}

#[test_only]
module coins::coin_8_tests {
    use sui::test_scenario::{Self};

    use coins::coin_8::{Self};

    #[test]
    public fun init_test() {
        let my_scenario = test_scenario::begin(@0x0);
        let scenario = &mut my_scenario;
        let creator = @0xDEADBEEF;

        // Proceed.
        test_scenario::next_tx(scenario, creator);

        // Init.
        coin_8::init_test_only(test_scenario::ctx(scenario));

        // Proceed.
        test_scenario::next_tx(scenario, creator);

        // Done.
        test_scenario::end(my_scenario);
    }
}
