/// Token Bridge VAA utilities
module token_bridge::vaa {
    use std::option;
    use sui::tx_context::{TxContext};

    use wormhole::myvaa::{Self as corevaa, VAA};
    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{ExternalAddress};

    use token_bridge::bridge_state::{Self as bridge_state, BridgeState};

    //friend token_bridge::contract_upgrade;
    friend token_bridge::register_chain;
    friend token_bridge::wrapped;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;

    #[test_only]
    friend token_bridge::token_bridge_vaa_test;

    /// We have no registration for this chain
    const E_UNKNOWN_CHAIN: u64 = 0;
    /// We have a registration, but it's different from what's given
    const E_UNKNOWN_EMITTER: u64 = 1;

    /// Aborts if the VAA has already been consumed. Marks the VAA as consumed
    /// the first time around.
    public(friend) fun replay_protect(bridge_state: &mut BridgeState, vaa: &VAA) {
        // this calls set::add which aborts if the element already exists
        bridge_state::store_consumed_vaa(bridge_state, corevaa::get_hash(vaa));
    }

    /// Asserts that the VAA is from a known token bridge.
    public fun assert_known_emitter(state: &BridgeState, vm: &VAA) {
        let maybe_emitter = bridge_state::get_registered_emitter(state, &corevaa::get_emitter_chain(vm));
        assert!(option::is_some<ExternalAddress>(&maybe_emitter), E_UNKNOWN_CHAIN);

        let emitter = option::extract(&mut maybe_emitter);
        assert!(emitter == corevaa::get_emitter_address(vm), E_UNKNOWN_EMITTER);
    }

    /// Parses, verifies, and replay protects a token bridge VAA.
    /// Aborts if the VAA is not from a known token bridge emitter.
    ///
    /// Has a 'friend' visibility so that it's only callable by the token bridge
    /// (otherwise the replay protection could be abused to DoS the bridge)
    public(friend) fun parse_verify_and_replay_protect(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        vaa: vector<u8>,
        ctx: &mut TxContext
    ): VAA {
        let vaa = parse_and_verify(wormhole_state, bridge_state, vaa, ctx);
        replay_protect(bridge_state, &vaa);
        vaa
    }

    /// Parses, and verifies a token bridge VAA.
    /// Aborts if the VAA is not from a known token bridge emitter.
    public fun parse_and_verify(wormhole_state: &mut WormholeState, bridge_state: &BridgeState, vaa: vector<u8>, ctx:&mut TxContext): VAA {
        let vaa = corevaa::parse_and_verify(wormhole_state, vaa, ctx);
        assert_known_emitter(bridge_state, &vaa);
        vaa
    }
}

#[test_only]
module token_bridge::token_bridge_vaa_test{
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_shared, return_shared};

    use wormhole::state::{State};
    use wormhole::myvaa::{Self as corevaa};
    use wormhole::myu16::{Self as u16};
    use wormhole::external_address::{Self};

    use token_bridge::bridge_state::{Self, BridgeState};
    use token_bridge::vaa::{Self};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    /// VAA sent from the ethereum token bridge 0xdeadbeef
    const VAA: vector<u8> = x"01000000000100102d399190fa61daccb11c2ea4f7a3db3a9365e5936bcda4cded87c1b9eeb095173514f226256d5579af71d4089eb89496befb998075ba94cd1d4460c5c57b84000000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000002634973000200000000000000000000000000000000000000000000000000000000beefface00020c0000000000000000000000000000000000000000000000000000000042454546000000000000000000000000000000000042656566206661636520546f6b656e";

    #[test]
    #[expected_failure(abort_code = vaa::E_UNKNOWN_CHAIN)]
    fun test_unknown_chain() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);
            let w_state = take_shared<State>(&test);
            let vaa = vaa::parse_verify_and_replay_protect(&mut w_state, &mut state, VAA, ctx(&mut test));
            corevaa::destroy(vaa);
            return_shared<BridgeState>(state);
            return_shared<State>(w_state);
        };
        test_scenario::end(test);
    }


    #[test]
    #[expected_failure(abort_code = vaa::E_UNKNOWN_EMITTER)]
    fun test_unknown_emitter() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);
            bridge_state::set_registered_emitter(
                &mut state,
                u16::from_u64(2),
                external_address::from_bytes(x"deadbeed"), // not deadbeef
            );
            return_shared<BridgeState>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);
            let w_state = take_shared<State>(&test);
            let vaa = vaa::parse_verify_and_replay_protect(&mut w_state, &mut state, VAA, ctx(&mut test));
            corevaa::destroy(vaa);
            return_shared<BridgeState>(state);
            return_shared<State>(w_state);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_known_emitter() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);
            bridge_state::set_registered_emitter(
                &mut state,
                u16::from_u64(2),
                external_address::from_bytes(x"deadbeef"),
            );
            return_shared<BridgeState>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);
            let w_state = take_shared<State>(&test);
            let vaa = vaa::parse_verify_and_replay_protect(&mut w_state, &mut state, VAA, ctx(&mut test));
            corevaa::destroy(vaa);
            return_shared<BridgeState>(state);
            return_shared<State>(w_state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = 0, location=0000000000000000000000000000000000000002::dynamic_field)]
    fun test_replay_protection_works() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);
            bridge_state::set_registered_emitter(
                &mut state,
                u16::from_u64(2),
                external_address::from_bytes(x"deadbeef"),
            );
            return_shared<BridgeState>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);
            let w_state = take_shared<State>(&test);

            // try to use the VAA twice
            let vaa = vaa::parse_verify_and_replay_protect(&mut w_state, &mut state, VAA, ctx(&mut test));
            corevaa::destroy(vaa);
            let vaa = vaa::parse_verify_and_replay_protect(&mut w_state, &mut state, VAA, ctx(&mut test));
            corevaa::destroy(vaa);
            return_shared<BridgeState>(state);
            return_shared<State>(w_state);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_can_verify_after_replay_protect() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);
            bridge_state::set_registered_emitter(
                &mut state,
                u16::from_u64(2),
                external_address::from_bytes(x"deadbeef"),
            );
            return_shared<BridgeState>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);
            let w_state = take_shared<State>(&test);

            // parse and verify and replay protect VAA the first time, don't replay protect the second time
            let vaa = vaa::parse_verify_and_replay_protect(&mut w_state, &mut state, VAA, ctx(&mut test));
            corevaa::destroy(vaa);
            let vaa = vaa::parse_and_verify(&mut w_state, &mut state, VAA, ctx(&mut test));
            corevaa::destroy(vaa);
            return_shared<BridgeState>(state);
            return_shared<State>(w_state);
        };
        test_scenario::end(test);
    }

}
