module token_bridge::wrapped {
    use sui::tx_context::TxContext;
    use sui::object::{Self, UID};
    use sui::coin::{Self, Coin, TreasuryCap};
    use sui::transfer::{Self};

    use token_bridge::bridge_state::{BridgeState};
    use token_bridge::vaa::{Self as token_bridge_vaa};
    use token_bridge::asset_meta::{AssetMeta, Self};

    use wormhole::state::{State as WormholeState};
    use wormhole::myvaa::{Self as corevaa};

    struct TreasuryCapContainer<phantom CoinType> has key, store {
        id: UID,
        t: TreasuryCap<CoinType>,
    }

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
        transfer::share_object(TreasuryCapContainer{id: object::new(ctx), t: treasury_cap});
    }

    // One can only call mint in complete_transfer when minting wrapped assets is necessary
    public(friend) fun mint<T: drop>(
        cap_container: &mut TreasuryCapContainer<T>,
        value: u64,
        ctx: &mut TxContext,
    ): Coin<T> {
        coin::mint<T>(&mut cap_container.t, value, ctx)
    }
}