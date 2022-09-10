module token_bridge::state {
    use std::table::{Self, Table};
    use std::option::{Self, Option};
    use aptos_framework::type_info::{Self, TypeInfo, type_of};
    use aptos_framework::account::{Self, SignerCapability, create_signer_with_capability};
    use aptos_framework::aptos_coin::AptosCoin;
    use aptos_framework::coin::Coin;

    use wormhole::u256::U256;
    use wormhole::u16::U16;
    use wormhole::emitter::EmitterCapability;
    use wormhole::state::{get_chain_id, get_governance_contract};
    use wormhole::wormhole;
    use wormhole::set::{Self, Set};

    use token_bridge::token_hash::{Self, TokenHash};

    friend token_bridge::contract_upgrade;
    friend token_bridge::register_chain;
    friend token_bridge::token_bridge;
    friend token_bridge::vaa;
    friend token_bridge::attest_token;
    friend token_bridge::wrapped;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::transfer_tokens;

    const E_ORIGIN_CHAIN_MISMATCH: u64 = 0;
    const E_ORIGIN_ADDRESS_MISMATCH: u64 = 1;
    const E_COIN_NOT_REGISTERED: u64 = 2;
    const E_FEE_EXCEEDS_AMOUNT: u64 = 3;

    struct Asset has key, store {
        chain_id: U16,
        asset_address: vector<u8>,
    }

    /// The origin chain and address of a token.  In case of native tokens
    /// (where the chain is aptos), the token_address is the hash of the token
    /// info (see token_hash.move for more details)
    struct OriginInfo has key, store, copy, drop {
        token_chain: U16,
        token_address: vector<u8>,
    }

    public(friend) fun create_origin_info(
        token_chain: U16,
        token_address: vector<u8>,
    ): OriginInfo {
        OriginInfo { token_address, token_chain }
    }

    struct State has key, store {
        governance_chain_id: U16,
        governance_contract: vector<u8>,

        // Set of consumed governance actions
        consumed_vaas: Set<vector<u8>>,

        // Mapping of wrapped assets (chain_id => origin_address => wrapped_address)
        //
        // A Wormhole wrapped coin on Aptos is identified by a single address, because
        // we assume it was initialized from the CoinType "deployer::coin::T", where the module and struct
        // names are fixed.
        //
        // TODO: maybe this should map to TypeInfos
        origin_info_to_wrapped_assets: Table<OriginInfo, TokenHash>,

        // https://github.com/aptos-labs/aptos-core/blob/devnet/aptos-move/framework/aptos-stdlib/sources/type_info.move
        // Mapping of native asset TypeInfo sha3_256 hash (32 bytes) => TypeInfo
        // We have to identify native assets using a 32 byte identifier, because that is what fits in
        // TokenTransferWithPayload struct, among others.
        assets_to_type_info: Table<TokenHash, TypeInfo>,

        // Mapping to safely identify native assets from a 32 byte hash of its TypeInfo
        // all CoinTypes that aren't Wormhole wrapped assets are presumed native assets...
        // TODO: use a Set
        is_registered_native_asset: Table<TokenHash, bool>,

        // Mapping from (wrapped) token address to Signer Capability.
        wrapped_asset_signer_capabilities: Table<OriginInfo, SignerCapability>,

        signer_cap: SignerCapability,

        emitter_cap: EmitterCapability,

        // Mapping of native assets to amount outstanding on other chains
        outstanding_bridged: Table<vector<u8>, U256>, // should be address => u256

        // Mapping of bridge contracts on other chains
        registered_emitters: Table<U16, vector<u8>>,
    }

    // getters

    public fun vaa_is_consumed(hash: vector<u8>): bool acquires State {
        let state = borrow_global<State>(@token_bridge);
        set::contains(&state.consumed_vaas, hash)
    }

    public fun governance_chain_id(): U16 acquires State {
        let state = borrow_global<State>(@token_bridge);
        return state.governance_chain_id
    }

    public fun governance_contract(): vector<u8> acquires State {
        let state = borrow_global<State>(@token_bridge);
        return state.governance_contract
    }

    public fun wrapped_asset(native_info: OriginInfo): TokenHash acquires State {
        let origin_info_to_wrapped_assets = &borrow_global<State>(@token_bridge).origin_info_to_wrapped_assets;
        *table::borrow(origin_info_to_wrapped_assets, native_info)
    }

    /// Returns the origin information for a CoinType
    public fun origin_info<CoinType>(): OriginInfo acquires OriginInfo {
        if (is_wrapped_asset<CoinType>()) {
            *borrow_global<OriginInfo>(type_info::account_address(&type_of<CoinType>()))
        } else {
            let token_chain = get_chain_id();
            let token_address = token_hash::get_bytes(&token_hash::derive<CoinType>());
            OriginInfo { token_chain, token_address }
        }
    }

    public fun asset_type_info(token_address: TokenHash): TypeInfo acquires State {
        let assets_to_type_info = &borrow_global<State>(@token_bridge).assets_to_type_info;
        *table::borrow(assets_to_type_info, token_address)
    }

    public fun get_registered_emitter(chain_id: U16): Option<vector<u8>> acquires State {
        let state = borrow_global<State>(@token_bridge);
        if (table::contains(&state.registered_emitters, chain_id)) {
            option::some(*table::borrow(&state.registered_emitters, chain_id))
        } else {
            option::none()
        }

    }

    public fun outstanding_bridged(token: vector<u8>): U256 acquires State {
        let state = borrow_global<State>(@token_bridge);
        *table::borrow(&state.outstanding_bridged, token)
    }

    // given the hash of the TypeInfo of a Coin, this tells us if it is registered with Token Bridge
    public fun is_registered_native_asset<CoinType>(): bool acquires State {
        let token = token_hash::derive<CoinType>();
        let state = borrow_global<State>(@token_bridge);
        //TODO - make is_registered_native_asset a set
        table::contains(&state.is_registered_native_asset, token)
    }

    public fun is_wrapped_asset<CoinType>(): bool {
        exists<OriginInfo>(type_info::account_address(&type_of<CoinType>()))
    }

    public(friend) fun setup_wrapped<CoinType>(
        coin_signer: &signer,
        origin_info: OriginInfo
    ) acquires State {
        // TODO: ensure that origin chain != current chain
        move_to(coin_signer, origin_info);
        set_wrapped_asset<CoinType>(&origin_info);
        set_wrapped_asset_type_info<CoinType>();
    }

    public fun get_origin_info_token_address(info: &OriginInfo): vector<u8>{
        info.token_address
    }

    public fun get_origin_info_token_chain(info: &OriginInfo): U16{
        info.token_chain
    }

    // TODO(csongor): add this check everywhere where a VAA comes in for a token
    // (complete transfer and create wrapped)
    // TODO(csongor): should this return some sort of witness proving the origin?
    public fun assert_coin_origin_info<CoinType>(origin: OriginInfo) acquires OriginInfo {
        let coin_origin = origin_info<CoinType>();
        assert!(coin_origin.token_chain == origin.token_chain, E_ORIGIN_CHAIN_MISMATCH);
        assert!(coin_origin.token_address == origin.token_address, E_ORIGIN_ADDRESS_MISMATCH);
    }

    public(friend) fun publish_message(
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<AptosCoin>,
    ): u64 acquires State {
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

    public(friend) fun set_vaa_consumed(hash: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        set::add(&mut state.consumed_vaas, hash);
    }

    public(friend) fun set_governance_chain_id(governance_chain_id: U16) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        state.governance_chain_id = governance_chain_id;
    }

    public(friend) fun set_governance_contract(governance_contract: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        state.governance_contract = governance_contract;
    }

    public(friend) fun set_registered_emitter(chain_id: U16, bridge_contract: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        table::upsert(&mut state.registered_emitters, chain_id, bridge_contract);
    }

    // OriginInfo => WrappedAsset
    fun set_wrapped_asset<CoinType>(native_info: &OriginInfo) acquires State {
        let wrapper = token_hash::derive<CoinType>();

        let state = borrow_global_mut<State>(@token_bridge);
        let origin_info_to_wrapped_assets = &mut state.origin_info_to_wrapped_assets;
        table::upsert(origin_info_to_wrapped_assets, *native_info, wrapper);
    }

    // 32-byte native asset address => type info
    // TODO: call this function register_native_asset
    public(friend) fun set_native_asset_type_info<CoinType>() acquires State {
        let token_address = token_hash::derive<CoinType>();
        let type_info = type_of<CoinType>();

        let state = borrow_global_mut<State>(@token_bridge);
        let assets_to_type_info = &mut state.assets_to_type_info;
        if (table::contains(assets_to_type_info, token_address)){
            //TODO: throw error, because we should only be able to set native asset type info once?
            table::remove(assets_to_type_info, token_address);
        };
        table::add(assets_to_type_info, token_address, type_info);
        let is_registered_native_asset = &mut state.is_registered_native_asset;
        table::upsert(is_registered_native_asset, token_address, true);
    }

    // 32-byte wrapped asset address => type info
    fun set_wrapped_asset_type_info<CoinType>() acquires State {
        let token_address = token_hash::derive<CoinType>();
        let type_info = type_of<CoinType>();

        let state = borrow_global_mut<State>(@token_bridge);
        let assets_to_type_info = &mut state.assets_to_type_info;
        table::add(assets_to_type_info, token_address, type_info);
    }

    // TODO: I don't think this is needed.
    public(friend) fun set_outstanding_bridged(token: vector<u8>, outstanding: U256) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        let outstanding_bridged = &mut state.outstanding_bridged;
        table::upsert(outstanding_bridged, token, outstanding);
    }

    public(friend) fun set_wrapped_asset_signer_capability(token: OriginInfo, signer_cap: SignerCapability) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        table::upsert(&mut state.wrapped_asset_signer_capabilities, token, signer_cap);
    }

    public(friend) fun get_wrapped_asset_signer(origin_info: OriginInfo): signer acquires State {
        let wrapped_coin_signer_caps
            = &borrow_global<State>(@token_bridge).wrapped_asset_signer_capabilities;
        let coin_signer_cap = table::borrow(wrapped_coin_signer_caps, origin_info);
        create_signer_with_capability(coin_signer_cap)
    }

    public(friend) fun init_token_bridge_state(
        signer_cap: SignerCapability,
        emitter_cap: EmitterCapability
    ) {
        let token_bridge = account::create_signer_with_capability(&signer_cap);
        move_to(&token_bridge, State{
            governance_chain_id: get_chain_id(),
            governance_contract: get_governance_contract(),
            consumed_vaas: set::new<vector<u8>>(),
            origin_info_to_wrapped_assets: table::new(),
            assets_to_type_info: table::new(),
            is_registered_native_asset: table::new(),
            wrapped_asset_signer_capabilities: table::new(),
            signer_cap: signer_cap,
            emitter_cap: emitter_cap,
            outstanding_bridged: table::new(),
            registered_emitters: table::new(),
            }
        );
    }
}
