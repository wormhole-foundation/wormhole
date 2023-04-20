// SPDX-License-Identifier: Apache 2

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
    use wormhole::consumed_vaas::{Self};
    use wormhole::vaa::{Self, VAA};

    use token_bridge::state::{Self, State};
    use token_bridge::version_control::{Vaa as VaaControl};

    // All friends need `parse_and_verify`.
    friend token_bridge::create_wrapped;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;

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
    public(friend) fun verify_only_once(
        token_bridge_state: &mut State,
        parsed: VAA
    ): VAA {
        state::check_minimum_requirement<VaaControl>(
            token_bridge_state
        );

        // First parse and verify VAA using Wormhole. This also consumes the VAA
        // hash to prevent replay.
        consumed_vaas::consume(
            state::borrow_mut_consumed_vaas(token_bridge_state),
            vaa::digest(&parsed)
        );

        // Does the emitter agree with a registered Token Bridge?
        state::assert_registered_emitter(token_bridge_state, &parsed);

        parsed
    }

    #[test_only]
    public fun verify_only_once_test_only(
        token_bridge_state: &mut State,
        parsed: VAA
    ): VAA {
        verify_only_once(token_bridge_state, parsed)
    }
}

#[test_only]
module token_bridge::vaa_tests {
    use sui::test_scenario::{Self};
    use wormhole::external_address::{Self};
    use wormhole::wormhole_scenario::{parse_and_verify_vaa};

    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        person,
        register_dummy_emitter,
        return_state,
        set_up_wormhole_and_token_bridge,
        take_state
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

        let token_bridge_state = take_state(scenario);

        // You shall not pass!
        let parsed = parse_and_verify_vaa(scenario, VAA);
        let verified =
            vaa::verify_only_once_test_only(&mut token_bridge_state, parsed);

        // Clean up.
        wormhole::vaa::destroy(verified);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = state::E_EMITTER_ADDRESS_MISMATCH)]
    fun test_cannot_parse_verify_and_consume_emitter_address_mismatch() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let token_bridge_state = take_state(scenario);

        // First register emitter.
        let emitter_chain = 2;
        let emitter_addr = external_address::from_address(@0xdeafbeef);
        state::register_new_emitter_test_only(
            &mut token_bridge_state,
            emitter_chain,
            emitter_addr
        );

        // Confirm that encoded emitter disagrees with registered emitter.
        let parsed = parse_and_verify_vaa(scenario, VAA);
        assert!(wormhole::vaa::emitter_address(&parsed) != emitter_addr, 0);

        // You shall not pass!
        let verified =
            vaa::verify_only_once_test_only(&mut token_bridge_state, parsed);

        // Clean up.
        wormhole::vaa::destroy(verified);
        return_state(token_bridge_state);

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

        let token_bridge_state = take_state(scenario);

        // Confirm VAA originated from where we expect.
        let parsed = parse_and_verify_vaa(scenario, VAA);
        assert!(
            wormhole::vaa::emitter_chain(&parsed) == expected_source_chain,
            0
        );

        // Finally verify.
        let verified =
            vaa::verify_only_once_test_only(&mut token_bridge_state, parsed);

        // Clean up.
        wormhole::vaa::destroy(verified);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wormhole::set::E_KEY_ALREADY_EXISTS)]
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

        let token_bridge_state = take_state(scenario);

        // Confirm VAA originated from where we expect.
        let parsed = parse_and_verify_vaa(scenario, VAA);
        assert!(
            wormhole::vaa::emitter_chain(&parsed) == expected_source_chain,
            0
        );

        // Finally verify.
        let verified =
            vaa::verify_only_once_test_only(&mut token_bridge_state, parsed);

        // You shall not pass!
        let verified_again =
            vaa::verify_only_once_test_only(&mut token_bridge_state, verified);

        // Clean up.
        wormhole::vaa::destroy(verified_again);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

}
