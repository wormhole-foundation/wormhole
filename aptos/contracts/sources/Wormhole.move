module Wormhole::Wormhole {
    use 0x1::acl::{Self};
    use 0x1::signer::{address_of};
    //use Wormhole::Governance::{init_guardian_set};
    use Wormhole::Structs::{GuardianSet};
    use Wormhole::State::{initMessageHandles, initWormholeState, storeGuardianSet, setChainId, setGovernanceChainId, setGovernanceContract};

    use 0x1::event::{Self, EventHandle};

    fun init(admin: &signer, initialGuardianSet:GuardianSet, chainId: u64, governanceChainId: u64, governanceContract: vector<u8>) {
        // init_guardian_set(admin); - this function seems unnecessary
        assert!(address_of(admin)==@Wormhole, 0);
        initWormholeState(admin);
        initMessageHandles(admin);
        storeGuardianSet(initialGuardianSet, 0);
        // initial guardian set index is 0, which is the default value of the storage slot anyways
        setChainId(chainId);
        setGovernanceChainId(governanceChainId);
        setGovernanceContract(governanceContract);
    }
}

