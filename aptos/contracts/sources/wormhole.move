module wormhole::wormhole {
    use aptos_framework::account;
    use wormhole::structs::{create_guardian, create_guardian_set};
    use wormhole::state::{
        init_message_handles,
        init_wormhole_state,
        store_guardian_set,
        set_governance_contract,
        set_chain_id,
        set_governance_chain_id,
    };
    use deployer::deployer;
    use wormhole::u16;
    use wormhole::u32;

    friend wormhole::contract_upgrade;

    // TODO(csongor): maybe merge the different capabilities into the same key
    // to reduce storage access

    // protect me at all cost
    // never expose the inner capability publicly
    struct WormholeCapability has key {
        signer_cap: account::SignerCapability
    }

    public entry fun init(
        deployer: &signer,
        chainId: u64,
        governance_chain_id: u64,
        governance_contract: vector<u8>,
        initial_guardian: vector<u8>
    ) {
        // account::SignerCapability can't be copied, so once it's stored into
        // WormholeCapability, the init function can no longer be called (since
        // the deployer signer capability must have been unlocked).
        let signer_cap = deployer::claim_signer_capability(deployer, @wormhole);
        let wormhole = account::create_signer_with_capability(&signer_cap);
        move_to(&wormhole, WormholeCapability { signer_cap });

        init_wormhole_state(&wormhole);
        init_message_handles(&wormhole);
        store_guardian_set(create_guardian_set(u32::from_u64(0), vector[create_guardian(initial_guardian)]));
        // initial guardian set index is 0, which is the default value of the storage slot anyways

        set_chain_id(u16::from_u64(chainId));
        set_governance_chain_id(u16::from_u64(governance_chain_id));
        set_governance_contract(governance_contract);
    }

    public(friend) fun wormhole_signer(): signer acquires WormholeCapability {
        account::create_signer_with_capability(&borrow_global<WormholeCapability>(@wormhole).signer_cap)
    }

}

#[test_only]
module wormhole::wormhole_test {
    use 0x1::hash;
    #[test]
    public fun test_foo() {
        assert!(hash::sha3_256(vector[0]) == x"5d53469f20fef4f8eab52b88044ede69c77a6a68a60728609fc4a65ff531e7d0", 0);
        // TODO: once keccak_256 is available, uncomment this line
        // assert!(hash::keccak_256(vector[0]) == x"bc36789e7a1e281436464229828f817d6612f7b477d66591ff96a9e064bcc98a", 0);
    }
}
