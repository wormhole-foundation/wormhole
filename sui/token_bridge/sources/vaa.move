/// This module builds on Wormhole's `vaa::parse_and_verify` method by adding
/// emitter verification and replay protection.
///
/// Token Bridge only cares about other Token Bridge messages, so the emitter
/// address must be a registered Token Bridge emitter according to the VAA's
/// emitter chain ID.
///
/// Token Bridge does not allow replaying any of its VAAs, so its hash is stored
/// in its `State`. If the encoded VAA passes through `parse_and_verify` again,
/// it will abort.
module token_bridge::vaa {
    use sui::tx_context::{TxContext};
    use wormhole::state::{State as WormholeState};
    use wormhole::vaa::{Self, VAA};

    use token_bridge::state::{Self, State};

    // All friends need `parse_and_verify`.
    friend token_bridge::create_wrapped;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::register_chain;

    /// We have no registration for this chain.
    const E_UNKNOWN_CHAIN: u64 = 0;
    /// We have a registration, but it's different from what is given.
    const E_UNKNOWN_EMITTER: u64 = 1;

    /// Parses and verifies encoded VAA. Because Token Bridge does not allow
    /// VAAs to be replayed, the VAA hash is stored in a set, which is checked
    /// against the next time the same VAA is used to make sure it cannot be
    /// used again.
    ///
    /// In its verification, this method checks whether the emitter is a
    /// registered Token Bridge contract on another network.
    ///
    /// NOTE: This method has `friend` visibility so it is only callable by this
    /// contract. Otherwise the replay protection could be abused to DoS the
    /// Token Bridge.
    public(friend) fun parse_verify_and_consume(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        ctx: &TxContext
    ): VAA {
        let verified =
            handle_parse_and_verify(
                token_bridge_state,
                worm_state,
                vaa_buf,
                ctx
            );

        // Consume the VAA hash to prevent replay.
        state::consume_vaa_hash(token_bridge_state, vaa::digest(&verified));

        verified
    }

    #[test_only]
    public fun parse_verify_and_consume_test_only(
        token_bridge_state: &mut State,
        worm_state: &WormholeState,
        vaa_buf: vector<u8>,
        ctx: &TxContext
    ): VAA {
        parse_verify_and_consume(
            token_bridge_state,
            worm_state,
            vaa_buf,
            ctx
        )
    }

    /// Parses and verifies a Token Bridge VAA. This method aborts if the VAA
    /// did not originate from a registered Token Bridge emitter.
    fun handle_parse_and_verify(
        token_bridge_state: &State,
        worm_state: &WormholeState,
        vaa: vector<u8>,
        ctx: &TxContext
    ): VAA {
        let parsed = vaa::parse_and_verify(worm_state, vaa, ctx);

        // Did the VAA originate from another Token Bridge contract?
        let emitter =
            state::registered_emitter(
                token_bridge_state,
                vaa::emitter_chain(&parsed)
            );
        assert!(emitter == vaa::emitter_address(&parsed), E_UNKNOWN_EMITTER);

        parsed
    }
}

#[test_only]
module token_bridge::token_bridge_vaa_test {
    use sui::test_scenario::{
        Self,
        Scenario,
        next_tx,
        ctx,
        take_shared,
        return_shared
    };

    use wormhole::state::{State as WormholeState};
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
        abort_code = token_bridge::emitter_registry::E_UNREGISTERED
    )]
    fun test_unknown_chain() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let w_state = take_shared<WormholeState>(&test);
            let vaa =
                vaa::parse_verify_and_consume_test_only(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            wormhole::vaa::destroy(vaa);
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
            state::register_new_emitter_test_only(
                &mut state,
                2, // chain ID
                external_address::from_any_bytes(x"deadbeed"), // not deadbeef
            );
            return_shared<State>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let w_state = take_shared<WormholeState>(&test);
            let vaa =
                vaa::parse_verify_and_consume_test_only(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            wormhole::vaa::destroy(vaa);
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
            state::register_new_emitter_test_only(
                &mut state,
                2, // chain ID
                external_address::from_any_bytes(x"deadbeef"),
            );
            return_shared<State>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let w_state = take_shared<WormholeState>(&test);
            let vaa =
                vaa::parse_verify_and_consume_test_only(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            wormhole::vaa::destroy(vaa);
            return_shared<State>(state);
            return_shared<WormholeState>(w_state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = state::E_VAA_ALREADY_CONSUMED)]
    fun test_replay_protection_works() {
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            state::register_new_emitter_test_only(
                &mut state,
                2, // chain ID
                external_address::from_any_bytes(x"deadbeef"),
            );
            return_shared<State>(state);
        };

        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let w_state = take_shared<WormholeState>(&test);

            // Try to use the VAA twice.
            let vaa =
                vaa::parse_verify_and_consume_test_only(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            wormhole::vaa::destroy(vaa);
            let vaa =
                vaa::parse_verify_and_consume_test_only(
                    &mut state,
                    &mut w_state,
                    VAA,
                    ctx(&mut test)
                );
            wormhole::vaa::destroy(vaa);
            return_shared<State>(state);
            return_shared<WormholeState>(w_state);
        };
        test_scenario::end(test);
    }

}
