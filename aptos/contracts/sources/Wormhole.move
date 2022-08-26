module Wormhole::Wormhole {
    use 0x1::vector::{Self};
    //use 0x1::vector::{Self};
    //use Wormhole::Governance::{init_guardian_set};
    use Wormhole::Structs::{createGuardianSet};
    use Wormhole::State::{initMessageHandles, initWormholeState, storeGuardianSet, setGovernanceContract};
    use Wormhole::u32;

    public entry fun init(admin: &signer, _chainId: u64, _governanceChainId: u64, governanceContract: vector<u8>) {
        // init_guardian_set(admin); - this function seems unnecessary
        //assert!(address_of(admin)==@Wormhole, 0);
        initWormholeState(admin);
        initMessageHandles(admin);
        storeGuardianSet(createGuardianSet(u32::from_u64(0), vector::empty()), u32::from_u64(0));
        // initial guardian set index is 0, which is the default value of the storage slot anyways

        //TODO: set chainIds, which are U32 types. These can't be passed into an entry fun atm.
        //setChainId(chainId);
        //setGovernanceChainId(governanceChainId);
        setGovernanceContract(governanceContract);
    }

    public entry fun testInitWormholeState(admin: &signer){
        initWormholeState(admin);
    }

    public entry fun testInitMessageHandles(admin: &signer){
         initMessageHandles(admin);
    }
}

