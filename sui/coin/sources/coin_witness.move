module coin::coin_witness {
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};
    use sui::coin::Self;

    use token_bridge::asset_meta::{Self, AssetMeta, get_decimals};

    use wormhole::myvaa::{parse_and_get_payload};

    struct COIN_WITNESS has drop {}

    fun init(coin_witness: COIN_WITNESS, ctx: &mut TxContext) {
        // Step 1. Paste token attestation VAA below
        let vaa_bytes = x"0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";

        let payload = parse_and_get_payload(vaa_bytes);
        let asset_meta: AssetMeta = asset_meta::parse(payload);
        let decimals = get_decimals(&asset_meta);
        let treasury_cap = coin::create_currency<COIN_WITNESS>(coin_witness, decimals, ctx);
        transfer::transfer(
            treasury_cap,
            tx_context::sender(ctx)
        );
    }

    //// Note: one-time callable function approach
    //// this design doesn't work!

    // public entry fun create_wrapped(
    //     worm_state: &mut State,
    //     token_state: &mut BridgeState,
    //     //witness_carrier: WitnessCarrier,
    //     ctx: &mut TxContext
    // ) {
    //     // Make sure to put your VAA here!
    //     // Alternatively we can make this
    //     let vaa = x"deadbeef00001231231231";

    //     create_wrapped_coin<COIN_WITNESS>(
    //         worm_state,
    //         token_state,
    //         //get_witness(witness_carrier),
    //         COIN_WITNESS {},
    //         vaa,
    //         ctx
    //     )
    // }
}
