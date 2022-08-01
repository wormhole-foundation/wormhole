module Wormhole::NFTBridge{
    use 0x1::table::{Self, Table, new, add, borrow, borrow_mut};
    use 0x1::token::{Self, create_unlimited_collection_script}; //Non-fungible token
    //use Std::string::String;
    //use Std::ACL;
    use 0x1::vector::{Self};
    use 0x1::signer::{Self};
    
    struct Transfer has drop, store {
        tokenAddress: vector<u8>, 
        tokenChain: vector<u8>,  
        symbol: u64, 
        name: vector<u8>,  
        tokenID: u128, //u256 
        uri: vector<u8>,  //string 
        to: vector<u8>, 
        toChain: u64, //u16 
    }
    
    struct RegisterChain { 
        nft_bridge_module: vector<u8>, 
        action: u8, 
        chainId: u64, //u16 
        emitterChainID: u64, //u16 
        emitterAddress: vector<u8>,
    }

    struct UpgradeContract {
        nft_bridge_module: vector<u8>,
        action: u8,
        chainId: u64, 
        // Address of the new contract
        newContract: vector<u8>,
    }

    struct Provider has key, store{
        chainId: u64,  
        governanceChainId: u64, 
        finality: u8, 
        governanceContract: vector<u8>, 
    }

    struct Asset has key{
        chainId: u64, //u16 
        assetAddress: vector<u8>, 
    }

    struct SPLCache has key, store{
        name: vector<u8>, 
        symbol: vector<u8>, 
    }

    struct State has key{
        wormhole: address,    
        tokenImplementation: address, 
        provider: Provider,   

        consumedGovernanceActions: Table<vector<u8>, bool>, 

        completedTransfers: Table<vector<u8>, bool>, 

        // Mapping of initialized implementations
        initializedImplementations: Table<address, bool>,

        // Mapping of wrapped assets (chainID => nativeAddress => wrappedAddress)
        wrappedAssets: Table<u64, Table<vector<u8>, address>>, 

        // Mapping to safely identify wrapped assets
        isWrappedAsset: Table<address, bool>, 

        // Mapping of bridge contracts on other chains
        bridgeImplementations: Table<u64, vector<u8>>,

        // Mapping of spl token info caches (chainID => nativeAddress => SPLCache)
        splCache: Table<u128, SPLCache>,  //u256 => SPLCache
    }

    public fun setup(
        admin: &signer, 
        implementation: address,
        chainId: u64,
        wormhole: address,
        governanceChainId: u64, 
        governanceContract: vector<u8>,
        tokenImplementation: address, 
        finality: u8,
    ){  
        assert!(!exists<State>(signer::address_of(admin)), 0);

        let provider = Provider {
            chainId, 
            governanceChainId, 
            finality, 
            governanceContract,
        };
        
        let state = State { 
             wormhole:                      wormhole, 
             tokenImplementation:           tokenImplementation,
             provider:                      provider, 
             consumedGovernanceActions:     new<vector<u8>, bool>(),
             completedTransfers:            new<vector<u8>, bool>(),
             initializedImplementations:    new<address, bool>(),
             wrappedAssets:                 new<u64, Table<vector<u8>, address>>(),
             isWrappedAsset:                new<address, bool>(),
             bridgeImplementations:         new<u64, vector<u8>>(),
             splCache:                      new<u128, SPLCache>(),
        };
        move_to(admin, state);
    }

    // setters
    public fun setWrappedAsset(tokenChainId: u64, tokenAddress: vector<u8>, wrapper: address) acquires State{
        let state = borrow_global_mut<State>(@Wormhole);
        let inner = borrow_mut<u64, Table<vector<u8>, address>>(&mut state.wrappedAssets, tokenChainId);
        add<vector<u8>, address>(inner, tokenAddress, wrapper);
        add(&mut  state.isWrappedAsset, wrapper, true);
    }

    //getters 
    public fun governanceActionIsConsumed(hash: vector<u8>): bool acquires State{
        let inner = &borrow_global<State>(@Wormhole).consumedGovernanceActions;
        let res = *table::borrow(inner, hash);
        res
    } 
    
    // // TODO: check collection name to see if it is initialized, not address
    // public fun isInitialized(address impl) public view returns (bool) {
    //     return _state.initializedImplementations[impl];
    // }

    public fun isTransferCompleted(hash: vector<u8>): bool acquires State{
        let inner = &borrow_global<State>(@Wormhole).completedTransfers;
        let res = *table::borrow(inner, hash);
        res
    }

    public fun wormhole(): address acquires State {
        borrow_global<State>(@Wormhole).wormhole
    }

    public fun chainId():u64 acquires State { //u16
        let state = borrow_global<State>(@Wormhole);
        state.provider.chainId
    }

    // function governanceChainId() public view returns (uint16){
    //     return _state.provider.governanceChainId;
    // }

    public fun governanceContract(): vector<u8> acquires State{
        borrow_global<State>(@Wormhole).provider.governanceContract
    }

    //public fun wrappedAsset(tokenChainId: u64, tokenAddress: vector<u8>){
    //     return _state.wrappedAssets[tokenChainId][tokenAddress];
    //}

    // function bridgeContracts(uint16 chainId_) public view returns (bytes32){
    //     return _state.bridgeImplementations[chainId_];
    // }

    // function tokenImplementation() public view returns (address){
    //     return _state.tokenImplementation;
    // }

    // function isWrappedAsset(address token) public view returns (bool){
    //     return _state.isWrappedAsset[token];
    // }

    // function splCache(uint256 tokenId) public view returns (NFTBridgeStorage.SPLCache memory) {
    //     return _state.splCache[tokenId];
    // }

    // function finality() public view returns (uint8) {
    //     return _state.provider.finality;
    // }

    public(script) fun createWrapped(admin: &signer, tokenChain: u64, tokenAddress: vector<u8>, name: vector<u8>, symbol: vector<u8>) acquires State{
        assert!(tokenChain != chainId(), 0);
        //assert!(wrappedAsset(tokenChain, tokenAddress, name) == address(0), 0);
        
        // SPL NFTs all use the same NFT contract, so unify the name
        if (tokenChain == 1) {
            // "Wormhole Bridged Solana-NFT" - right-padded
            //name =   0x576f726d686f6c65204272696467656420536f6c616e612d4e46540000000000;
            // "WORMSPLNFT" - right-padded
            //symbol = 0x574f524d53504c4e465400000000000000000000000000000000000000000000;
        };
        
        create_unlimited_collection_script(admin, name, vector::empty<u8>(), vector::empty<u8>());

        // TODO - set wrapped Asset
        //setWrappedAsset(tokenChain, tokenAddress, token);
    }

}
