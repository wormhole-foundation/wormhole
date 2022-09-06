module token_bridge::bridge_state {

    use std::table::{Self, Table};
    use aptos_framework::type_info::{TypeInfo};
    use aptos_framework::account::{SignerCapability, create_signer_with_capability};
    use aptos_framework::coin::{Coin, MintCapability, BurnCapability, FreezeCapability, mint};
    use aptos_framework::aptos_coin::{AptosCoin};

    use wormhole::u256::{U256};
    use wormhole::u16::{U16};
    use wormhole::emitter::{EmitterCapability};
    use wormhole::state::{get_chain_id, get_governance_contract};
    use wormhole::wormhole;
    use wormhole::set::{Self, Set};

    friend token_bridge::token_bridge;
    friend token_bridge::bridge_implementation;

    struct Asset has key, store {
        chain_id: U16,
        asset_address: vector<u8>,
    }

    struct CoinCapabilities<phantom CoinType> has key {
        mint_cap: MintCapability<CoinType>,
        freeze_cap: FreezeCapability<CoinType>,
        burn_cap: BurnCapability<CoinType>,
    }

    struct State has key, store {
        governance_chain_id: U16,
        governance_contract: vector<u8>,

        // Set of consumed governance actions
        consumed_vaas: Set<vector<u8>>,

        // TODO: does this nested mapping setup buy us anything over
        // (chainId, nativeAddress) => wrappedAddress?
        // that would be more efficient since it's a single hash and a single lookup
        //
        // Mapping of wrapped assets (chain_id => native_address => wrapped_address)
        //
        // A Wormhole wrapped coin on Aptos is identified by a single address, because
        // we assume it was initialized from the CoinType "deployer::coin::T", where the module and struct
        // names are fixed.
        //
        wrapped_assets: Table<U16, Table<vector<u8>, vector<u8>>>,

        // https://github.com/aptos-labs/aptos-core/blob/devnet/aptos-move/framework/aptos-stdlib/sources/type_info.move
        // Mapping of native asset TypeInfo sha3_256 hash (32 bytes) => TypeInfo
        // We have to identify native assets using a 32 byte identifier, because that is what fits in
        // TokenTransferWithPayload struct, among others.
        native_assets: Table<vector<u8>, TypeInfo>,

        // Mapping to safely identify wrapped assets from a 32 byte hash of its TypeInfo
        is_wrapped_asset: Table<vector<u8>, bool>,

        wrapped_asset_signer_capabilities: Table<vector<u8>, SignerCapability>,

        signer_cap: SignerCapability,

        emitter_cap: EmitterCapability,

        // Mapping to safely identify native assets from a 32 byte hash of its TypeInfo
        // all CoinTypes that aren't Wormhole wrapped assets are presumed native assets...
        is_registered_native_asset: Table<vector<u8>, bool>,

        // Mapping of native assets to amount outstanding on other chains
        outstanding_bridged: Table<vector<u8>, U256>, // should be address => u256

        // Mapping of bridge contracts on other chains
        bridge_implementations: Table<U16, vector<u8>>, //should be u16=>vector<u8>
    }

    // getters

    // TODO: these shouldn't be entry functions...

    public entry fun vaa_is_consumed(hash: vector<u8>): bool acquires State {
        let state = borrow_global<State>(@token_bridge);
        set::contains(&state.consumed_vaas, hash)
    }

    public entry fun governance_chain_id(): U16 acquires State { //should return u16
        let state = borrow_global<State>(@token_bridge);
        return state.governance_chain_id
    }

    public entry fun governance_contract(): vector<u8> acquires State { //should return u16
        let state = borrow_global<State>(@token_bridge);
        return state.governance_contract
    }

    public entry fun wrapped_asset(token_chain_id: U16, token_address: vector<u8>): vector<u8> acquires State {
        let state = borrow_global<State>(@token_bridge);
        let inner = table::borrow(&state.wrapped_assets, token_chain_id);
        *table::borrow(inner, token_address)
    }

    public entry fun native_asset(token_address: vector<u8>): TypeInfo acquires State {
        let native_assets = &borrow_global<State>(@token_bridge).native_assets;
        *table::borrow(native_assets, token_address)
    }

    public entry fun bridge_contracts(chain_id: U16): vector<u8> acquires State {
        let state = borrow_global<State>(@token_bridge);
        *table::borrow(&state.bridge_implementations, chain_id)
    }

    public entry fun outstanding_bridged(token: vector<u8>): U256 acquires State {
        let state = borrow_global<State>(@token_bridge);
        *table::borrow(&state.outstanding_bridged, token)
    }

    // given the hash of the TypeInfo of a Coin, this tells us if it is registered with Token Bridge
    public fun is_registered_native_asset(token: vector<u8>): bool acquires State {
        let state = borrow_global<State>(@token_bridge);
         *table::borrow(&state.is_registered_native_asset, token)
    }

    // the input arg is the hash of the TypeInfo of the wrapped asset
    public entry fun is_wrapped_asset(token: vector<u8>): bool acquires State {
        let state = borrow_global<State>(@token_bridge);
         *table::borrow(&state.is_wrapped_asset, token)
    }

    fun mint_wrapped<CoinType>(amount:u64, token: vector<u8>): Coin<CoinType> acquires CoinCapabilities, State{
        assert!(is_wrapped_asset(token), 0);
        assert!(exists<CoinCapabilities<CoinType>>(@token_bridge), 0);
        let caps = borrow_global<CoinCapabilities<CoinType>>(@token_bridge);
        let mint_cap = &caps.mint_cap;
        let coins = mint<CoinType>(amount, mint_cap);
        coins
    }

    public(friend) fun publish_message(
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<AptosCoin>,
    ) acquires State {
        let emitter_cap = &mut borrow_global_mut<State>(@token_bridge).emitter_cap;

        wormhole::publish_message(
            emitter_cap,
            nonce,
            payload,
            message_fee
        )
    }

    public(friend) fun token_bridge_signer(): signer acquires State {
        create_signer_with_capability(&borrow_global<State>(@token_bridge).signer_cap)
    }

    // setters

    public entry fun set_vaa_consumed(hash: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        set::add(&mut state.consumed_vaas, hash);
    }

    public entry fun set_governance_chain_id(governance_chain_id: U16) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        state.governance_chain_id = governance_chain_id;
    }

    public entry fun set_governance_contract(governance_contract: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        state.governance_contract = governance_contract;
    }

    public entry fun set_bridge_implementation(chain_id: U16, bridge_contract: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        table::upsert(&mut state.bridge_implementations, chain_id, bridge_contract);
    }

    public entry fun set_wrapped_asset(token_chain_id: U16, token_address: vector<u8>, wrapper: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        let inner_map = table::borrow_mut(&mut state.wrapped_assets, token_chain_id);
        table::upsert(inner_map, token_address, wrapper);
        let is_wrapped_asset = &mut state.is_wrapped_asset;
        table::upsert(is_wrapped_asset, wrapper, true);
    }

    public entry fun set_native_asset(token_address: vector<u8>, type_info: TypeInfo) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        let native_assets = &mut state.native_assets;
        if (table::contains(native_assets, token_address)){
            //TODO: throw error, because we should only be able to set native asset type info once?
            table::remove(native_assets, token_address);
        };
        table::add(native_assets, token_address, type_info);
        let is_registered_native_asset = &mut state.is_registered_native_asset;
        table::upsert(is_registered_native_asset, token_address, true);
    }

    public entry fun set_outstanding_bridged(token: vector<u8>, outstanding: U256) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        let outstanding_bridged = &mut state.outstanding_bridged;
        table::upsert(outstanding_bridged, token, outstanding);
    }

    public fun set_wrapped_asset_signer_capability(token: vector<u8>, signer_cap: SignerCapability) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        table::upsert(&mut state.wrapped_asset_signer_capabilities, token, signer_cap);
    }

    public(friend) fun init_token_bridge_state(
        token_bridge: &signer,
        signer_cap: SignerCapability,
        emitter_cap: EmitterCapability
    ) {
        move_to(token_bridge, State{
            governance_chain_id: get_chain_id(),
            governance_contract: get_governance_contract(),
            consumed_vaas: set::new<vector<u8>>(),
            wrapped_assets: table::new<U16, Table<vector<u8>, vector<u8>>>(),
            native_assets: table::new<vector<u8>, TypeInfo>(),
            is_wrapped_asset: table::new<vector<u8>, bool>(),
            wrapped_asset_signer_capabilities: table::new<vector<u8>, SignerCapability>(),
            signer_cap: signer_cap,
            emitter_cap: emitter_cap,
            is_registered_native_asset: table::new<vector<u8>, bool>(),
            outstanding_bridged: table::new<vector<u8>, U256>(),
            bridge_implementations: table::new<U16, vector<u8>>(),
            }
        );
    }
}
