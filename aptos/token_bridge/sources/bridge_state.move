module token_bridge::bridge_state {
    use std::table::{Self, Table};
    use std::option::{Self, Option};
    use aptos_framework::type_info::{Self, TypeInfo, type_of};
    use aptos_framework::account::{Self, SignerCapability, create_signer_with_capability};
    use aptos_framework::coin::{Self, Coin};
    use aptos_framework::aptos_coin::{AptosCoin};
    use aptos_framework::bcs::to_bytes;

    use wormhole::u256::{Self, U256};
    use wormhole::u16::{Self, U16};
    use wormhole::emitter::{EmitterCapability};
    use wormhole::state::{get_chain_id, get_governance_contract};
    use wormhole::wormhole;
    use wormhole::set::{Self, Set};

    use token_bridge::transfer;
    use token_bridge::transfer_with_payload;
    use token_bridge::token_hash::{Self, TokenHash};
    use token_bridge::transfer_result::{Self, TransferResult};
    use token_bridge::utils::{Self};
    //use token_bridge::wrapped::{Self};

    friend token_bridge::contract_upgrade;
    friend token_bridge::register_chain;
    friend token_bridge::token_bridge;
    friend token_bridge::vaa;
    friend token_bridge::attest_token;
    friend token_bridge::wrapped;

    #[test_only]
    friend token_bridge::token_bridge_test;

    const E_COIN_NOT_REGISTERED: u64 = 2;
    const E_FEE_EXCEEDS_AMOUNT: u64 = 3;

    struct Asset has key, store {
        chain_id: U16,
        asset_address: vector<u8>,
    }

    // the native chain and address of a wrapped token
    struct OriginInfo has key, store, copy, drop {
        token_address: vector<u8>,
        token_chain: U16,
    }

    public(friend) fun create_origin_info(
        token_address: vector<u8>,
        token_chain: U16
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

    // TODO: these shouldn't be entry functions...

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

    public fun origin_info<CoinType>(): OriginInfo acquires OriginInfo {
        *borrow_global<OriginInfo>(type_info::account_address(&type_of<CoinType>()))
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
        move_to(coin_signer, origin_info);
        set_wrapped_asset<CoinType>(&origin_info);
        set_wrapped_asset_type_info<CoinType>();
    }

    public fun get_origin_info_token_address(info: OriginInfo): vector<u8>{
        info.token_address
    }

    public fun get_origin_info_token_chain(info: OriginInfo): U16{
        info.token_chain
    }

    public entry fun transfer_tokens_with_signer<CoinType>(
        sender: &signer,
        amount: u64,
        recipient_chain: u64,
        recipient: vector<u8>,
        relayer_fee: u64,
        wormhole_fee: u64,
        nonce: u64
        ): u64 acquires State, OriginInfo {
        let coins = coin::withdraw<CoinType>(sender, amount);
        //let relayer_fee_coins = coin::withdraw<AptosCoin>(sender, relayer_fee);
        let wormhole_fee_coins = coin::withdraw<AptosCoin>(sender, wormhole_fee);
        transfer_tokens<CoinType>(coins, wormhole_fee_coins, u16::from_u64(recipient_chain), recipient, relayer_fee, nonce)
    }

    public fun transfer_tokens<CoinType>(
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<AptosCoin>,
        recipient_chain: U16,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u64
        ): u64 acquires State, OriginInfo {
        let wormhole_fee = coin::value<AptosCoin>(&wormhole_fee_coins);
        let result = transfer_tokens_internal<CoinType>(coins, relayer_fee, wormhole_fee);
        log_transfer(
            transfer_result::get_token_chain(&result),
            transfer_result::get_token_address(&result),
            transfer_result::get_normalized_amount(&result),
            recipient_chain,
            recipient,
            transfer_result::get_normalized_relayer_fee(&result),
            wormhole_fee_coins,
            nonce
        )
    }

    public fun transfer_tokens_with_payload_with_signer<CoinType>(
        sender: &signer,
        amount: u64,
        wormhole_fee: u64,
        recipient_chain: U16,
        recipient: vector<u8>,
        nonce: u64,
        payload: vector<u8>
        ): u64 acquires State, OriginInfo {
        let coins = coin::withdraw<CoinType>(sender, amount);
        let wormhole_fee_coins = coin::withdraw<AptosCoin>(sender, wormhole_fee);
        transfer_tokens_with_payload(coins, wormhole_fee_coins, recipient_chain, recipient, nonce, payload)
    }

    public fun transfer_tokens_with_payload<CoinType>(
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<AptosCoin>,
        recipient_chain: U16,
        recipient: vector<u8>,
        nonce: u64,
        payload: vector<u8>
        ): u64 acquires State, OriginInfo {
        let result = transfer_tokens_internal<CoinType>(coins, 0, 0);
        log_transfer_with_payload(
            transfer_result::get_token_chain(&result),
            transfer_result::get_token_address(&result),
            transfer_result::get_normalized_amount(&result),
            recipient_chain,
            recipient,
            wormhole_fee_coins,
            nonce,
            payload
        )
    }

    // transfer a native or wraped token from sender to token_bridge
    fun transfer_tokens_internal<CoinType>(
        coins: Coin<CoinType>,
        relayer_fee: u64,
        wormhole_fee: u64,
        ): TransferResult acquires State, OriginInfo {
        let token_address = token_hash::derive<CoinType>();

        // transfer coin to token_bridge
        if (!coin::is_account_registered<CoinType>(@token_bridge)){
            coin::register<CoinType>(&token_bridge_signer());
        };
        if (!coin::is_account_registered<AptosCoin>(@token_bridge)){
            coin::register<AptosCoin>(&token_bridge_signer());
        };
        let amount = coin::value<CoinType>(&coins);
        coin::deposit<CoinType>(@token_bridge, coins);

        let origin_chain;
        let origin_address;
        if (is_wrapped_asset<CoinType>()) {
            let origin_info = origin_info<CoinType>();
            origin_chain = origin_info.token_chain;
            origin_address = origin_info.token_address;
            // now we burn the wrapped coins to remove them from circulation
            // TODO - wrapped::burn<CoinType>(amount);
            // wrapped::burn<CoinType>(amount);
            // problem here is that wrapped imports state, so state can't import wrapped...
        } else {
             if (!is_registered_native_asset<CoinType>()) {
                set_native_asset_type_info<CoinType>();
             };
            origin_chain = get_chain_id();
            origin_address = token_hash::get_bytes(&token_address);
        };

        let decimals_token = coin::decimals<CoinType>();
        let decimals_aptos = coin::decimals<AptosCoin>();

        let normalized_amount = utils::normalize_amount(u256::from_u64(amount), decimals_token);
        let normalized_relayer_fee = utils::normalize_amount(u256::from_u64(relayer_fee), decimals_aptos);

        let transfer_result: TransferResult = transfer_result::create(
            origin_chain,
            origin_address,
            normalized_amount,
            normalized_relayer_fee,
            u256::from_u64(wormhole_fee)
        );
        transfer_result
    }

    public(friend) fun log_transfer(
        token_chain: U16,
        token_address: vector<u8>,
        amount: U256,
        recipient_chain: U16,
        recipient: vector<u8>,
        fee: U256,
        wormhole_fee: Coin<AptosCoin>,
        nonce: u64
    ): u64 acquires State{
        // TODO - check fee values
        //let fee_value = coin::value<AptosCoin>(&wormhole_fee);
        //assert!(u256::compare(&u256::from_u64(fee_value), &amount)==1, E_FEE_EXCEEDS_AMOUNT);
        let transfer = transfer::create(1, amount, token_address, token_chain, recipient, recipient_chain, fee);
        let payload = transfer::encode(transfer);
        publish_message(
            nonce,
            payload,
            wormhole_fee,
        )
    }

    public(friend) fun log_transfer_with_payload(
        token_chain: U16,
        token_address: vector<u8>,
        amount: U256,
        recipient_chain: U16,
        recipient: vector<u8>,
        wormhole_fee: Coin<AptosCoin>,
        nonce: u64,
        payload: vector<u8>
    ): u64 acquires State{
        let transfer = transfer_with_payload::create(
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            to_bytes<address>(&@token_bridge), //TODO - is token bridge the only one who will ever call log_transfer_with_payload?
            payload
        );
        let payload = transfer_with_payload::encode(transfer);
        publish_message(
            nonce,
            payload,
            wormhole_fee,
        )
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
