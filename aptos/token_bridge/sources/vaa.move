/// Token Bridge VAA utilities
module token_bridge::vaa {
    use wormhole::vaa::{Self, VAA};
    use token_bridge::bridge_state as state;

    friend token_bridge::bridge_implementation;
    friend token_bridge::contract_upgrade;
    friend token_bridge::register_chain;

    const E_UNKNOWN_EMITTER: u64 = 0;

    /// Aborts if the VAA has already been consumed. Marks the VAA as consumed
    /// the first time around.
    public(friend) fun replay_protect(vaa: &VAA) {
        // this calls set::add which aborts if the element already exists
        state::set_vaa_consumed(vaa::get_hash(vaa));
    }

    /// Asserts that the VAA is from a known token bridge.
    public fun assert_known_emitter(vm: &VAA) {
        assert!(
            state::get_registered_emitter(vaa::get_emitter_chain(vm)) == vaa::get_emitter_address(vm),
            E_UNKNOWN_EMITTER
        );
    }

    /// Parses, verifies, and replay protects a token bridge VAA.
    /// Aborts if the VAA is not from a known token bridge emitter.
    ///
    /// Has a 'friend' visibility so that it's only callable by the token bridge
    /// (otherwise the replay protection could be abused to DoS the bridge)
    public(friend) fun parse_verify_and_replay_protect(vaa: vector<u8>): VAA {
        let vaa = vaa::parse_and_verify(vaa);
        assert_known_emitter(&vaa);
        replay_protect(&vaa);
        vaa
    }
}

#[test_only]
module token_bridge::vaa_test {

}
