module token_bridge::wrapped {
    use sui::tx_context::TxContext;
    //use sui::object::{Self, UID};
    use sui::coin::{Self};
    //use sui::coin::{Self, Coin, TreasuryCap};
    //use sui::transfer::{Self};

    use token_bridge::bridge_state::{BridgeState};
    use token_bridge::vaa::{Self as token_bridge_vaa};
    use token_bridge::asset_meta::{AssetMeta, Self};
    use token_bridge::treasury::{Self};

    use wormhole::state::{State as WormholeState};
    use wormhole::myvaa::{Self as corevaa};

    // struct TreasuryCapContainer<phantom CoinType> has key, store {
    //     id: UID,
    //     t: TreasuryCap<CoinType>,
    // }

    public fun create_wrapped_coin<T: drop>(
        state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        vaa: vector<u8>,
        witness: T,
        ctx: &mut TxContext,
    ) {
        let vaa = token_bridge_vaa::parse_verify_and_replay_protect(state, bridge_state, vaa, ctx);
        let _asset_meta: AssetMeta = asset_meta::parse(corevaa::destroy(vaa));
        let treasury_cap = coin::create_currency<T>(witness, ctx);
        // TODO - assert emitter is registered, extract decimals, token name, symbol, etc. from asset meta
        // TODO - figure out where to store name, symbol, etc.
        treasury::create_treasury_cap_store<T>(bridge_state, treasury_cap, ctx);
    }
}