module token_bridge::register_chain {
    use sui::tx_context::TxContext;
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::governance_message::{Self};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{Self, State};

    const E_INVALID_MODULE: u64 = 0;
    const E_INVALID_ACTION: u64 = 1;
    const E_INVALID_TARGET: u64 = 2;

    const ACTION_REGISTER_CHAIN: u8 = 1;

    struct RegisterChain {
        emitter_chain: u16,
        emitter_address: ExternalAddress,
    }

    public entry fun submit_vaa(
        token_bridge_state: &mut State,
        wormhole_state: &mut WormholeState,
        vaa_buf: vector<u8>,
        ctx: &TxContext
    ) {
        let msg =
            governance_message::parse_and_verify_vaa(
                wormhole_state,
                vaa_buf,
                ctx
            );

        // Protect against replaying the VAA.
        state::consume_vaa_hash(
            token_bridge_state,
            governance_message::vaa_hash(&msg)
        );

        assert!(
            governance_message::module_name(&msg) == state::governance_module(),
            E_INVALID_MODULE
        );
        assert!(
            governance_message::action(&msg) == ACTION_REGISTER_CHAIN,
            E_INVALID_ACTION
        );
        assert!(
            governance_message::is_global_action(&msg),
            E_INVALID_TARGET
        );

        let cur = cursor::new(governance_message::take_payload(msg));
        let emitter_chain = bytes::take_u16_be(&mut cur);
        let emitter_address = external_address::take_bytes(&mut cur);
        cursor::destroy_empty(cur);

        state::register_emitter(
            token_bridge_state,
            emitter_chain,
            emitter_address
        );
    }

    #[test_only]
    public fun emitter_chain(self: &RegisterChain): u16 {
        self.emitter_chain
    }

    #[test_only]
    public fun emitter_address(self: &RegisterChain): ExternalAddress {
        self.emitter_address
    }
}

#[test_only]
module token_bridge::register_chain_test {
    // use sui::test_scenario::{
    //     Self,
    //     Scenario,
    //     next_tx,
    //     ctx,
    //     take_shared,
    //     return_shared
    // };

    // use wormhole::state::{State as WormholeState};
    // //use wormhole::test_state::{init_wormhole_state};
    // //use wormhole::wormhole::{Self};

    // use wormhole::external_address::{Self};
    // use wormhole::vaa::{Self as corevaa};

    // use token_bridge::state::{Self, State};
    // use token_bridge::register_chain::{Self, submit_vaa};
    // use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};

    // fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    // fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    // struct MyCoinType1 {}

    // /// Registration VAA for the etheruem token bridge 0xdeadbeef
    // const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    // /// Another registration VAA for the ethereum token bridge, 0xbeefface
    // const ETHEREUM_TOKEN_REG_2:vector<u8> = x"01000000000100c2157fa1c14957dff26d891e4ad0d993ad527f1d94f603e3d2bb1e37541e2fbe45855ffda1efc7eb2eb24009a1585fa25a267815db97e4a9d4a5eb31987b5fb40100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000017ca43300000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000beefface";

    // /// Registration VAA for the etheruem NFT bridge 0xdeadbeef
    // const ETHEREUM_NFT_REG: vector<u8> = x"0100000000010066cce2cb12d88c97d4975cba858bb3c35d6430003e97fced46a158216f3ca01710fd16cc394441a08fef978108ed80c653437f43bb2ca039226974d9512298b10000000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000018483540000000000000000000000000000000000000000000000004e4654427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    // const CHAIN_ID_ETH: u16 = 2;

    // #[test]
    // fun test_parse(){
    //     test_parse_(scenario())
    // }

    // #[test]
    // #[expected_failure(abort_code = 0, location=token_bridge::register_chain)]
    // fun test_parse_fail(){
    //     test_parse_fail_(scenario())
    // }

    // #[test]
    // fun test_register_chain(){
    //     test_register_chain_(scenario())
    // }

    // #[test]
    // #[expected_failure(abort_code = 0, location=sui::dynamic_field)]
    // fun test_replay_protect(){
    //     test_replay_protect_(scenario())
    // }

    // #[test]
    // #[expected_failure(
    //     abort_code = token_bridge::state::E_EMITTER_ALREADY_REGISTERED,
    //     location = token_bridge::state
    // )]
    // fun test_re_registration(){
    //     test_re_registration_(scenario())
    // }

    // public fun test_parse_(test: Scenario) {
    //     let (admin, _, _) = people();
    //     next_tx(&mut test, admin); {
    //         let vaa = corevaa::parse_test(ETHEREUM_TOKEN_REG);
    //         let register_chain =
    //             register_chain::parse_payload_test(corevaa::take_payload(vaa));
    //         let chain = register_chain::emitter_chain(&register_chain);
    //         let addr = register_chain::emitter_address(&register_chain);

    //         assert!(chain == CHAIN_ID_ETH, 0);
    //         assert!(addr == external_address::from_any_bytes(x"deadbeef"), 0);
    //     };
    //     test_scenario::end(test);
    // }

    // public fun test_parse_fail_(test: Scenario) {
    //     let (admin, _, _) = people();
    //     next_tx(&mut test, admin); {
    //         let vaa = corevaa::parse_test(ETHEREUM_NFT_REG);
    //         // this should fail because it's an NFT registration
    //         let _register_chain =
    //             register_chain::parse_payload_test(corevaa::take_payload(vaa));
    //     };
    //     test_scenario::end(test);
    // }

    // fun test_register_chain_(test: Scenario) {
    //     let (admin, _, _) = people();
    //     test = set_up_wormhole_core_and_token_bridges(admin, test);
    //     next_tx(&mut test, admin); {
    //         let worm_state = take_shared<WormholeState>(&test);
    //         let bridge_state = take_shared<State>(&test);
    //         submit_vaa(&mut worm_state, &mut bridge_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
    //         return_shared<WormholeState>(worm_state);
    //         return_shared<State>(bridge_state);
    //     };
    //     next_tx(&mut test, admin); {
    //         let bridge_state = take_shared<State>(&test);
    //         let addr = state::registered_emitter(&bridge_state, CHAIN_ID_ETH);
    //         assert!(addr == external_address::from_any_bytes(x"deadbeef"), 0);
    //         return_shared<State>(bridge_state);
    //     };
    //     test_scenario::end(test);
    // }

    // public fun test_replay_protect_(test: Scenario) {
    //     let (admin, _, _) = people();
    //     test = set_up_wormhole_core_and_token_bridges(admin, test);
    //     next_tx(&mut test, admin); {
    //         let worm_state = take_shared<WormholeState>(&test);
    //         let bridge_state = take_shared<State>(&test);
    //         // submit vaa (register chain) twice - triggering replay protection
    //         submit_vaa(&mut worm_state, &mut bridge_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
    //         submit_vaa(&mut worm_state, &mut bridge_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
    //         return_shared<WormholeState>(worm_state);
    //         return_shared<State>(bridge_state);
    //     };
    //     test_scenario::end(test);
    // }

    // public fun test_re_registration_(test: Scenario) {
    //     // first register chain using ETHEREUM_TOKEN_REG_1
    //     let (admin, _, _) = people();
    //     test = set_up_wormhole_core_and_token_bridges(admin, test);
    //     next_tx(&mut test, admin); {
    //         let worm_state = take_shared<WormholeState>(&test);
    //         let bridge_state = take_shared<State>(&test);
    //         submit_vaa(&mut worm_state, &mut bridge_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
    //         return_shared<WormholeState>(worm_state);
    //         return_shared<State>(bridge_state);
    //     };
    //     next_tx(&mut test, admin); {
    //         let bridge_state = take_shared<State>(&test);
    //         let addr = state::registered_emitter(&bridge_state, CHAIN_ID_ETH);
    //         assert!(addr == external_address::from_any_bytes(x"deadbeef"), 0);
    //         return_shared<State>(bridge_state);
    //     };
    //     next_tx(&mut test, admin); {
    //         let worm_state = take_shared<WormholeState>(&test);
    //         let bridge_state = take_shared<State>(&test);
    //         // TODO(csongor): we register ethereum again, which overrides the
    //         // previous one. This deviates from other chains (where this is
    //         // rejected), but I think this is the right behaviour.
    //         // Easy to change, should be discussed.
    //         submit_vaa(&mut worm_state, &mut bridge_state, ETHEREUM_TOKEN_REG_2, ctx(&mut test));
    //         let address = state::registered_emitter(&bridge_state, CHAIN_ID_ETH);
    //         assert!(address == external_address::from_any_bytes(x"beefface"), 0);
    //         return_shared<WormholeState>(worm_state);
    //         return_shared<State>(bridge_state);
    //     };
    //     test_scenario::end(test);
    // }
}




