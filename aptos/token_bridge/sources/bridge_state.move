module token_bridge::bridge_state {

    use std::table::{Self, Table};
    use aptos_framework::type_info::{TypeInfo, type_of, account_address};
    use aptos_framework::account::{SignerCapability, create_signer_with_capability};
    use aptos_framework::coin::{Coin, MintCapability, BurnCapability, FreezeCapability, mint, initialize, transfer};
    use aptos_framework::aptos_coin::{AptosCoin};
    use aptos_framework::string::{utf8};
    use aptos_framework::bcs::{to_bytes};
    use aptos_framework::vector::{Self};

    use wormhole::u256::{Self, U256};
    use wormhole::u16::{Self, U16};
    use wormhole::emitter::{EmitterCapability};
    use wormhole::state::{get_chain_id, get_governance_contract};
    use wormhole::wormhole;
    use wormhole::set::{Self, Set};
    use wormhole::vaa::{Self, parse_and_verify};

    use token_bridge::bridge_structs::{Self, AssetMeta, TransferResult, create_transfer_result};
    use token_bridge::utils::{hash_type_info};
    //use token_bridge::vaa::{parse_verify_and_replay_protect};

    friend token_bridge::token_bridge;
    friend token_bridge::bridge_implementation;

    const E_IS_NOT_WRAPPED_ASSET: u64 = 0;
    const E_COIN_CAP_DOES_NOT_EXIST: u64 = 1;

    struct Asset has key, store {
        chain_id: U16,
        asset_address: vector<u8>,
    }

    struct CoinCapabilities<phantom CoinType> has key, store {
        mint_cap: MintCapability<CoinType>,
        freeze_cap: FreezeCapability<CoinType>,
        burn_cap: BurnCapability<CoinType>,
    }

    // the native chain and address of a wrapped token
    struct NativeInfo has store, copy, drop {
        token_address: vector<u8>,
        token_chain: U16,
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
        native_info_to_wrapped_assets: Table<NativeInfo, vector<u8>>,

        wrapped_assets_to_native_info: Table<vector<u8>, NativeInfo>,

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

    public entry fun wrapped_asset(native_info: NativeInfo): vector<u8> acquires State {
        let native_info_to_wrapped_assets = &borrow_global<State>(@token_bridge).native_info_to_wrapped_assets;
        *table::borrow(native_info_to_wrapped_assets, native_info)
    }

    public entry fun native_info(token_address: vector<u8>): NativeInfo acquires State {
        let wrapped_assets_to_native_info = &borrow_global<State>(@token_bridge).wrapped_assets_to_native_info;
        *table::borrow(wrapped_assets_to_native_info, token_address)
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
        assert!(is_wrapped_asset(token), E_IS_NOT_WRAPPED_ASSET);
        assert!(exists<CoinCapabilities<CoinType>>(@token_bridge), E_COIN_CAP_DOES_NOT_EXIST);
        let caps = borrow_global<CoinCapabilities<CoinType>>(@token_bridge);
        let mint_cap = &caps.mint_cap;
        let coins = mint<CoinType>(amount, mint_cap);
        coins
    }

    // this function is called in tandem with bridge_implementation::create_wrapped_coin_type
    public entry fun create_wrapped_coin<CoinType>(vaa: vector<u8>) acquires State{
        //TODO: parse and verify and replay protect
        //let vaa = parse_verify_and_replay_protect(vaa);
        let vaa = parse_and_verify(vaa);
        let _asset_meta: AssetMeta = bridge_structs::parse_asset_meta(vaa::get_payload(&vaa));

        // fetch signer_cap and create signer for CoinType
        let token_address = account_address(&type_of<CoinType>());
        let wrapped_coin_signer_caps = &borrow_global<State>(@token_bridge).wrapped_asset_signer_capabilities;
        let coin_signer_cap = table::borrow(wrapped_coin_signer_caps, to_bytes(&token_address));
        let coin_signer = create_signer_with_capability(coin_signer_cap);

        // initialize new coin using CoinType
        let name = bridge_structs::get_name(&_asset_meta);
        let symbol = bridge_structs::get_symbol(&_asset_meta);
        let decimals = bridge_structs::get_decimals(&_asset_meta);
        let monitor_supply = false;
        let (burn_cap, freeze_cap, mint_cap) = initialize<CoinType>(&coin_signer, utf8(name), utf8(symbol), decimals, monitor_supply);

        // update the following two mappings in State
        // 1. (native chain, native address) => wrapped address
        // 2. wrapped address => (native chain, native address)
        let token_address = bridge_structs::get_token_address(& _asset_meta);
        let token_chain = bridge_structs::get_token_chain(& _asset_meta);
        let native_info = NativeInfo {token_address: token_address, token_chain: token_chain};
        set_native_info(token_address, &native_info);
        set_wrapped_asset(&native_info, token_address);

        // store coin capabilities
        let _token_bridge_signer = token_bridge_signer();
        let coin_caps = CoinCapabilities<CoinType> { mint_cap: mint_cap, freeze_cap: freeze_cap, burn_cap: burn_cap};
        move_to(&_token_bridge_signer, coin_caps);

        vaa::destroy(vaa);
    }

    // transfer a native or wraped token from sender to token_bridge
    public fun transfer_tokens<CoinType>(_sender: &signer, _amount: u128, _arbiter_fee: u128): TransferResult acquires State {
        // TODO: assertions and checks,
        // check if CoinType is registered with @token_bridge
        let token_address = hash_type_info<CoinType>();

        // transfer tokens from sender to token_bridge
        transfer<CoinType>(_sender, @token_bridge, (_amount as u64));

        // return TransferResult encapsulating details of token transferred
        let native_chain = u16::from_u64(0);
        let native_address = vector::empty<u8>();

        if (is_wrapped_asset(token_address)) {
            let _native_info = native_info(token_address);
            native_chain = _native_info.token_chain;
            native_address = _native_info.token_address;
        } else if (is_registered_native_asset(token_address)) {
            native_chain = get_chain_id();
            native_address = token_address;
        } else {
            // TODO - if unregistered asset, then register the asset in the native_asset map?
        };

        // TODO - normalize amount by using helpers from utils.move
        let normalized_amount = u256::from_u128(_amount);
        // TODO - normalize arbiter fee
        let normalized_arbiter_fee = u256::from_u128(_arbiter_fee);
        let wormhole_fee = u256::from_u64(0);

        let transfer_result: TransferResult = create_transfer_result(
            native_chain,
            native_address,
            normalized_amount,
            normalized_arbiter_fee,
            wormhole_fee
        );
        transfer_result
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

    public entry fun set_wrapped_asset(native_info: &NativeInfo, wrapper: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        let native_info_to_wrapped_assets = &mut state.native_info_to_wrapped_assets;
        table::upsert(native_info_to_wrapped_assets, *native_info, wrapper);
        let is_wrapped_asset = &mut state.is_wrapped_asset;
        table::upsert(is_wrapped_asset, wrapper, true);
    }

    public entry fun set_native_info(wrapper: vector<u8>, native_info: &NativeInfo) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        let wrapped_assets_to_native_info = &mut state.wrapped_assets_to_native_info;
        table::upsert(wrapped_assets_to_native_info, wrapper, *native_info);
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
            native_info_to_wrapped_assets: table::new<NativeInfo, vector<u8>>(),
            wrapped_assets_to_native_info: table::new<vector<u8>, NativeInfo>(),
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
