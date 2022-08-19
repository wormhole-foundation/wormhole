module Wormhole::Wormhole {
    use 0x1::signer::{address_of};
    //use 0x1::vector::{Self};
    //use Wormhole::Governance::{init_guardian_set};
    //use Wormhole::Structs::{GuardianSet, createGuardianSet};
    use Wormhole::State::{initMessageHandles, initWormholeState, storeGuardianSet, setChainId, setGovernanceChainId, setGovernanceContract};

    public entry fun init(admin: &signer, chainId: u64, governanceChainId: u64, governanceContract: vector<u8>) {
        // init_guardian_set(admin); - this function seems unnecessary
        //assert!(address_of(admin)==@Wormhole, 0);
        initWormholeState(admin);
        initMessageHandles(admin);
        //storeGuardianSet(createGuardianSet(0, vector::empty()), 0);
        // initial guardian set index is 0, which is the default value of the storage slot anyways
        setChainId(chainId);
        setGovernanceChainId(governanceChainId);
        setGovernanceContract(governanceContract);
    }

    public entry fun testInit(admin: &signer){ 
        setChainId(3);
    }

    public entry fun doNothing(admin: &signer){ 
        //setChainId(3);
    }

    // public entry fun testEntry2(){ 
    //     setChainId(4);
    // }
}

