module token_bridge::state {
    use std::table::{Self, Table};
    use std::option::{Self, Option};
    use aptos_framework::type_info::{Self, TypeInfo, type_of};
    use aptos_framework::account::{Self, SignerCapability};
    use aptos_framework::aptos_coin::AptosCoin;
    use aptos_framework::coin::Coin;

    use wormhole::u16::U16;
    use wormhole::emitter::EmitterCapability;
    use wormhole::state;
    use wormhole::wormhole;
    use wormhole::set::{Self, Set};
    use wormhole::external_address::ExternalAddress;

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

    #[test_only]
    friend token_bridge::wrapped_test;
    #[test_only]
    friend token_bridge::vaa_test;

    const E_ORIGIN_CHAIN_MISMATCH: u64 = 0;
    const E_ORIGIN_ADDRESS_MISMATCH: u64 = 1;
    const E_WRAPPING_NATIVE_COIN: u64 = 2;
    const E_WRAPPED_ASSET_NOT_INITIALIZED: u64 = 3;

    /// The origin chain and address of a token.  In case of native tokens
    /// (where the chain is aptos), the token_address is the hash of the token
    /// info (see token_hash.move for more details)
    struct OriginInfo has key, store, copy, drop {
        token_chain: U16,
        token_address: ExternalAddress,
    }

    public fun get_origin_info_token_address(info: &OriginInfo): ExternalAddress {
        info.token_address
    }

    public fun get_origin_info_token_chain(info: &OriginInfo): U16 {
        info.token_chain
    }

    public(friend) fun create_origin_info(
        token_chain: U16,
        token_address: ExternalAddress,
    ): OriginInfo {
        OriginInfo { token_address: token_address, token_chain: token_chain }
    }

    struct WrappedInfo has store {
        type_info: Option<TypeInfo>,
        signer_cap: SignerCapability
    }

    struct State has key, store {
        /// Set of consumed VAA hashes
        consumed_vaas: Set<vector<u8>>,

        /// Mapping of wrapped assets ((chain_id, origin_address) => wrapped_asset info)
        wrapped_infos: Table<OriginInfo, WrappedInfo>,

        /// Reverse mapping of hash(TypeInfo) for native tokens, so their
        /// information can be looked up externally by knowing their hash (which
        /// is the 32 byte "address" that goes into the VAA).
        native_infos: Table<TokenHash, TypeInfo>,

        signer_cap: SignerCapability,

        emitter_cap: EmitterCapability,

        // Mapping of bridge contracts on other chains
        registered_emitters: Table<U16, ExternalAddress>,
    }

    // getters

    public fun vaa_is_consumed(hash: vector<u8>): bool acquires State {
        let state = borrow_global<State>(@token_bridge);
        set::contains(&state.consumed_vaas, hash)
    }

    public fun wrapped_asset_info(native_info: OriginInfo): TypeInfo acquires State {
        let wrapped_infos = &borrow_global<State>(@token_bridge).wrapped_infos;
        let type_info = table::borrow(wrapped_infos, native_info).type_info;
        assert!(option::is_some(&type_info), E_WRAPPED_ASSET_NOT_INITIALIZED);
        option::extract(&mut type_info)
    }

    public fun native_asset_info(token_address: TokenHash): TypeInfo acquires State {
        let native_infos = &borrow_global<State>(@token_bridge).native_infos;
        *table::borrow(native_infos, token_address)
    }

    /// Returns the origin information for a CoinType
    public fun origin_info<CoinType>(): OriginInfo acquires OriginInfo {
        if (is_wrapped_asset<CoinType>()) {
            *borrow_global<OriginInfo>(type_info::account_address(&type_of<CoinType>()))
        } else {
            let token_chain = state::get_chain_id();
            let token_address = token_hash::get_external_address(&token_hash::derive<CoinType>());
            OriginInfo { token_chain, token_address }
        }
    }

    public fun get_registered_emitter(chain_id: U16): Option<ExternalAddress> acquires State {
        let state = borrow_global<State>(@token_bridge);
        if (table::contains(&state.registered_emitters, chain_id)) {
            option::some(*table::borrow(&state.registered_emitters, chain_id))
        } else {
            option::none()
        }

    }

    // given the hash of the TypeInfo of a Coin, this tells us if it is registered with Token Bridge
    public fun is_registered_native_asset<CoinType>(): bool acquires State {
        let token = token_hash::derive<CoinType>();
        let native_infos = &borrow_global<State>(@token_bridge).native_infos;
        !is_wrapped_asset<CoinType>() && table::contains(native_infos, token)
    }

    public fun is_wrapped_asset<CoinType>(): bool {
        exists<OriginInfo>(type_info::account_address(&type_of<CoinType>()))
    }

    public(friend) fun setup_wrapped<CoinType>(
        origin_info: OriginInfo
    ) acquires State {
        assert!(origin_info.token_chain != state::get_chain_id(), E_WRAPPING_NATIVE_COIN);
        let wrapped_infos = &mut borrow_global_mut<State>(@token_bridge).wrapped_infos;
        let wrapped_info = table::borrow_mut(wrapped_infos, origin_info);

        let coin_signer = account::create_signer_with_capability(&wrapped_info.signer_cap);
        move_to(&coin_signer, origin_info);

        wrapped_info.type_info = option::some(type_of<CoinType>());

    }

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
        account::create_signer_with_capability(&borrow_global<State>(@token_bridge).signer_cap)
    }

    // setters

    public(friend) fun set_vaa_consumed(hash: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        set::add(&mut state.consumed_vaas, hash);
    }

    public(friend) fun set_registered_emitter(chain_id: U16, bridge_contract: ExternalAddress) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        table::upsert(&mut state.registered_emitters, chain_id, bridge_contract);
    }

    // 32-byte native asset address => type info
    public(friend) fun set_native_asset_type_info<CoinType>() acquires State {
        let token_address = token_hash::derive<CoinType>();
        let type_info = type_of<CoinType>();

        let state = borrow_global_mut<State>(@token_bridge);
        let native_infos = &mut state.native_infos;
        if (table::contains(native_infos, token_address)) {
            //TODO: throw error, because we should only be able to set native asset type info once?
            table::remove(native_infos, token_address);
        };
        table::add(native_infos, token_address, type_info);
    }

    public(friend) fun set_wrapped_asset_signer_capability(token: OriginInfo, signer_cap: SignerCapability) acquires State {
        let state = borrow_global_mut<State>(@token_bridge);
        let wrapped_info = WrappedInfo {
            type_info: option::none(),
            signer_cap
        };
        table::add(&mut state.wrapped_infos, token, wrapped_info);
    }

    public(friend) fun get_wrapped_asset_signer(origin_info: OriginInfo): signer acquires State {
        let wrapped_coin_signer_caps
            = &borrow_global<State>(@token_bridge).wrapped_infos;
        let wrapped_info = table::borrow(wrapped_coin_signer_caps, origin_info);
        account::create_signer_with_capability(&wrapped_info.signer_cap)
    }

    public(friend) fun init_token_bridge_state(
        signer_cap: SignerCapability,
        emitter_cap: EmitterCapability
    ) {
        let token_bridge = account::create_signer_with_capability(&signer_cap);
        move_to(&token_bridge, State {
            consumed_vaas: set::new<vector<u8>>(),
            wrapped_infos: table::new(),
            native_infos: table::new(),
            signer_cap: signer_cap,
            emitter_cap: emitter_cap,
            registered_emitters: table::new(),
            }
        );
    }
}
