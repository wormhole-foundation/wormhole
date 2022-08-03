module Wormhole::Wormhole {
    use Wormhole::Governance::{init_guardian_set, GuardianSet};
    use Wormhole::State::{initMesssageHandles, storeGuardianSet, setChainId, setGovernanceChainId, setGovernanceContract};

    use 0x1::event::{Self, EventHandle};
    use 0x1::signer::{address_of};

    fun init(admin: &signer, initialGuardianSet:GuardianSet, chainId: u64, governanceChainId: u64, governanceContract: vector<u8>) {
        init_guardian_set(admin);
        initMesssageHandles(admin);

        storeGuardianSet(initialGuardianSet, 0);
        // initial guardian set index is 0, which is the default value of the storage slot anyways
        setChainId(chainId);
        setGovernanceChainId(governanceChainId);
        setGovernanceContract(governanceContract);
    }
}

