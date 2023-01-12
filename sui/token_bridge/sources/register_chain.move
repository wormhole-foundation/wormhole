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

    public entry fun submit_vaa(wormhole_state: &mut WormholeState, bridge_state: &mut BridgeState, vaa: vector<u8>, ctx: &mut TxContext) {
        let vaa = corevaa::parse_and_verify(wormhole_state, vaa, ctx);
        corevaa::assert_governance(wormhole_state, &vaa);
        token_bridge_vaa::replay_protect(bridge_state, &vaa);
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

#[test_only]
module token_bridge::register_chain_test {
    use std::option::{Self};

    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_shared, return_shared};

    use wormhole::state::{State};
    //use wormhole::test_state::{init_wormhole_state};
    //use wormhole::wormhole::{Self};

    use wormhole::myu16::{Self as u16};
    use wormhole::external_address::{Self};
    use wormhole::myvaa::{Self as corevaa};

    use token_bridge::bridge_state::{Self as bridge_state, BridgeState};
    use token_bridge::register_chain::{Self, submit_vaa};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    struct MyCoinType1 {}

    /// Registration VAA for the etheruem token bridge 0xdeadbeef
    const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    /// Another registration VAA for the ethereum token bridge, 0xbeefface
    const ETHEREUM_TOKEN_REG_2:vector<u8> = x"01000000000100c2157fa1c14957dff26d891e4ad0d993ad527f1d94f603e3d2bb1e37541e2fbe45855ffda1efc7eb2eb24009a1585fa25a267815db97e4a9d4a5eb31987b5fb40100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000017ca43300000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000beefface";

    /// Registration VAA for the etheruem NFT bridge 0xdeadbeef
    const ETHEREUM_NFT_REG: vector<u8> = x"0100000000010066cce2cb12d88c97d4975cba858bb3c35d6430003e97fced46a158216f3ca01710fd16cc394441a08fef978108ed80c653437f43bb2ca039226974d9512298b10000000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000018483540000000000000000000000000000000000000000000000004e4654427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    const ETH_ID: u64 = 2;

    #[test]
    fun test_parse(){
        test_parse_(scenario())
    }

    #[test]
    #[expected_failure(abort_code = 0, location=token_bridge::register_chain)]
    fun test_parse_fail(){
        test_parse_fail_(scenario())
    }

    #[test]
    fun test_register_chain(){
        test_register_chain_(scenario())
    }

    #[test]
    #[expected_failure(abort_code = 0, location=0000000000000000000000000000000000000002::dynamic_field)]
    fun test_replay_protect(){
        test_replay_protect_(scenario())
    }

    #[test]
    fun test_re_registration(){
        test_re_registration_(scenario())
    }

    public fun test_parse_(test: Scenario) {
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            let vaa = corevaa::parse_test(ETHEREUM_TOKEN_REG);
            let register_chain = register_chain::parse_payload_test(corevaa::destroy(vaa));
            let chain = register_chain::get_emitter_chain_id(&register_chain);
            let address = register_chain::get_emitter_address(&register_chain);

            assert!(chain == u16::from_u64(ETH_ID), 0);
            assert!(address == external_address::from_bytes(x"deadbeef"), 0);
        };
        test_scenario::end(test);
    }

    public fun test_parse_fail_(test: Scenario) {
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            let vaa = corevaa::parse_test(ETHEREUM_NFT_REG);
            // this should fail because it's an NFT registration
            let _register_chain = register_chain::parse_payload_test(corevaa::destroy(vaa));
        };
        test_scenario::end(test);
    }

    fun test_register_chain_(test: Scenario) {
        let (admin, _, _) = people();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<State>(&test);
            let bridge_state = take_shared<BridgeState>(&test);
            submit_vaa(&mut wormhole_state, &mut bridge_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<State>(wormhole_state);
            return_shared<BridgeState>(bridge_state);
        };
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<BridgeState>(&test);
            let addr = bridge_state::get_registered_emitter(&bridge_state, &u16::from_u64(ETH_ID));
            assert!(addr == option::some(external_address::from_bytes(x"deadbeef")), 0);
            return_shared<BridgeState>(bridge_state);
        };
        test_scenario::end(test);
    }

    public fun test_replay_protect_(test: Scenario) {
        let (admin, _, _) = people();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<State>(&test);
            let bridge_state = take_shared<BridgeState>(&test);
            // submit vaa (register chain) twice - triggering replay protection
            submit_vaa(&mut wormhole_state, &mut bridge_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            submit_vaa(&mut wormhole_state, &mut bridge_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<State>(wormhole_state);
            return_shared<BridgeState>(bridge_state);
        };
        test_scenario::end(test);
    }

    public fun test_re_registration_(test: Scenario) {
        // first register chain using ETHEREUM_TOKEN_REG_1
        let (admin, _, _) = people();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<State>(&test);
            let bridge_state = take_shared<BridgeState>(&test);
            submit_vaa(&mut wormhole_state, &mut bridge_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<State>(wormhole_state);
            return_shared<BridgeState>(bridge_state);
        };
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<BridgeState>(&test);
            let addr = bridge_state::get_registered_emitter(&bridge_state, &u16::from_u64(ETH_ID));
            assert!(addr == option::some(external_address::from_bytes(x"deadbeef")), 0);
            return_shared<BridgeState>(bridge_state);
        };
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<State>(&test);
            let bridge_state = take_shared<BridgeState>(&test);
            // TODO(csongor): we register ethereum again, which overrides the
            // previous one. This deviates from other chains (where this is
            // rejected), but I think this is the right behaviour.
            // Easy to change, should be discussed.
            submit_vaa(&mut wormhole_state, &mut bridge_state, ETHEREUM_TOKEN_REG_2, ctx(&mut test));
            let address = bridge_state::get_registered_emitter(&bridge_state, &u16::from_u64(ETH_ID));
            assert!(address == option::some(external_address::from_bytes(x"beefface")), 0);
            return_shared<State>(wormhole_state);
            return_shared<BridgeState>(bridge_state);
        };
        test_scenario::end(test);
    }
}




