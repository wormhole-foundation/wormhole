module wormhole::wormhole {
    use 0x1::vector::{Self};
    use aptos_framework::account;
    use wormhole::structs::{create_guardian_set};
    use wormhole::state::{
        init_message_handles,
        init_wormhole_state,
        store_guardian_set,
        set_governance_contract,
        set_chain_id,
        set_governance_chain_id,
    };
    use wormhole::u16;
    use wormhole::u32;
    use deployer::deployer;
    use aptos_framework::code;

    // protect me at all cost
    // never expose the inner capability publicly
    struct WormholeCapability has key {
        signer_cap: account::SignerCapability
    }

    // TODO(csongor): Verifying an upgrade vaa should create one of these, which
    // the `upgrade` function could consume below.
    // Think about how to handle in-flight upgrades that may fail to deploy.
    // In particular, a newer upgrade VAA should be able to drop the previous one.
    struct UpgradeCapability has key {
        hash: vector<u8>
    }

    public entry fun init(deployer: &signer, chainId: u64, governance_chain_id: u64, governance_contract: vector<u8>) {
        // account::SignerCapability can't be copied, so once it's stored into
        // WormholeCapability, the init function can no longer be called (since
        // the deployer signer capability must have been unlocked).
        let signer_cap = deployer::claim_signer_capability(deployer, @wormhole);
        let wormhole = account::create_signer_with_capability(&signer_cap);
        move_to(&wormhole, WormholeCapability { signer_cap });

        init_wormhole_state(&wormhole);
        init_message_handles(&wormhole);
        //TODO: take guardian set as input also
        store_guardian_set(create_guardian_set(u32::from_u64(0), vector::empty()), u32::from_u64(0));
        // initial guardian set index is 0, which is the default value of the storage slot anyways

        set_chain_id(u16::from_u64(chainId));
        set_governance_chain_id(u16::from_u64(governance_chain_id));
        set_governance_contract(governance_contract);
    }

    fun wormhole_signer(): signer acquires WormholeCapability {
        account::create_signer_with_capability(&borrow_global<WormholeCapability>(@wormhole).signer_cap)
    }

    public entry fun upgrade(_anyone: &signer, metadata_serialized: vector<u8>, code: vector<vector<u8>>) acquires WormholeCapability {
        // TODO(csongor): gate this with `UpgradeCapability` above and check
        // that metadata_serialized's hash matches that
        let wormhole = wormhole_signer();
        code::publish_package_txn(&wormhole, metadata_serialized, code);
    }
}
