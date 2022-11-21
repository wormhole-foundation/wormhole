module coin::coin_witness {
    use sui::transfer;
    use sui::object::{Self, UID};
    use sui::tx_context::{Self, TxContext};
    //use token_bridge::wrapped::create_wrapped_coin;

    //struct COIN_WITNESS has drop, store {}

    // publish this module to call into token_bridge and create a wrapped coin
    //fun init(ctx: &mut TxContext) {
        // paste the VAA below
        //let vaa = x"deadbeef00001231231231";
        //create_wrapped_coin(
        //     b"0x9404271a20a3f22d61300f23503f276d4b42cb02",
        //     b"0xd43fabcfa47c7f9a33ff3c88b32e707c701c1eb8",
        //     vaa,
        //     COIN_WITNESS {},
        //     ctx
        // )
        //transfer::transfer(COIN_WITNESS {}, tx_context::sender(ctx));
    //}

    /// Witness now has a `store` that allows us to store it inside a wrapper.
    struct WITNESS has store, drop {}

    /// Carries the witness type. Can be used only once to get a Witness.
    struct WitnessCarrier has key { id: UID, witness: WITNESS }

    /// Send a `WitnessCarrier` to the module publisher.
    fun init(ctx: &mut TxContext) {
        transfer::transfer(
            WitnessCarrier { id: object::new(ctx), witness: WITNESS {} },
            tx_context::sender(ctx)
        )
    }

    /// Unwrap a carrier and get the inner WITNESS type.
    public fun get_witness(carrier: WitnessCarrier): WITNESS {
        let WitnessCarrier { id, witness } = carrier;
        object::delete(id);
        witness
    }
}