/// This module uses the one-time witness (OTW)
/// Sui one-time witness pattern reference: https://examples.sui.io/basics/one-time-witness.html
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

    use wormhole::state::{Self as state, State as WormholeState};
    use wormhole::myvaa::{Self as corevaa};

    const E_WRAPPING_NATIVE_COIN: u64 = 0;
    const E_WRAPPING_REGISTERED_NATIVE_COIN: u64 = 1;
    const E_WRAPPED_COIN_ALREADY_INITIALIZED: u64 = 2;

    public entry fun create_wrapped_coin<CoinType: drop>(
        state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        vaa: vector<u8>,
        witness: CoinType,
        ctx: &mut TxContext,
    ) {
        let vaa = token_bridge_vaa::parse_verify_and_replay_protect(state, bridge_state, vaa, ctx);
        let asset_meta: AssetMeta = asset_meta::parse(corevaa::destroy(vaa));
        let decimals = get_decimals(&asset_meta);
        let treasury_cap = coin::create_currency<CoinType>(witness, decimals, ctx);
        // assert emitter is registered

        // TODO (pending Mysten Labs uniform token standard) -  extract/store decimals, token name, symbol, etc. from asset meta
        let t_cap_store = treasury::create_treasury_cap_store<CoinType>(treasury_cap, ctx);

        let origin_chain = asset_meta::get_token_chain(&asset_meta);
        assert!(origin_chain != state::get_chain_id(state), E_WRAPPING_NATIVE_COIN);
        assert!(!bridge_state::is_registered_native_asset<CoinType>(bridge_state), E_WRAPPING_REGISTERED_NATIVE_COIN);
        assert!(!bridge_state::is_wrapped_asset<CoinType>(bridge_state), E_WRAPPED_COIN_ALREADY_INITIALIZED);

        let external_address = asset_meta::get_token_address(&asset_meta);
        let wrapped_asset_info = bridge_state::create_wrapped_asset_info(origin_chain, external_address, ctx);

        bridge_state::register_wrapped_asset<CoinType>(bridge_state, wrapped_asset_info);
        bridge_state::store_treasury_cap<CoinType>(bridge_state, t_cap_store);
    }
}

#[test_only]
module token_bridge::test_token {
    use sui::tx_context::TxContext;
    use token_bridge::test_create_wrapped::create_wrapped;

    struct TEST_TOKEN has drop {}

    // ======================== One time witness pattern
    //#[test] - we can't seem to put a #[test] annotation here, because
    //          the args for ctx and OTW are supposed to be supplied by the runtime and not us?
    //          if we don't put a test, the code isn't actually run it seems?
    //
    //          Reference to OTW: https://examples.sui.io/basics/one-time-witness.html
    //
    //fun init(x: TEST_TOKEN, _ctx: &mut TxContext){
    //    //create_wrapped<TEST_TOKEN>(x)
    //}

    // ========================= Transferable witness pattern
    struct WitnessCarrier has key { id: UID, witness: TEST_TOKEN }
    /// Send a `WitnessCarrier` to the module publisher.
    fun init(ctx: &mut TxContext) {
        transfer::transfer(
            WitnessCarrier { id: object::new(ctx), witness: WITNESS {} },
            tx_context::sender(ctx)
        )
    }

    /// Unwrap a carrier and get the inner WITNESS type.
    public fun get_witness(carrier: WitnessCarrier): TEST_TOKEN {
        let WitnessCarrier { id, witness } = carrier;
        object::delete(id);
        witness
    }
}

#[test_only]
module token_bridge::test_create_wrapped {
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_shared, return_shared};

    use wormhole::state::{State};

    use token_bridge::bridge_state::{BridgeState};
    use token_bridge::test_bridge_state::{set_up_wormhole_core_and_token_bridges};
    use token_bridge::wrapped::{create_wrapped_coin};

    use token_bridge::test_token::{get_witness};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    /// Registration VAA for the etheruem token bridge 0xdeadbeef
    const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    /// Attestation VAA sent from the ethereum token bridge 0xdeadbeef
    const ATTESTATION_VAA: vector<u8> = x"0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";

    // TODO - first register emitter for eth token bridge 0xdeadbeef?

    public fun create_wrapped<T: drop>(x: T) {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<BridgeState>(&test);
            let wormhole_state = take_shared<State>(&test);
            create_wrapped_coin<T>(
                &mut wormhole_state,
                &mut bridge_state,
                ATTESTATION_VAA,
                x,
                ctx(&mut test)
            );
            return_shared<BridgeState>(bridge_state);
            return_shared<State>(wormhole_state);
        };
        test_scenario::end(test);
    }

}