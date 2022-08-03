module Wormhole::State{
    use 0x1::table::{Self, Table};
    //includes getters and setters
    use Wormhole::Governance::{GuardianSet};

    struct GuardianSetChanged has store, drop{
        oldGuardianIndex: u64, //should be u32
        newGuardianIndex: u64, //should be u32
    } 

    struct WormholeMessage has store, drop{
        sender: address, 
        sequence: u64,  
        nonce: u64, //should be u32 
        payload: vector<u8>,
        consistencyLevel: u8,
    }

	struct Provider has key, store{
        chainId: u64, // u16
		governanceChainId: u64, // U16
        governanceContract: vector<u8>, //bytes32
	}

    struct WormholeState has key{
        provider: Provider,

        // Mapping of guardian_set_index => guardian set
        guardianSets: Table<u64, GuardianSet>,

        // Current active guardian set index
        guardianSetIndex: u64,  //should be u32

        // Period for which a guardian set stays active after it has been replaced
        guardianSetExpiry: u64, //should be u32

        // Sequence numbers per emitter
        sequences: Table<address, u64>,

        // Mapping of consumed governance actions
        consumedGovernanceActions: Table<vector<u8>, bool>,

        // Mapping of initialized implementations
        initializedImplementations: Table<address, bool>,

        messageFee: u128, //should be u256
    }

    public fun updateGuardianSetIndex(newIndex: u64) acquires WormholeState { //should be u32
        let state = borrow_global_mut<WormholeState>(@Wormhole);
        state.guardianSetIndex = newIndex;
    }

    // TODO
    // public fun expireGuardianSet(index: u64){
    //     let state = borrow_global_mut<WormholeState>(@Wormhole);
    //     let inner = borrow_mut<u64, Table<vector<u8>, String>>(&mut state.wrappedAssets, tokenChainId);
    //        _state.guardianSets[index].expirationTime = uint32(block.timestamp) + 86400;
    // }

    public fun storeGuardianSet(set: GuardianSet, index: u64) acquires WormholeState{ 
        let state = borrow_global_mut<WormholeState>(@Wormhole);
        table::add(&mut state.guardianSets, index, set);
    }

    //TODO: what is analogue of setInitialized?
    // function setInitialized(address implementatiom) internal {
    //     _state.initializedImplementations[implementatiom] = true;
    // }

    public fun setGovernanceActionConsumed(hash: vector<u8>) acquires WormholeState{
        let state = borrow_global_mut<WormholeState>(@Wormhole);
        table::add(&mut state.consumedGovernanceActions, hash, true);
    }

    public fun setChainId(chaindId: u64) acquires WormholeState{
        let state = borrow_global_mut<WormholeState>(@Wormhole);
        let provider = &mut state.provider;
        provider.chainId = chaindId;
    }

    public fun setGovernanceChainId(chainId: u64) acquires WormholeState{
        let state = borrow_global_mut<WormholeState>(@Wormhole);
        let provider = &mut state.provider;
        provider.governanceChainId = chainId;
    }

    // function setGovernanceChainId(uint16 chainId) internal {
    //     _state.provider.governanceChainId = chainId;
    // }

    // function setGovernanceContract(bytes32 governanceContract) internal {
    //     _state.provider.governanceContract = governanceContract;
    // }

    // function setMessageFee(uint256 newFee) internal {
    //     _state.messageFee = newFee;
    // }

    // function setNextSequence(address emitter, uint64 sequence) internal {
    //     _state.sequences[emitter] = sequence;
    // }


}