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
    use sui::table::{Self};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::vaa::{Self, VAA};

    use token_bridge::state::{Self, State};

    friend token_bridge::create_wrapped;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;

    /// For a given chain ID, Token Bridge is non-existent.
    const E_UNREGISTERED_EMITTER: u64 = 0;
    /// Encoded emitter address does not match registered Token Bridge.
    const E_EMITTER_ADDRESS_MISMATCH: u64 = 1;

    /// This type represents VAA data whose emitter is a registered Token Bridge
    /// emitter. This message is also representative of a VAA that cannot be
    /// replayed.
    struct TokenBridgeMessage {
        /// Wormhole chain ID from which network the message originated from.
        emitter_chain: u16,
        /// Address of Token Bridge (standardized to 32 bytes) that produced
        /// this message.
        emitter_address: ExternalAddress,
        /// Sequence number of Token Bridge's Wormhole message.
        sequence: u64,
        /// Token Bridge payload.
        payload: vector<u8>
    }

    /// Parses and verifies encoded VAA. Because Token Bridge does not allow
    /// VAAs to be replayed, the VAA hash is stored in a set, which is checked
    /// against the next time the same VAA is used to make sure it cannot be
    /// used again.
    ///
    /// In its verification, this method checks whether the emitter is a
    /// registered Token Bridge contract on another network.
    ///
    /// NOTE: It is important for integrators to refrain from calling this
    /// method within their contracts. This method is meant to be called within
    /// a transaction block, passing the `TokenBridgeMessage` to one of the
    /// Token Bridge methods that consumes this type. If in a circumstance where
    /// this module has a breaking change in an upgrade, another method  (e.g.
    /// `complete_transfer_with_payload`) will not be affected by this change.
    public fun verify_only_once(
        token_bridge_state: &mut State,
        verified_vaa: VAA
    ): TokenBridgeMessage {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        // First parse and verify VAA using Wormhole. This also consumes the VAA
        // hash to prevent replay.
        vaa::consume(
            state::borrow_mut_consumed_vaas(&latest_only, token_bridge_state),
            &verified_vaa
        );

        // Does the emitter agree with a registered Token Bridge?
        assert_registered_emitter(token_bridge_state, &verified_vaa);

        // Take emitter info, sequence and payload.
        let sequence = vaa::sequence(&verified_vaa);
        let (
            emitter_chain,
            emitter_address,
            payload
        ) = vaa::take_emitter_info_and_payload(verified_vaa);

        TokenBridgeMessage {
            emitter_chain,
            emitter_address,
            sequence,
            payload
        }
    }

    public fun emitter_chain(self: &TokenBridgeMessage): u16 {
        self.emitter_chain
    }

    public fun emitter_address(self: &TokenBridgeMessage): ExternalAddress {
        self.emitter_address
    }

    public fun sequence(self: &TokenBridgeMessage): u64 {
        self.sequence
    }

    /// Destroy `TokenBridgeMessage` and extract payload, which is the same
    /// payload in the `VAA`.
    ///
    /// NOTE: This is a privileged method, which only friends within the Token
    /// Bridge package can use. This guarantees that no other package can redeem
    /// a VAA intended for Token Bridge as a denial-of-service by calling
    /// `verify_only_once` and then destroying it by calling it this method.
    public(friend) fun take_payload(msg: TokenBridgeMessage): vector<u8> {
        let TokenBridgeMessage {
            emitter_chain: _,
            emitter_address: _,
            sequence: _,
            payload
        } = msg;

        payload
    }

    /// Assert that a given emitter equals one that is registered as a foreign
    /// Token Bridge.
    fun assert_registered_emitter(
        token_bridge_state: &State,
        verified_vaa: &VAA
    ) {
        let chain = vaa::emitter_chain(verified_vaa);
        let registry = state::borrow_emitter_registry(token_bridge_state);
        assert!(table::contains(registry, chain), E_UNREGISTERED_EMITTER);

        let registered = table::borrow(registry, chain);
        let emitter_addr = vaa::emitter_address(verified_vaa);
        assert!(*registered == emitter_addr, E_EMITTER_ADDRESS_MISMATCH);
    }

    #[test_only]
    public fun destroy(msg: TokenBridgeMessage) {
        take_payload(msg);
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
    #[expected_failure(abort_code = vaa::E_UNREGISTERED_EMITTER)]
    fun test_cannot_verify_only_once_unregistered_chain() {
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let token_bridge_state = take_state(scenario);

        let verified_vaa = parse_and_verify_vaa(scenario, VAA);
        // You shall not pass!
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Clean up.
        vaa::destroy(msg);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_EMITTER_ADDRESS_MISMATCH)]
    fun test_cannot_verify_only_once_emitter_address_mismatch() {
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
        token_bridge::register_chain::register_new_emitter_test_only(
            &mut token_bridge_state,
            emitter_chain,
            emitter_addr
        );

        // Confirm that encoded emitter disagrees with registered emitter.
        let verified_vaa = parse_and_verify_vaa(scenario, VAA);
        assert!(
            wormhole::vaa::emitter_address(&verified_vaa) != emitter_addr,
            0
        );

        // You shall not pass!
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Clean up.
        vaa::destroy(msg);

        abort 42
    }

    #[test]
    fun test_verify_only_once() {
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
        let verified_vaa = parse_and_verify_vaa(scenario, VAA);
        assert!(
            wormhole::vaa::emitter_chain(&verified_vaa) == expected_source_chain,
            0
        );

        // Finally verify.
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Clean up.
        vaa::destroy(msg);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wormhole::set::E_KEY_ALREADY_EXISTS)]
    fun test_cannot_verify_only_once_again() {
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
        let verified_vaa = parse_and_verify_vaa(scenario, VAA);
        assert!(
            wormhole::vaa::emitter_chain(&verified_vaa) == expected_source_chain,
            0
        );

        // Finally verify.
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);
        vaa::destroy(msg);

        let verified_vaa = parse_and_verify_vaa(scenario, VAA);
        // You shall not pass!
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Clean up.
        vaa::destroy(msg);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_verify_only_once_outdated_version() {
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

        // Verify VAA.
        let verified_vaa = parse_and_verify_vaa(scenario, VAA);

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut token_bridge_state);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        state::migrate_version_test_only(
            &mut token_bridge_state,
            token_bridge::version_control::previous_version_test_only(),
            token_bridge::version_control::next_version()
        );

        // You shall not pass!
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Clean up.
        vaa::destroy(msg);

        abort 42
    }

}
