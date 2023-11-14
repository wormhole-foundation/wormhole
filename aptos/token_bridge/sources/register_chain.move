module token_bridge::register_chain {

    use wormhole::u16::{Self, U16};
    use wormhole::cursor;
    use wormhole::deserialize;
    use wormhole::vaa;
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::vaa as token_bridge_vaa;
    use token_bridge::state;

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

    public fun get_emitter_chain_id(a: &RegisterChain): U16 {
        a.emitter_chain_id
    }

    public fun get_emitter_address(a: &RegisterChain): ExternalAddress {
        a.emitter_address
    }

    #[test_only]
    public fun parse_payload_test(payload: vector<u8>): RegisterChain {
        parse_payload(payload)
    }

    fun parse_payload(payload: vector<u8>): RegisterChain {
        let cur = cursor::init(payload);
        let target_module = deserialize::deserialize_vector(&mut cur, 32);

        assert!(target_module == TOKEN_BRIDGE, E_INVALID_MODULE);

        let action = deserialize::deserialize_u8(&mut cur);
        assert!(action == 0x01, E_INVALID_ACTION);

        // TODO(csongor): should we also accept a VAA targeting aptos directly?
        // why would a registration VAA target a specific chain?
        let target_chain = deserialize::deserialize_u16(&mut cur);
        assert!(target_chain == u16::from_u64(0x0), E_INVALID_TARGET);

        let emitter_chain_id = deserialize::deserialize_u16(&mut cur);

        let emitter_address = external_address::deserialize(&mut cur);

        cursor::destroy_empty(cur);

        RegisterChain { emitter_chain_id, emitter_address }
    }

    public fun submit_vaa(vaa: vector<u8>): RegisterChain {
        let vaa = vaa::parse_and_verify(vaa);
        vaa::assert_governance(&vaa); // not tested
        token_bridge_vaa::replay_protect(&vaa);

        let register_chain = parse_payload(vaa::destroy(vaa));

        state::set_registered_emitter(get_emitter_chain_id(&register_chain), get_emitter_address(&register_chain));
        register_chain
    }

    public entry fun submit_vaa_entry(vaa: vector<u8>) {
        submit_vaa(vaa);
    }

}

#[test_only]
module token_bridge::register_chain_test {
    use std::option;
    use wormhole::u16;
    use token_bridge::register_chain;
    use wormhole::vaa;
    use wormhole::wormhole;
    use wormhole::external_address;
    use token_bridge::token_bridge;
    use token_bridge::state;

    /// Registration VAA for the ethereum token bridge 0xdeadbeef
    const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    /// Another registration VAA for the ethereum token bridge, 0xbeefface
    const ETHEREUM_TOKEN_REG_2:vector<u8> = x"01000000000100c2157fa1c14957dff26d891e4ad0d993ad527f1d94f603e3d2bb1e37541e2fbe45855ffda1efc7eb2eb24009a1585fa25a267815db97e4a9d4a5eb31987b5fb40100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000017ca43300000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000beefface";

    /// Registration VAA for the ethereum NFT bridge 0xdeadbeef
    const ETHEREUM_NFT_REG: vector<u8> = x"0100000000010066cce2cb12d88c97d4975cba858bb3c35d6430003e97fced46a158216f3ca01710fd16cc394441a08fef978108ed80c653437f43bb2ca039226974d9512298b10000000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000018483540000000000000000000000000000000000000000000000004e4654427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    const ETH_ID: u64 = 2;

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

    #[test]
    public fun test_parse() {
        let vaa = vaa::parse_test(ETHEREUM_TOKEN_REG);
        let register_chain = register_chain::parse_payload_test(vaa::destroy(vaa));
        let chain = register_chain::get_emitter_chain_id(&register_chain);
        let address = register_chain::get_emitter_address(&register_chain);

        assert!(chain == u16::from_u64(ETH_ID), 0);
        assert!(address == external_address::from_bytes(x"deadbeef"), 0);

    }

    #[test]
    #[expected_failure(abort_code = 0, location = token_bridge::register_chain)]
    public fun test_parse_fail() {
        let vaa = vaa::parse_test(ETHEREUM_NFT_REG);
        // this should fail because it's an NFT registration
        let _register_chain = register_chain::parse_payload_test(vaa::destroy(vaa));

    }

    #[test(deployer = @deployer)]
    public fun test_registration(deployer: &signer) {
        setup(deployer);

        register_chain::submit_vaa(ETHEREUM_TOKEN_REG);
        let address = state::get_registered_emitter(u16::from_u64(ETH_ID));
        assert!(address == option::some(external_address::from_bytes(x"deadbeef")), 0);
    }

    #[test(deployer = @deployer)]
    #[expected_failure(abort_code = 25607, location = 0x1::table)]
    public fun test_replay_protect(deployer: &signer) {
        setup(deployer);

        register_chain::submit_vaa(ETHEREUM_TOKEN_REG);
        register_chain::submit_vaa(ETHEREUM_TOKEN_REG);
    }

    #[test(deployer = @deployer)]
    public fun test_re_registration(deployer: &signer) {
        test_registration(deployer);

        // TODO(csongor): we register ethereum again, which overrides the
        // previous one. This deviates from other chains (where this is
        // rejected), but I think this is the right behaviour.
        // Easy to change, should be discussed.
        register_chain::submit_vaa(ETHEREUM_TOKEN_REG_2);
        let address = state::get_registered_emitter(u16::from_u64(ETH_ID));
        assert!(address == option::some(external_address::from_bytes(x"beefface")), 0);
    }

}
