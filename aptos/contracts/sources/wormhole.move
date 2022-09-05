module wormhole::wormhole {
    use aptos_framework::account;
    use wormhole::structs::{create_guardian, create_guardian_set};
    use wormhole::state;
    use deployer::deployer;
    use wormhole::u16;
    use wormhole::u32::{Self, U32};
    use wormhole::emitter;

// -----------------------------------------------------------------------------
// Sending messages

    public fun publish_message(
        emitter_cap: &mut emitter::EmitterCapability,
        nonce: u64,
        payload: vector<u8>,
        // TODO(csongor): this is an instant finality chain. Does it even make
        // sense to expose this argument? We could just set it to 0 or 1 internally.
        consistency_level: u8,
    ) {
        state::publish_event(
            emitter::get_emitter(emitter_cap),
            emitter::use_sequence(emitter_cap),
            nonce,
            payload,
            consistency_level
        );
    }

// -----------------------------------------------------------------------------
// Emitter registration

    public fun register_emitter(): emitter::EmitterCapability {
        state::new_emitter()
    }

// -----------------------------------------------------------------------------
// Contract initialization

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
        init_internal(
            signer_cap,
            chain_id,
            governance_chain_id,
            governance_contract,
            initial_guardian,
            u32::from_u64(86400),
        )
    }

    fun init_internal(
        signer_cap: account::SignerCapability,
        chain_id: u64,
        governance_chain_id: u64,
        governance_contract: vector<u8>,
        initial_guardian: vector<u8>,
        guardian_set_expiry: U32,
    ) {
        let wormhole = account::create_signer_with_capability(&signer_cap);
        state::init_wormhole_state(
            &wormhole,
            u16::from_u64(chain_id),
            u16::from_u64(governance_chain_id),
            governance_contract,
            guardian_set_expiry,
            signer_cap
        );
        state::init_message_handles(&wormhole);
        state::store_guardian_set(
            create_guardian_set(
                u32::from_u64(0),
                vector[create_guardian(initial_guardian)]
            )
        );
    }

    #[test_only]
    /// Initialise a dummy contract for testing. Returns the wormhole signer.
    public fun init_test(
        user: &signer,
        chain_id: u64,
        governance_chain_id: u64,
        governance_contract: vector<u8>,
        initial_guardian: vector<u8>,
    ): signer {
        let (wormhole, signer_cap) = account::create_resource_account(user, b"wormhole");
        init_internal(
            signer_cap,
            chain_id,
            governance_chain_id,
            governance_contract,
            initial_guardian,
            u32::from_u64(86400),
        );
        wormhole
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
