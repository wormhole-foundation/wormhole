/// Token Bridge VAA utilities
module token_bridge::vaa {
    use std::option;
    use wormhole::vaa::{Self, VAA};
    use token_bridge::state;

    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::contract_upgrade;
    friend token_bridge::register_chain;
    friend token_bridge::wrapped;

    #[test_only]
    friend token_bridge::vaa_test;

    /// We have no registration for this chain
    const E_UNKNOWN_CHAIN: u64 = 0;
    /// We have a registration, but it's different from what's given
    const E_UNKNOWN_EMITTER: u64 = 1;

    /// Aborts if the VAA has already been consumed. Marks the VAA as consumed
    /// the first time around.
    public(friend) fun replay_protect(vaa: &VAA) {
        // this calls set::add which aborts if the element already exists
        state::set_vaa_consumed(vaa::get_hash(vaa));
    }

    /// Asserts that the VAA is from a known token bridge.
    public fun assert_known_emitter(vm: &VAA) {
        let maybe_emitter = state::get_registered_emitter(vaa::get_emitter_chain(vm));
        assert!(option::is_some(&maybe_emitter), E_UNKNOWN_CHAIN);

        let emitter = option::extract(&mut maybe_emitter);
        assert!(emitter == vaa::get_emitter_address(vm), E_UNKNOWN_EMITTER);
    }

    /// Parses, verifies, and replay protects a token bridge VAA.
    /// Aborts if the VAA is not from a known token bridge emitter.
    ///
    /// Has a 'friend' visibility so that it's only callable by the token bridge
    /// (otherwise the replay protection could be abused to DoS the bridge)
    public(friend) fun parse_verify_and_replay_protect(vaa: vector<u8>): VAA {
        let vaa = parse_and_verify(vaa);
        replay_protect(&vaa);
        vaa
    }

    /// Parses, and verifies a token bridge VAA.
    /// Aborts if the VAA is not from a known token bridge emitter.
    public fun parse_and_verify(vaa: vector<u8>): VAA {
        let vaa = vaa::parse_and_verify(vaa);
        assert_known_emitter(&vaa);
        vaa
    }
}

#[test_only]
module token_bridge::vaa_test {
    use token_bridge::vaa;
    use token_bridge::state;
    use token_bridge::token_bridge;

    use wormhole::vaa as core_vaa;
    use wormhole::wormhole;
    use wormhole::u16;
    use wormhole::external_address;

    /// VAA sent from the ethereum token bridge 0xdeadbeef
    const VAA: vector<u8> = x"01000000000100102d399190fa61daccb11c2ea4f7a3db3a9365e5936bcda4cded87c1b9eeb095173514f226256d5579af71d4089eb89496befb998075ba94cd1d4460c5c57b84000000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000002634973000200000000000000000000000000000000000000000000000000000000beefface00020c0000000000000000000000000000000000000000000000000000000042454546000000000000000000000000000000000042656566206661636520546f6b656e";

    fun setup(deployer: &signer) {
        let aptos_framework = std::account::create_account_for_test(@aptos_framework);
        std::timestamp::set_time_has_started_for_testing(&aptos_framework);
        wormhole::init_test(
            22,
            1,
            x"0000000000000000000000000000000000000000000000000000000000000004",
            x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
            0
        );
        token_bridge::init_test(deployer);
    }

    #[test(deployer = @deployer)]
    #[expected_failure(abort_code = 0, location = token_bridge::vaa)] // E_UNKNOWN_CHAIN
    public fun test_unknown_chain(deployer: &signer) {
        setup(deployer);
        let vaa = vaa::parse_verify_and_replay_protect(VAA);
        core_vaa::destroy(vaa);
    }

    #[test(deployer = @deployer)]
    #[expected_failure(abort_code = 1, location = token_bridge::vaa)] // E_UNKNOWN_EMITTER
    public fun test_unknown_emitter(deployer: &signer) {
        setup(deployer);
        state::set_registered_emitter(
            u16::from_u64(2),
            external_address::from_bytes(x"deadbeed"), // not deadbeef
        );
        let vaa = vaa::parse_verify_and_replay_protect(VAA);
        core_vaa::destroy(vaa);
    }

    #[test(deployer = @deployer)]
    public fun test_known_emitter(deployer: &signer) {
        setup(deployer);
        state::set_registered_emitter(
            u16::from_u64(2),
            external_address::from_bytes(x"deadbeef"),
        );
        let vaa = vaa::parse_verify_and_replay_protect(VAA);
        core_vaa::destroy(vaa);
    }

    #[test(deployer = @deployer)]
    #[expected_failure(abort_code = 25607, location = 0x1::table)] // add_box error
    public fun test_replay_protect(deployer: &signer) {
        setup(deployer);
        state::set_registered_emitter(
            u16::from_u64(2),
            external_address::from_bytes(x"deadbeef"),
        );
        let vaa = vaa::parse_verify_and_replay_protect(VAA);
        core_vaa::destroy(vaa);
        let vaa = vaa::parse_verify_and_replay_protect(VAA);
        core_vaa::destroy(vaa);
    }

    #[test(deployer = @deployer)]
    public fun test_can_verify_after_replay_protect(deployer: &signer) {
        setup(deployer);
        state::set_registered_emitter(
            u16::from_u64(2),
            external_address::from_bytes(x"deadbeef"),
        );
        let vaa = vaa::parse_verify_and_replay_protect(VAA);
        core_vaa::destroy(vaa);
        let vaa = vaa::parse_and_verify(VAA);
        core_vaa::destroy(vaa);
    }
}
