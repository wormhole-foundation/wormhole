/// Token Bridge VAA utilities.
module token_bridge::vaa {
    use sui::tx_context::{TxContext};

    use wormhole::myvaa::{Self as corevaa, VAA};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{Self, State};

    friend token_bridge::register_chain;
    friend token_bridge::create_wrapped;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;

    #[test_only]
    friend token_bridge::token_bridge_vaa_test;

    // We have no registration for this chain.
    const E_UNKNOWN_CHAIN: u64 = 0;
    // We have a registration, but it's different from what's given.
    const E_UNKNOWN_EMITTER: u64 = 1;

    /// Aborts if the VAA has already been consumed. Marks the VAA as consumed
    /// the first time around.
    public(friend) fun replay_protect(
        token_bridge_state: &mut State,
        vaa: &VAA
    ) {
        // This calls set::add which aborts if the element already exists.
        state::store_consumed_vaa(token_bridge_state, corevaa::get_hash(vaa));
    }

    /// Asserts that the VAA is from a known token bridge.
    public fun verify_emitter(token_bridge_state: &State, vm: &VAA) {
        let foreign_emitter =
            state::registered_emitter(
                token_bridge_state,
                corevaa::get_emitter_chain(vm)
            );
        assert!(
            foreign_emitter == corevaa::get_emitter_address(vm),
            E_UNKNOWN_EMITTER
        );
    }

    /// Parses, verifies, and replay protects a token bridge VAA.
    /// Aborts if the VAA is not from a known token bridge emitter.
    ///
    /// Has a 'friend' visibility so that it's only callable by the token bridge
    /// (otherwise the replay protection could be abused to DoS the bridge).
    public(friend) fun parse_verify_and_replay_protect(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        vaa: vector<u8>,
        ctx: &mut TxContext
    ): VAA {
        let vaa = parse_and_verify(token_bridge_state, worm_state, vaa, ctx);
        replay_protect(token_bridge_state, &vaa);
        vaa
    }

    /// Parses, and verifies a token bridge VAA.
    /// Aborts if the VAA is not from a known token bridge emitter.
    public fun parse_and_verify(
        token_bridge_state: &State,
        worm_state: &mut WormholeState,
        vaa: vector<u8>,
        ctx:&mut TxContext
    ): VAA {
        let vaa = corevaa::parse_and_verify(worm_state, vaa, ctx);
        verify_emitter(token_bridge_state, &vaa);
        vaa
    }
}

#[test_only]
module token_bridge::token_bridge_vaa_test{
    use sui::test_scenario::{
        Self,
        Scenario,
        next_tx,
        ctx,
        take_shared,
        return_shared
    };

    use wormhole::state::{State as WormholeState};
    use wormhole::myvaa::{Self as corevaa};
    use wormhole::external_address::{Self};

    use token_bridge::state::{Self, State};
    use token_bridge::vaa::{Self};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    /// VAA sent from the ethereum token bridge 0xdeadbeef.
    const VAA: vector<u8> =
        x"01000000000100102d399190fa61daccb11c2ea4f7a3db3a9365e5936bcda4cded87c1b9eeb095173514f226256d5579af71d4089eb89496befb998075ba94cd1d4460c5c57b84000000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000002634973000200000000000000000000000000000000000000000000000000000000beefface00020c0000000000000000000000000000000000000000000000000000000042454546000000000000000000000000000000000042656566206661636520546f6b656e";

    #[test]
    #[expected_failure(
        abort_code = token_bridge::state::E_UNREGISTERED_EMITTER,
        location = token_bridge::state
    )]
    fun test_unknown_chain() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let w_state = take_shared<WormholeState>(&test);
            let vaa =
                vaa::parse_verify_and_replay_protect(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            corevaa::destroy(vaa);
            return_shared<State>(state);
            return_shared<WormholeState>(w_state);
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
            let state = take_shared<State>(&test);
            state::register_emitter(
                &mut state,
                2, // chain ID
                external_address::from_bytes(x"deadbeed"), // not deadbeef
            );
            return_shared<State>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let w_state = take_shared<WormholeState>(&test);
            let vaa =
                vaa::parse_verify_and_replay_protect(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            corevaa::destroy(vaa);
            return_shared<State>(state);
            return_shared<WormholeState>(w_state);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_known_emitter() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            state::register_emitter(
                &mut state,
                2, // chain ID
                external_address::from_bytes(x"deadbeef"),
            );
            return_shared<State>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let w_state = take_shared<WormholeState>(&test);
            let vaa =
                vaa::parse_verify_and_replay_protect(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            corevaa::destroy(vaa);
            return_shared<State>(state);
            return_shared<WormholeState>(w_state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = 0, location=sui::dynamic_field)]
    fun test_replay_protection_works() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            state::register_emitter(
                &mut state,
                2, // chain ID
                external_address::from_bytes(x"deadbeef"),
            );
            return_shared<State>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let w_state = take_shared<WormholeState>(&test);

            // Try to use the VAA twice.
            let vaa =
                vaa::parse_verify_and_replay_protect(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            corevaa::destroy(vaa);
            let vaa =
                vaa::parse_verify_and_replay_protect(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            corevaa::destroy(vaa);
            return_shared<State>(state);
            return_shared<WormholeState>(w_state);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_can_verify_after_replay_protect() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            state::register_emitter(
                &mut state,
                2, // chain ID
                external_address::from_bytes(x"deadbeef"),
            );
            return_shared<State>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let w_state = take_shared<WormholeState>(&test);

            // Parse and verify and replay protect VAA the first time, don't
            // replay protect the second time.
            let vaa =
                vaa::parse_verify_and_replay_protect(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            corevaa::destroy(vaa);
            let vaa = vaa::parse_and_verify(
                &mut state,
                &mut w_state,
                VAA,
                ctx(&mut test)
            );
            corevaa::destroy(vaa);
            return_shared<State>(state);
            return_shared<WormholeState>(w_state);
        };
        test_scenario::end(test);
    }

}
