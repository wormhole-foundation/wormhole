module token_bridge::wrapped {
    use sui::tx_context::TxContext;
    //use sui::object::{Self, UID};
    use sui::coin::{Self};
    //use sui::coin::{Self, Coin, TreasuryCap};
    //use sui::transfer::{Self};

    use token_bridge::bridge_state::{Self, BridgeState};
    use token_bridge::vaa::{Self as token_bridge_vaa};
    use token_bridge::asset_meta::{AssetMeta, Self, get_decimals};
    use token_bridge::treasury::{Self};

    use wormhole::state::{State as WormholeState};
    use wormhole::myvaa::{Self as corevaa};

    // struct TreasuryCapContainer<phantom CoinType> has key, store {
    //     id: UID,
    //     t: TreasuryCap<CoinType>,
    // }

    public entry fun create_wrapped_coin<T: drop>(
        state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        vaa: vector<u8>,
        witness: T,
        ctx: &mut TxContext,
    ) {
        let vaa = token_bridge_vaa::parse_verify_and_replay_protect(state, bridge_state, vaa, ctx);
        let asset_meta: AssetMeta = asset_meta::parse(corevaa::destroy(vaa));
        let decimals = get_decimals(&asset_meta);
        let treasury_cap = coin::create_currency<T>(witness, decimals, ctx);
        // TODO - assert emitter is registered, extract decimals, token name, symbol, etc. from asset meta
        // TODO - figure out where to store name, symbol, etc.
        let t_cap_store = treasury::create_treasury_cap_store<T>(treasury_cap, ctx);

        let origin_chain = asset_meta::get_token_chain(&asset_meta);
        let external_address = asset_meta::get_token_address(&asset_meta);
        let origin_info = bridge_state::create_origin_info(origin_chain, external_address);
        bridge_state::store_treasury_cap<T>(bridge_state, origin_info, t_cap_store);
    }
}