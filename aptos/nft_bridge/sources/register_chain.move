module nft_bridge::register_chain {

    use wormhole::u16::{Self, U16};
    use wormhole::cursor;
    use wormhole::deserialize;
    use wormhole::vaa;
    use wormhole::external_address::{Self, ExternalAddress};

    use nft_bridge::vaa as nft_bridge_vaa;
    use nft_bridge::state;

    /// "NFTBridge" (left padded)
    const NFT_BRIDGE: vector<u8> = x"00000000000000000000000000000000000000000000004e4654427269646765";

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

        assert!(target_module == NFT_BRIDGE, E_INVALID_MODULE);

        let action = deserialize::deserialize_u8(&mut cur);
        assert!(action == 0x01, E_INVALID_ACTION);

        // NOTE: currently we only allow VAAs targeting the "0" chain (which is
        // how registration VAAs are produced via governance.)  Technically it
        // would be possible to produce a VAA targeting only a single chain, but
        // it's unclear if that would ever happen in practice.
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
        nft_bridge_vaa::replay_protect(&vaa);

        let register_chain = parse_payload(vaa::destroy(vaa));

        state::set_registered_emitter(get_emitter_chain_id(&register_chain), get_emitter_address(&register_chain));
        register_chain
    }

    public entry fun submit_vaa_entry(vaa: vector<u8>) {
        submit_vaa(vaa);
    }

}

#[test_only]
module nft_bridge::register_chain_test {
    use std::option;
    use wormhole::u16;
    use nft_bridge::register_chain;
    use wormhole::vaa;
    use wormhole::wormhole;
    use wormhole::external_address;
    use nft_bridge::nft_bridge;
    use nft_bridge::state;

    /// Registration VAA for the etheruem NFT bridge 0xdeadbeef
    const ETHEREUM_NFT_REG: vector<u8> = x"0100000000010062e307224ed7a222234012fe1cd38450076ef30eb488a1cfac75ec16476d750959c9851e7549aeb16a99c73d381a9c416657567ae6ae46613b9236d511a8a5b601000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000162f5340000000000000000000000000000000000000000000000004e4654427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    /// Registration VAA for the etheruem NFT bridge 0xbeefface
    const ETHEREUM_NFT_REG_2: vector<u8> = x"01000000000100d7984b0abd82cbf9e39b74d16ada7cb43c9476fd4b9656a02f40eca2d1ffa560049dc5265946a88e7f643f8321cd5417e388b86580ed4c3a03f73d5599d4a9ed010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000000312bca0000000000000000000000000000000000000000000000004e4654427269646765010000000200000000000000000000000000000000000000000000000000000000beefface";

    /// Registration VAA for the etheruem token bridge 0xdeadbeef
    const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

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
        nft_bridge::init_test(deployer);
    }

    #[test]
    public fun test_parse() {
        let vaa = vaa::parse_test(ETHEREUM_NFT_REG);
        let register_chain = register_chain::parse_payload_test(vaa::destroy(vaa));
        let chain = register_chain::get_emitter_chain_id(&register_chain);
        let address = register_chain::get_emitter_address(&register_chain);

        assert!(chain == u16::from_u64(ETH_ID), 0);
        assert!(address == external_address::from_bytes(x"deadbeef"), 0);

    }

    #[test]
    #[expected_failure(abort_code = 0, location = nft_bridge::register_chain)]
    public fun test_parse_fail() {
        let vaa = vaa::parse_test(ETHEREUM_TOKEN_REG);
        // this should fail because it's an token bridge registration
        let _register_chain = register_chain::parse_payload_test(vaa::destroy(vaa));
    }

    #[test(deployer = @deployer)]
    public fun test_registration(deployer: &signer) {
        setup(deployer);

        register_chain::submit_vaa(ETHEREUM_NFT_REG);
        let address = state::get_registered_emitter(u16::from_u64(ETH_ID));
        assert!(address == option::some(external_address::from_bytes(x"deadbeef")), 0);
    }

    #[test(deployer = @deployer)]
    #[expected_failure(abort_code = 25607, location = 0x1::table)]
    public fun test_replay_protect(deployer: &signer) {
        setup(deployer);

        register_chain::submit_vaa(ETHEREUM_NFT_REG);
        register_chain::submit_vaa(ETHEREUM_NFT_REG);
    }

    #[test(deployer = @deployer)]
    public fun test_re_registration(deployer: &signer) {
        test_registration(deployer);

        // we register aptos again, which overrides the
        // previous one. This deviates from other chains (where this is
        // rejected), but I think this is the right behaviour.
        register_chain::submit_vaa(ETHEREUM_NFT_REG_2);
        let address = state::get_registered_emitter(u16::from_u64(ETH_ID));
        assert!(address == option::some(external_address::from_bytes(x"beefface")), 0);
    }
}
