module nft_bridge::state {
    use std::table::{Self, Table};
    use std::option::{Self, Option};
    use std::string::String;
    use aptos_framework::account::{Self, SignerCapability};
    use aptos_framework::aptos_coin::AptosCoin;
    use aptos_framework::coin::Coin;

    use aptos_token::token::{Self, TokenId};

    use wormhole::u16::{Self, U16};
    use wormhole::emitter::EmitterCapability;
    use wormhole::state;
    use wormhole::wormhole;
    use wormhole::set::{Self, Set};
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::string32::{Self, String32};

    use nft_bridge::token_hash::{Self, TokenHash};
    use nft_bridge::wrapped_token_name;

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
    #[test_only]
    friend nft_bridge::complete_transfer_test;

    const E_ORIGIN_CHAIN_MISMATCH: u64 = 0;
    const E_ORIGIN_ADDRESS_MISMATCH: u64 = 1;
    const W_WRAPPING_NATIVE_NFT: u64 = 2;
    const E_WRAPPED_ASSET_NOT_INITIALIZED: u64 = 3;

    /// The origin chain and address of a token (represents the origin of a collection)
    struct OriginInfo has key, store, copy, drop {
        /// Chain from which the token originates
        token_chain: U16,
        /// Address of the collection (unique per chain)
        /// For native tokens, it's derived as the hash of (creator || hash(collection))
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
        OriginInfo { token_chain, token_address }
    }

    struct WrappedInfo has store {
        signer_cap: SignerCapability,
        /// The token's symbol in the NFT bridge standard does not map to any of
        /// the fields in the Aptos NFT standard, so there's no natural way to
        /// store that information when creating a wrapped NFT collection.
        /// However, when transferring out these assets, we want to preserve the
        /// original symbol, so we do that here.
        symbol: String32
    }

    /// See `is_unified_solana_collection` for the purpose of this type
    /// It has the `drop` ability so old cache entries can be overridden
    struct SPLCacheEntry has store, drop {
        name: String32,
        symbol: String32
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

        /// See `is_unified_solana_collection` for the purpose
        /// of this field.
        /// Mapping of token_id => spl cache entry
        spl_cache: Table<ExternalAddress, SPLCacheEntry>,
    }

    // getters

    public fun vaa_is_consumed(hash: vector<u8>): bool acquires State {
        let state = borrow_global<State>(@nft_bridge);
        set::contains(&state.consumed_vaas, hash)
    }

    /// Returns the origin information for a token
    public fun get_origin_info(token_id: &TokenId): (OriginInfo, ExternalAddress) acquires OriginInfo {
        if (is_wrapped_asset(token_id)) {
            let (creator, _, token_name, _) = token::get_token_id_fields(token_id);
            let external_address = wormhole::external_address::from_bytes(wrapped_token_name::parse_hex(token_name));
            (*borrow_global<OriginInfo>(creator), external_address)
        } else {
            let token_chain = state::get_chain_id();
            let (collection_hash, token_hash) = token_hash::derive(token_id);
            let token_address = token_hash::get_collection_external_address(&collection_hash);
            let token_id = token_hash::get_token_external_address(&token_hash);
            (OriginInfo { token_chain, token_address }, token_id)
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

    public fun get_spl_cache(token_id: ExternalAddress): (String32, String32) acquires State {
        let state = borrow_global<State>(@nft_bridge);
        let SPLCacheEntry { name, symbol } = table::borrow(&state.spl_cache, token_id);
        (*name, *symbol)
    }

    public(friend) fun set_spl_cache(token_id: ExternalAddress, name: String32, symbol: String32) acquires State {
        let state = borrow_global_mut<State>(@nft_bridge);
        table::upsert(&mut state.spl_cache, token_id, SPLCacheEntry { name, symbol });
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

    public(friend) fun set_wrapped_asset_info(
        token: OriginInfo,
        signer_cap: SignerCapability,
        symbol: String32
    ) acquires State {
        let state = borrow_global_mut<State>(@nft_bridge);
        let wrapped_info = WrappedInfo {
            signer_cap,
            symbol
        };
        table::add(&mut state.wrapped_infos, token, wrapped_info);
    }

    public(friend) fun get_wrapped_asset_signer(origin_info: OriginInfo): signer acquires State {
        let wrapped_coin_infos
            = &borrow_global<State>(@nft_bridge).wrapped_infos;
        let wrapped_info = table::borrow(wrapped_coin_infos, origin_info);
        account::create_signer_with_capability(&wrapped_info.signer_cap)
    }

    public fun get_wrapped_asset_name_and_symbol(
        origin_info: OriginInfo,
        collection_name: String,
        token_id: ExternalAddress
    ): (String32, String32) acquires State {
        if (is_unified_solana_collection(origin_info)) {
            get_spl_cache(token_id)
        } else {
            let wrapped_coin_infos
                = &borrow_global<State>(@nft_bridge).wrapped_infos;
            let wrapped_info = table::borrow(wrapped_coin_infos, origin_info);
            (string32::from_string(&collection_name), wrapped_info.symbol)
        }
    }

    public(friend) fun wrapped_asset_signer_exists(origin_info: OriginInfo): bool acquires State {
        let wrapped_coin_signer_caps
            = &borrow_global<State>(@nft_bridge).wrapped_infos;
        table::contains(wrapped_coin_signer_caps, origin_info)
    }


    /// Tokens from Solana currently all have a token_address of [1u8; 32], i.e.
    /// 32 1-bytes. This was originally devised back when Solana NFTs didn't
    /// have collection information, and minting each NFT into a different
    /// contract would have been too expensive on Eth, so instead all Solana
    /// NFTs appear to originate from a single collection.
    ///
    /// This requires some additional bookkeping however, in particular the name
    /// and symbol of the original collection are no longer retrievable from
    /// just the wrapped collection, so we need to store those separately.
    ///
    /// This function determines whether a Solana NFT is to be minted into the
    /// unified collection. In addition to checking the source chain, we also
    /// check the token address. Doing so is future proof: when the Solana
    /// implementation is ugpraded to use the collection key as opposed to the
    /// dummy address as the token_address, newly transferred NFTs will simply
    /// be minted into their respective collections without needing to upgrade
    /// the aptos contract.
    public fun is_unified_solana_collection(origin_info: OriginInfo): bool {
        let token_chain = get_origin_info_token_chain(&origin_info);
        let token_address = get_origin_info_token_address(&origin_info);
        let dummy_address = x"0101010101010101010101010101010101010101010101010101010101010101";
        token_chain == u16::from_u64(1) && token_address == external_address::from_bytes(dummy_address)
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
            spl_cache: table::new(),
            }
        );
    }
}
