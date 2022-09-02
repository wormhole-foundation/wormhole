module wormhole::wormhole {
    use aptos_framework::account;
    use wormhole::structs::{create_guardian, create_guardian_set};
    use wormhole::state::{
        init_message_handles,
        init_wormhole_state,
        store_guardian_set,
    };
    use deployer::deployer;
    use wormhole::u16;
    use wormhole::u32;

    // TODO(csongor): maybe merge the different capabilities into the same key
    // to reduce storage access

    // protect me at all cost
    // never expose the inner capability publicly
    struct WormholeCapability has key {
        signer_cap: account::SignerCapability
    }

    public entry fun init(
        deployer: &signer,
        chain_id: u64,
        governance_chain_id: u64,
        governance_contract: vector<u8>,
        initial_guardian: vector<u8>
    ) {
        // account::SignerCapability can't be copied, so once it's stored into
        // state, the init function can no longer be called (since
        // the deployer signer capability must have been unlocked).
        let signer_cap = deployer::claim_signer_capability(deployer, @wormhole);
        let wormhole = account::create_signer_with_capability(&signer_cap);

        init_wormhole_state(
            &wormhole,
            u16::from_u64(chain_id),
            u16::from_u64(governance_chain_id),
            governance_contract,
            signer_cap
        );
        init_message_handles(&wormhole);
        store_guardian_set(create_guardian_set(u32::from_u64(0), vector[create_guardian(initial_guardian)]));
    }

}

#[test_only]
module wormhole::wormhole_test {
    use 0x1::hash;
    use 0x1::aptos_hash;
    #[test]
    public fun test_hash() {
        assert!(hash::sha3_256(vector[0]) == x"5d53469f20fef4f8eab52b88044ede69c77a6a68a60728609fc4a65ff531e7d0", 0);
        assert!(aptos_hash::keccak256(vector[0]) == x"bc36789e7a1e281436464229828f817d6612f7b477d66591ff96a9e064bcc98a", 0);
    }
}
