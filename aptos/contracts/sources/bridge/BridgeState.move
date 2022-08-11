module Wormhole::BridgeState {
    
    use 0x1::table::{Self, Table};
    use 0x1::type_info::{Self, TypeInfo};

    struct Provider has key, store {
        chainId: u64, //should be u16
        governanceChainId: u64, //should be u16
        // Required number of block confirmations to assume finality
        finality: u8,
        governanceContract: vector<u8>, //what is this used for?
        // weth: vector<u8>, - not needed, because of AptosCoin?
    }

    struct Asset has key, store {
        chainId: u64, //should be u16
        assetAddress: vector<u8>,
    }

    struct State has key, store {
        wormhole: address,
        //  address tokenImplementation, - not needed, because there is canonical coin module?

        provider: Provider,

        // Mapping of consumed governance actions
        consumedGovernanceActions: Table<vector<u8>, bool>,

        // Mapping of consumed token transfers
        completedTransfers: Table<vector<u8>, bool>, 

        // Mapping of initialized implementations
        initializedImplementations: Table<address, bool>,

        // Mapping of wrapped assets (chainID => nativeAddress => wrappedAddress)
        // https://github.com/aptos-labs/aptos-core/blob/devnet/aptos-move/framework/aptos-stdlib/sources/type_info.move
        wrappedAssets: Table<u64, Table<vector<u8>, TypeInfo>>, //a Resource/Asset is fully described by TypeInfo

        // Mapping to safely identify wrapped assets
        isWrappedAsset: Table<TypeInfo, bool>,

        // Mapping of native assets to amount outstanding on other chains
        outstandingBridged: Table<TypeInfo, u128>, // should be address => u256

        // Mapping of bridge contracts on other chains
        bridgeImplementations: Table<u64, vector<u8>>, //should be u16=>vector<u8>
    }

    //getters

    public entry fun governanceActionIsConsumed(hash: vector<u8>): bool acquires State{
        let state = borrow_global<State>(@Wormhole);
        return *table::borrow(&state.consumedGovernanceActions, hash)
    }

    // TODO: isInitialized?

    public entry fun isTransferCompleted(hash: vector<u8>): bool acquires State{
        let state = borrow_global<State>(@Wormhole);
        return *table::borrow(&state.completedTransfers, hash)
    }

    public entry fun wormhole(): address acquires State{
        let state = borrow_global<State>(@Wormhole);
        return state.wormhole
    }

    public entry fun chainId(): u64 acquires State{ //should return u16
        let state = borrow_global<State>(@Wormhole);
        return state.provider.chainId
    }

    public entry fun governanceChainId(): u64 acquires State{ //should return u16
        let state = borrow_global<State>(@Wormhole);
        return state.provider.governanceChainId
    }

    public entry fun governanceContract(): vector<u8> acquires State{ //should return u16
        let state = borrow_global<State>(@Wormhole);
        return state.provider.governanceContract
    }

    public entry fun wrappedAsset(tokenChainId: u64, tokenAddress: vector<u8>): TypeInfo acquires State{
        let state = borrow_global<State>(@Wormhole);
        let inner = table::borrow(&state.wrappedAssets, tokenChainId);
        *table::borrow(inner, tokenAddress)
    }

    public entry fun bridgeContracts(chainId: u64): vector<u8> acquires State{
        let state = borrow_global<State>(@Wormhole);
        *table::borrow(&state.bridgeImplementations, chainId)
    }

    // function APT() public view returns (IWETH){
    //     return IWETH(_state.provider.WETH);
    // }

    public entry fun outstandingBridged(token: TypeInfo): u128 acquires State{
        let state = borrow_global<State>(@Wormhole);
        *table::borrow(&state.outstandingBridged, token)
    }

    public entry fun isWrappedAsset(token: TypeInfo): bool acquires State {
        let state = borrow_global<State>(@Wormhole);
         *table::borrow(&state.isWrappedAsset, token)
    }

    public entry fun finality(): u8 acquires State {
        let state = borrow_global<State>(@Wormhole);
        state.provider.finality
    }
    
    // setters

    // function setInitialized(address implementatiom) internal {
    //     _state.initializedImplementations[implementatiom] = true;
    // }

    public entry fun setGovernanceActionConsumed(hash: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@Wormhole);
        if (table::contains(&state.consumedGovernanceActions, hash)){
            table::remove(&mut state.consumedGovernanceActions, hash);
        };
        table::add(&mut state.consumedGovernanceActions, hash, true);
    }

    public entry fun setTransferCompleted(hash: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@Wormhole);
        if (table::contains(&state.completedTransfers, hash)){
            table::remove(&mut state.completedTransfers, hash);
        };
        table::add(&mut state.completedTransfers, hash, true);
    }

    public entry fun setChainId(chainId: u64) acquires State {  
        let state = borrow_global_mut<State>(@Wormhole);
        let provider = &mut state.provider;
        provider.chainId = chainId;
    }

    public entry fun setGovernanceChainId(governanceChainId: u64) acquires State { 
        let state = borrow_global_mut<State>(@Wormhole);
        let provider = &mut state.provider;
        provider.governanceChainId = governanceChainId;
    }

    public entry fun setGovernanceContract(governanceContract: vector<u8>) acquires State { 
        let state = borrow_global_mut<State>(@Wormhole);
        let provider = &mut state.provider;
        provider.governanceContract=governanceContract;
    }

    public entry fun setBridgeImplementation(chainId: u64, bridgeContract: vector<u8>) acquires State { 
        let state = borrow_global_mut<State>(@Wormhole);
        if (table::contains(&state.bridgeImplementations, chainId)){
            table::remove(&mut state.bridgeImplementations, chainId);
        };
        table::add(&mut state.bridgeImplementations, chainId, bridgeContract);
    }

    public entry fun setWormhole(wh: address) acquires State{
        let state = borrow_global_mut<State>(@Wormhole);
        state.wormhole = wh;
    }

    public entry fun setWrappedAsset(tokenChainId: u64, tokenAddress: vector<u8>, wrapper: TypeInfo) acquires State {
        let state = borrow_global_mut<State>(@Wormhole);
        let inner_map = table::borrow_mut(&mut state.wrappedAssets, tokenChainId);
        if (table::contains(inner_map, tokenAddress)){
            table::remove(inner_map, tokenAddress);
        };
        table::add(inner_map, tokenAddress, wrapper);
        let isWrappedAsset = &mut state.isWrappedAsset;
        if (table::contains(isWrappedAsset, wrapper)){
            table::remove(isWrappedAsset, wrapper);
        };
        table::add(isWrappedAsset, wrapper, true);
    }

    public entry fun setOutstandingBridged(token: TypeInfo, outstanding: u128) acquires State { 
        let state = borrow_global_mut<State>(@Wormhole);
        let outstandingBridged = &mut state.outstandingBridged;
        if (table::contains(outstandingBridged, token)){
            table::remove(outstandingBridged, token);
        };
        table::add(outstandingBridged, token, outstanding);
    }

    public entry fun setFinality(finality: u8) acquires State{ 
        let state = borrow_global_mut<State>(@Wormhole);
        state.provider.finality = finality;
    }

}





