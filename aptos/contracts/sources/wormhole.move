module wormhole::wormhole {
    use 0x1::vector::{Self};
    //use 0x1::vector::{Self};
    //use Wormhole::Governance::{init_guardian_set};
    use wormhole::structs::{create_guardian_set};
    use wormhole::state::{
        init_message_handles,
        init_wormhole_state,
        store_guardian_set,
        set_governance_contract
    };
    use wormhole::u32;

    public entry fun init(admin: &signer, _chainId: u64, _governance_chain_id: u64, governance_contract: vector<u8>) {
        // init_guardian_set(admin); - this function seems unnecessary
        //assert!(address_of(admin)==@wormhole, 0);
        init_wormhole_state(admin);
        init_message_handles(admin);
        store_guardian_set(create_guardian_set(u32::from_u64(0), vector::empty()), u32::from_u64(0));
        // initial guardian set index is 0, which is the default value of the storage slot anyways

        //TODO: set chainIds, which are U32 types. These can't be passed into an entry fun atm.
        //set_chain_id(chainId);
        //set_governance_chain_id(governance_chain_id);
        set_governance_contract(governance_contract);
    }

    public entry fun test_init_wormhole_state(admin: &signer){
        init_wormhole_state(admin);
    }

    public entry fun test_init_message_handles(admin: &signer){
         init_message_handles(admin);
    }
}

