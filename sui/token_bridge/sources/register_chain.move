module token_bridge::register_chain {

    use sui::tx_context::TxContext;

    use wormhole::myu16::{Self as u16, U16};
    use wormhole::cursor;
    use wormhole::deserialize;
    use wormhole::myvaa::{Self as corevaa};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::state::{State as WormholeState};

    use token_bridge::vaa as token_bridge_vaa;
    use token_bridge::bridge_state::{Self as bridge_state, BridgeState};

    /// "TokenBridge" (left padded)
    const TOKEN_BRIDGE: vector<u8> = x"000000000000000000000000000000000000000000546f6b656e427269646765";

    const E_INVALID_MODULE: u64 = 0;
    const E_INVALID_ACTION: u64 = 1;
    const E_INVALID_TARGET: u64 = 2;

    struct RegisterChain has copy, drop {
        /// Chain ID
        emitter_chain_id: U16,
        /// Emitter address. Left-zero-padded if shorter than 32 bytes
        emitter_address: ExternalAddress,
    }

    #[test_only]
    public fun parse_payload_test(payload: vector<u8>): RegisterChain {
        parse_payload(payload)
    }

    fun parse_payload(payload: vector<u8>): RegisterChain {
        let cur = cursor::cursor_init(payload);
        let target_module = deserialize::deserialize_vector(&mut cur, 32);

        assert!(target_module == TOKEN_BRIDGE, E_INVALID_MODULE);

        let action = deserialize::deserialize_u8(&mut cur);
        assert!(action == 0x01, E_INVALID_ACTION);

        // TODO(csongor): should we also accept a VAA directly?
        // why would a registration VAA target a specific chain?
        let target_chain = deserialize::deserialize_u16(&mut cur);
        assert!(target_chain == u16::from_u64(0x0), E_INVALID_TARGET);

        let emitter_chain_id = deserialize::deserialize_u16(&mut cur);

        let emitter_address = external_address::deserialize(&mut cur);

        cursor::destroy_empty(cur);

        RegisterChain { emitter_chain_id, emitter_address }
    }

    // TODO - make this an entry fun?
    public entry fun submit_vaa(wormhole_state: &mut WormholeState, bridge_state: &mut BridgeState, vaa: vector<u8>, ctx: &mut TxContext) {
        let vaa = corevaa::parse_and_verify(wormhole_state, vaa, ctx);
        corevaa::assert_governance(wormhole_state, &vaa); // not tested
        token_bridge_vaa::replay_protect(bridge_state, &vaa, ctx);

        let RegisterChain { emitter_chain_id, emitter_address } = parse_payload(corevaa::destroy(vaa));

        bridge_state::set_registered_emitter(bridge_state, emitter_chain_id, emitter_address);
    }

    public fun get_emitter_chain_id(a: &RegisterChain): U16 {
        a.emitter_chain_id
    }

    public fun get_emitter_address(a: &RegisterChain): ExternalAddress {
        a.emitter_address
    }

}

// TODO - test register chain
