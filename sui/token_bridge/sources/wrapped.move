/// This module uses the one-time witness (OTW)
/// Sui one-time witness pattern reference: https://examples.sui.io/basics/one-time-witness.html
module token_bridge::wrapped {
    use sui::tx_context::TxContext;
    use sui::coin::{TreasuryCap};
    use sui::transfer::{Self};
    use sui::tx_context::{Self};

    use token_bridge::bridge_state::{Self, BridgeState};
    use token_bridge::asset_meta::{AssetMeta, Self};
    use token_bridge::treasury::{Self};

    use wormhole::state::{Self as state, State as WormholeState};
    use wormhole::myvaa::{parse_and_get_payload};

    const E_WRAPPING_NATIVE_COIN: u64 = 0;
    const E_WRAPPING_REGISTERED_NATIVE_COIN: u64 = 1;
    const E_WRAPPED_COIN_ALREADY_INITIALIZED: u64 = 2;

    public entry fun create_wrapped_coin_test_1<CoinType>(
        _state: &mut WormholeState,
        _bridge_state: &mut BridgeState,
     ){}

    public entry fun create_wrapped_coin_test_2<CoinType>(
        _state: &mut WormholeState,
        _bridge_state: &mut BridgeState,
        treasury_cap: TreasuryCap<CoinType>,
        _vaa: vector<u8>,
        ctx: &mut TxContext,
     ){
        transfer::transfer(treasury_cap, tx_context::sender(ctx));
     }

    public entry fun create_wrapped_coin_test_3<CoinType>(
        _state: &mut WormholeState,
        _bridge_state: &mut BridgeState,
        treasury_cap: TreasuryCap<CoinType>,
        bytes: vector<u8>,
        ctx: &mut TxContext,
    ){
        let payload = parse_and_get_payload(bytes);
        let _asset_meta: AssetMeta = asset_meta::parse(payload);
        transfer::transfer(treasury_cap, tx_context::sender(ctx));
    }

     public entry fun create_wrapped_coin_test_4<CoinType>(
        _state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        treasury_cap: TreasuryCap<CoinType>,
        bytes: vector<u8>,
        ctx: &mut TxContext,
    ){
        let payload = parse_and_get_payload(bytes);
        let metadata = asset_meta::parse(payload);
        let t_cap_store = treasury::create_treasury_cap_store<CoinType>(treasury_cap, ctx);
        let origin_chain = asset_meta::get_token_chain(&metadata);
        let external_address = asset_meta::get_token_address(&metadata);
        let wrapped_asset_info = bridge_state::create_wrapped_asset_info(origin_chain, external_address, ctx);

        bridge_state::store_treasury_cap<CoinType>(bridge_state, t_cap_store);
        transfer::transfer(wrapped_asset_info, tx_context::sender(ctx));
    }

    public entry fun create_wrapped_coin_test_5<CoinType>(
        state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        treasury_cap: TreasuryCap<CoinType>,
        bytes: vector<u8>,
        ctx: &mut TxContext,
    ){
        let payload = parse_and_get_payload(bytes);

        let metadata = asset_meta::parse(payload);
        let t_cap_store = treasury::create_treasury_cap_store<CoinType>(treasury_cap, ctx);
        let origin_chain = asset_meta::get_token_chain(&metadata);
        let external_address = asset_meta::get_token_address(&metadata);
        let wrapped_asset_info = bridge_state::create_wrapped_asset_info(origin_chain, external_address, ctx);

        assert!(origin_chain != state::get_chain_id(state), E_WRAPPING_NATIVE_COIN);
        assert!(!bridge_state::is_registered_native_asset<CoinType>(bridge_state), E_WRAPPING_REGISTERED_NATIVE_COIN);
        assert!(!bridge_state::is_wrapped_asset<CoinType>(bridge_state), E_WRAPPED_COIN_ALREADY_INITIALIZED);

        bridge_state::register_wrapped_asset<CoinType>(bridge_state, wrapped_asset_info);
        bridge_state::store_treasury_cap<CoinType>(bridge_state, t_cap_store);
    }

    public entry fun create_wrapped_coin<CoinType>(
        state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        treasury_cap: TreasuryCap<CoinType>,
        bytes: vector<u8>,
        ctx: &mut TxContext,
    ){
        let payload = parse_and_get_payload(bytes);

        // TODO - parse and verify VAA instead of merely extracting payload
        //        let vaa = token_bridge_vaa::parse_and_verify(state, bridge_state, bytes, ctx);
        //        let _payload = destroy(vaa);

        // TODO - check that emitter is registered

        // TODO (pending Mysten Labs uniform token standard) -  extract and store token metadata

        // TODO - confirm TreasuryCap token supply is zero

        // TODO - check token metadata corresponding with TreasuryCap is correct

        let metadata = asset_meta::parse(payload);
        let t_cap_store = treasury::create_treasury_cap_store<CoinType>(treasury_cap, ctx);
        let origin_chain = asset_meta::get_token_chain(&metadata);
        let external_address = asset_meta::get_token_address(&metadata);
        let wrapped_asset_info = bridge_state::create_wrapped_asset_info(origin_chain, external_address, ctx);

        assert!(origin_chain != state::get_chain_id(state), E_WRAPPING_NATIVE_COIN);
        assert!(!bridge_state::is_registered_native_asset<CoinType>(bridge_state), E_WRAPPING_REGISTERED_NATIVE_COIN);
        assert!(!bridge_state::is_wrapped_asset<CoinType>(bridge_state), E_WRAPPED_COIN_ALREADY_INITIALIZED);

        bridge_state::register_wrapped_asset<CoinType>(bridge_state, wrapped_asset_info);
        bridge_state::store_treasury_cap<CoinType>(bridge_state, t_cap_store);
    }
}
