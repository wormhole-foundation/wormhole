// TODO(csongor): implement SPL cache (solana tokens are minted into a single
// collection, but we want to preserve the original mint information in a
// separate cache, like the eth contract does)
module nft_bridge::state {
    use std::table::{Self, Table};
    use std::option::{Self, Option};
    use aptos_framework::account::{Self, SignerCapability};
    use aptos_framework::aptos_coin::AptosCoin;
    use aptos_framework::coin::Coin;

    use aptos_token::token::{Self, TokenId};

    use wormhole::u16::U16;
    use wormhole::emitter::EmitterCapability;
    use wormhole::state;
    use wormhole::wormhole;
    use wormhole::set::{Self, Set};
    use wormhole::external_address::ExternalAddress;

    use nft_bridge::token_hash::{Self, TokenHash};

    friend nft_bridge::contract_upgrade;
    friend nft_bridge::register_chain;
    friend nft_bridge::nft_bridge;
    friend nft_bridge::vaa;
    friend nft_bridge::wrapped;
    friend nft_bridge::complete_transfer;
    friend nft_bridge::transfer_nft;

    #[test_only]
    friend nft_bridge::wrapped_test;
    #[test_only]
    friend nft_bridge::vaa_test;
    #[test_only]
    friend nft_bridge::transfer_nft_test;

    const E_ORIGIN_CHAIN_MISMATCH: u64 = 0;
    const E_ORIGIN_ADDRESS_MISMATCH: u64 = 1;
    const W_WRAPPING_NATIVE_NFT: u64 = 2;
    const E_WRAPPED_ASSET_NOT_INITIALIZED: u64 = 3;

    /// The origin chain and address of a token.
    struct OriginInfo has key, store, copy, drop {
        /// Chain from which the token originates
        token_chain: U16,
        /// Address of the collection (unique per chain)
        /// For native tokens, it's derived as the hash of (creator || collection)
        token_address: ExternalAddress,
        /// Token identifier (unique within collection)
        /// For native tokens, it's derived as the hash of
        /// (creator || collection || name || property version)
        /// which means it's also globally unique on Aptos
        token_id: ExternalAddress
    }

    public fun get_origin_info_token_address(info: &OriginInfo): ExternalAddress {
        info.token_address
    }

    public fun get_origin_info_token_id(info: &OriginInfo): ExternalAddress {
        info.token_id
    }

    public fun get_origin_info_token_chain(info: &OriginInfo): U16 {
        info.token_chain
    }

    public(friend) fun create_origin_info(
        token_chain: U16,
        token_address: ExternalAddress,
        token_id: ExternalAddress,
    ): OriginInfo {
        OriginInfo { token_chain, token_address, token_id }
    }

    struct WrappedInfo has store {
        signer_cap: SignerCapability
    }

    struct State has key, store {
        /// Set of consumed VAA hashes
        consumed_vaas: Set<vector<u8>>,

        /// Mapping of wrapped assets ((chain_id, origin_address) => wrapped_asset info)
        wrapped_infos: Table<OriginInfo, WrappedInfo>,

        /// Reverse mapping of hash(TokenId) for native tokens, so their
        /// information can be looked up externally by knowing their hash (which
        /// is the 32 byte "address" that goes into the VAA).
        native_infos: Table<TokenHash, TokenId>,

        signer_cap: SignerCapability,

        emitter_cap: EmitterCapability,

        /// Mapping of bridge contracts on other chains
        registered_emitters: Table<U16, ExternalAddress>,
    }

    // getters

    public fun vaa_is_consumed(hash: vector<u8>): bool acquires State {
        let state = borrow_global<State>(@nft_bridge);
        set::contains(&state.consumed_vaas, hash)
    }

    /// Returns the origin information for a token
    public fun get_origin_info(token_id: &TokenId): OriginInfo acquires OriginInfo {
        if (is_wrapped_asset(token_id)) {
            let (creator, _, _, _) = token::get_token_id_fields(token_id);
            *borrow_global<OriginInfo>(creator)
        } else {
            let token_chain = state::get_chain_id();
            let (collection_hash, token_hash) = token_hash::derive(token_id);
            let token_address = token_hash::get_collection_external_address(&collection_hash);
            let token_id = token_hash::get_token_external_address(&token_hash);
            OriginInfo { token_chain, token_address, token_id }
        }
    }

    public fun get_registered_emitter(chain_id: U16): Option<ExternalAddress> acquires State {
        let state = borrow_global<State>(@nft_bridge);
        if (table::contains(&state.registered_emitters, chain_id)) {
            option::some(*table::borrow(&state.registered_emitters, chain_id))
        } else {
            option::none()
        }

    }

    public fun is_wrapped_asset(token_id: &TokenId): bool {
        let (creator, _, _, _) = token::get_token_id_fields(token_id);
        exists<OriginInfo>(creator)
    }

    public(friend) fun setup_wrapped(
        origin_info: OriginInfo
    ) acquires State {
        assert!(origin_info.token_chain != state::get_chain_id(), W_WRAPPING_NATIVE_NFT);
        let wrapped_infos = &mut borrow_global_mut<State>(@nft_bridge).wrapped_infos;
        let wrapped_info = table::borrow_mut(wrapped_infos, origin_info);

        let coin_signer = account::create_signer_with_capability(&wrapped_info.signer_cap);
        move_to(&coin_signer, origin_info);
    }

    public(friend) fun publish_message(
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<AptosCoin>,
    ): u64 acquires State {
        let emitter_cap = &mut borrow_global_mut<State>(@nft_bridge).emitter_cap;
        wormhole::publish_message(
            emitter_cap,
            nonce,
            payload,
            message_fee
        )
    }

    public(friend) fun nft_bridge_signer(): signer acquires State {
        account::create_signer_with_capability(&borrow_global<State>(@nft_bridge).signer_cap)
    }

    // setters

    public(friend) fun set_vaa_consumed(hash: vector<u8>) acquires State {
        let state = borrow_global_mut<State>(@nft_bridge);
        set::add(&mut state.consumed_vaas, hash);
    }

    public(friend) fun set_registered_emitter(chain_id: U16, bridge_contract: ExternalAddress) acquires State {
        let state = borrow_global_mut<State>(@nft_bridge);
        table::upsert(&mut state.registered_emitters, chain_id, bridge_contract);
    }

    // 32-byte native asset address => token info
    public(friend) fun set_native_asset_info(token_id: TokenId) acquires State {
        let (_, token_hash) = token_hash::derive(&token_id);

        let state = borrow_global_mut<State>(@nft_bridge);
        let native_infos = &mut state.native_infos;
        if (!table::contains(native_infos, token_hash)) {
            table::add(native_infos, token_hash, token_id);
        }
    }

    public(friend) fun get_native_asset_info(token_hash: TokenHash): TokenId acquires State {
        *table::borrow(&borrow_global<State>(@nft_bridge).native_infos, token_hash)
    }

    public(friend) fun set_wrapped_asset_signer_capability(token: OriginInfo, signer_cap: SignerCapability) acquires State {
        let state = borrow_global_mut<State>(@nft_bridge);
        let wrapped_info = WrappedInfo {
            signer_cap
        };
        table::add(&mut state.wrapped_infos, token, wrapped_info);
    }

    public(friend) fun get_wrapped_asset_signer(origin_info: OriginInfo): signer acquires State {
        let wrapped_coin_signer_caps
            = &borrow_global<State>(@nft_bridge).wrapped_infos;
        let wrapped_info = table::borrow(wrapped_coin_signer_caps, origin_info);
        account::create_signer_with_capability(&wrapped_info.signer_cap)
    }

    public(friend) fun wrapped_asset_signer_exists(origin_info: OriginInfo): bool acquires State {
        let wrapped_coin_signer_caps
            = &borrow_global<State>(@nft_bridge).wrapped_infos;
        table::contains(wrapped_coin_signer_caps, origin_info)
    }

    public(friend) fun init_nft_bridge_state(
        signer_cap: SignerCapability,
        emitter_cap: EmitterCapability
    ) {
        let nft_bridge = account::create_signer_with_capability(&signer_cap);
        move_to(&nft_bridge, State {
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
