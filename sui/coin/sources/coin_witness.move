module coin::coin_witness {
    //use sui::transfer;
    //use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    // use token_bridge::wrapped::create_wrapped_coin

    struct COIN_WITNESS has drop {}

    // publish this module to call into token_bridge and create a wrapped coin
    fun init(_ctx: &mut TxContext) {
        // paste the VAA below
        let _vaa = x"deadbeef00001231231231";
        // create_wrapped_coin(vaa, COIN_WITNESS {})
    }
}