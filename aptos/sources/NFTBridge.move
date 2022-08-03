module Wormhole::NFTBridge{
    use 0x1::table::{Self, Table, new, add, borrow, borrow_mut};
    use 0x1::token::{Self, create_unlimited_collection_script}; //Non-fungible token
    //use Std::ACL;
    use 0x1::vector::{Self};
    use 0x1::signer::{Self};
    use 0x1::string::{Self, String};
    use Wormhole::Deserialize::{deserialize_u8, deserialize_u64, deserialize_u128, deserialize_vector};
    use Wormhole::VAA::{parse, parseAndVerifyVAA, get_payload, get_hash, destroy};
    
    struct Transfer has drop, store {
        tokenAddress: vector<u8>, 
        tokenChain: u64,//should be u16 - chain ID of the token  
        symbol: u64, 
        name: vector<u8>,  
        tokenID: u128, //should be u256 
        uri: String,  
        to: vector<u8>, 
        toChain: u64, //should be u16 
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
        // collection: String, 
        // creator: address, 
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

        // Mapping of wrapped assets (chainID => nativeAddress => name of asset collection as a string)
        wrappedAssets: Table<u64, Table<vector<u8>, String>>, 

        // Mapping to safely identify wrapped assets
        isWrappedAsset: Table<String, bool>, 

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
             wrappedAssets:                 new<u64, Table<vector<u8>, String>>(),
             isWrappedAsset:                new<String, bool>(),
             bridgeImplementations:         new<u64, vector<u8>>(),
             splCache:                      new<u128, SPLCache>(),
        };
        move_to(admin, state);
    }

    // setters
    public fun setWrappedAsset(tokenChainId: u64, tokenAddress: vector<u8>, wrapper: String) acquires State{
        let state = borrow_global_mut<State>(@Wormhole);
        let inner = borrow_mut<u64, Table<vector<u8>, String>>(&mut state.wrappedAssets, tokenChainId);
        add<vector<u8>, String>(inner, tokenAddress, wrapper);
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

    // public fun isWrappedAsset(address token) public view returns (bool){
    //     return _state.isWrappedAsset[token];
    // }

    // function splCache(uint256 tokenId) public view returns (NFTBridgeStorage.SPLCache memory) {
    //     return _state.splCache[tokenId];
    // }

    // function finality() public view returns (uint8) {
    //     return _state.provider.finality;
    // }

    public entry fun transferNFT(collection: String, name: String, recipientChain: u64, recipient:vector<u8>, nonce: u64){
    
    }

    public entry fun completeTransfer(admin: &signer, encodedVm: vector<u8>) acquires State{

        //TODO - complete this function
        
        let (vaa, valid, reason) = parseAndVerifyVAA(encodedVm);
        assert!(valid==true, 0);

        let transfer = parseTransfer(get_payload(&vaa));

        assert!(isTransferCompleted(get_hash(&vaa))==false, 0);

        assert!(chainId()==transfer.toChain, 0);

        
        if (transfer.tokenChain == chainId()){
            assert!(1==1, 0);
        } else{
            createWrapped(admin, transfer.tokenChain, transfer.tokenAddress, transfer.name, transfer.symbol);
            assert!(1==1, 0);
        };

        destroy(vaa);
    }


    public(script) fun createWrapped(admin: &signer, tokenChain: u64, tokenAddress: vector<u8>, name: vector<u8>, symbol: u64) acquires State{
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

    public entry fun parseTransfer(encoded: vector<u8>): Transfer {

        let (payloadID, encoded) = deserialize_u8(encoded);
        // require(payloadID == 1, "invalid Transfer");

        let (tokenAddress, encoded) = deserialize_vector(encoded, 32); //should be 32 bytes

        let (tokenChain, encoded) = deserialize_u64(encoded); //should be u16

        let (symbol, encoded) = deserialize_u64(encoded); //should be u32

        let (name, encoded) = deserialize_vector(encoded, 32); //should be u32

        let (tokenID, encoded) = deserialize_u128(encoded); //should be u256

        let n = vector::length(&encoded);

        let (uri, encoded) = deserialize_vector(encoded, n - 34); //uri has variable length?

        let (toChain, encoded) = deserialize_u64(encoded); //should be u16

        let (to, encoded) = deserialize_vector(encoded, 32); //should be 32 bytes

        Transfer {
            tokenAddress: tokenAddress, 
            tokenChain: tokenChain,  
            symbol: symbol, 
            name: name,  
            tokenID: tokenID, 
            uri: string::utf8(uri),  
            to: to, 
            toChain: toChain,  
        }
    }
}
