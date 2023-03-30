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
    use sui::clock::{Clock};
    use wormhole::state::{State as WormholeState};
    use wormhole::vaa::{Self, VAA};

    use token_bridge::state::{Self, State};
    use token_bridge::version_control::{Vaa as VaaControl};

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
        the_clock: &Clock
    ): VAA {
        state::check_minimum_requirement<VaaControl>(
            token_bridge_state
        );

        let verified =
            handle_parse_and_verify(
                token_bridge_state,
                worm_state,
                vaa_buf,
                the_clock
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
        the_clock: &Clock
    ): VAA {
        parse_verify_and_consume(
            token_bridge_state,
            worm_state,
            vaa_buf,
            the_clock
        )
    }

    /// Parses and verifies a Token Bridge VAA. This method aborts if the VAA
    /// did not originate from a registered Token Bridge emitter.
    fun handle_parse_and_verify(
        token_bridge_state: &State,
        worm_state: &WormholeState,
        vaa: vector<u8>,
        the_clock: &Clock
    ): VAA {
        let parsed = vaa::parse_and_verify(worm_state, vaa, the_clock);

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
module token_bridge::vaa_tests {
    use sui::test_scenario::{Self};
    use wormhole::external_address::{Self};

    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        person,
        register_dummy_emitter,
        return_clock,
        return_states,
        set_up_wormhole_and_token_bridge,
        take_clock,
        take_states
    };
    use token_bridge::vaa::{Self};

    /// VAA sent from the ethereum token bridge 0xdeadbeef.
    const VAA: vector<u8> =
        x"01000000000100102d399190fa61daccb11c2ea4f7a3db3a9365e5936bcda4cded87c1b9eeb095173514f226256d5579af71d4089eb89496befb998075ba94cd1d4460c5c57b84000000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000002634973000200000000000000000000000000000000000000000000000000000000beefface00020c0000000000000000000000000000000000000000000000000000000042454546000000000000000000000000000000000042656566206661636520546f6b656e";

    #[test]
    #[expected_failure(abort_code = state::E_UNREGISTERED_EMITTER)]
    fun test_cannot_parse_verify_and_consume_unregistered_chain() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // You shall not pass!
        let parsed =
            vaa::parse_verify_and_consume_test_only(
                &mut token_bridge_state,
                &worm_state,
                VAA,
                &the_clock
            );

        // Clean up.
        wormhole::vaa::destroy(parsed);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_UNKNOWN_EMITTER)]
    fun test_cannot_parse_verify_and_consume_unknown_emitter() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // First register emitter.
        let emitter_chain = 2;
        let emitter_addr = external_address::from_address(@0xdeafbeef);
        state::register_new_emitter_test_only(
            &mut token_bridge_state,
            emitter_chain,
            emitter_addr
        );
        assert!(
            state::registered_emitter(&token_bridge_state, emitter_chain) == emitter_addr,
            0
        );

        // Confirm that encoded emitter disagrees with registered emitter.
        let parsed =
            wormhole::vaa::parse_and_verify(&worm_state, VAA, &the_clock);
        assert!(wormhole::vaa::emitter_address(&parsed) != emitter_addr, 0);
        wormhole::vaa::destroy(parsed);

        // You shall not pass!
        let parsed =
            vaa::parse_verify_and_consume_test_only(
                &mut token_bridge_state,
                &worm_state,
                VAA,
                &the_clock
            );

        // Clean up.
        wormhole::vaa::destroy(parsed);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_parse_verify_and_consume() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Confirm VAA originated from where we expect.
        let parsed =
            wormhole::vaa::parse_and_verify(&worm_state, VAA, &the_clock);
        assert!(
            wormhole::vaa::emitter_chain(&parsed) == expected_source_chain,
            0
        );
        wormhole::vaa::destroy(parsed);

        // Finally deserialize.
        let parsed =
            vaa::parse_verify_and_consume_test_only(
                &mut token_bridge_state,
                &worm_state,
                VAA,
                &the_clock
            );

        // Clean up.
        wormhole::vaa::destroy(parsed);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = state::E_VAA_ALREADY_CONSUMED)]
    fun test_cannot_parse_verify_and_consume_again() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let (token_bridge_state, worm_state) = take_states(scenario);
        let the_clock = take_clock(scenario);

        // Confirm VAA originated from where we expect.
        let parsed =
            wormhole::vaa::parse_and_verify(
                &worm_state,
                VAA,
                &the_clock
            );
        assert!(
            wormhole::vaa::emitter_chain(&parsed) == expected_source_chain,
            0
        );
        wormhole::vaa::destroy(parsed);

        // Finally deserialize.
        let parsed =
            vaa::parse_verify_and_consume_test_only(
                &mut token_bridge_state,
                &worm_state,
                VAA,
                &the_clock
            );
        wormhole::vaa::destroy(parsed);

        // You shall not pass!
        let parsed =
            vaa::parse_verify_and_consume_test_only(
                &mut token_bridge_state,
                &worm_state,
                VAA,
                &the_clock
            );

        // Clean up.
        wormhole::vaa::destroy(parsed);
        return_states(token_bridge_state, worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

}
