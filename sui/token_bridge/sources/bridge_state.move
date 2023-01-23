module token_bridge::bridge_state {
    use std::option::{Self, Option};
    use std::ascii::{Self};

    use sui::object::{Self, UID};
    use sui::vec_map::{Self, VecMap};
    use sui::tx_context::{TxContext};
    use sui::coin::{Self, Coin, TreasuryCap, CoinMetadata};
    use sui::transfer::{Self};
    use sui::tx_context::{Self};
    use sui::sui::SUI;

    use token_bridge::string32;
    use token_bridge::dynamic_set;
    use token_bridge::asset_meta::{Self, AssetMeta};

    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::myu16::{U16};
    use wormhole::wormhole::{Self};
    use wormhole::state::{Self as wormhole_state, State as WormholeState};
    use wormhole::emitter::{EmitterCapability};
    use wormhole::set::{Self, Set};

    const E_IS_NOT_WRAPPED_ASSET: u64 = 0;
    const E_IS_NOT_REGISTERED_NATIVE_ASSET: u64 = 1;
    const E_COIN_TYPE_HAS_NO_REGISTERED_INTEGER_ADDRESS: u64 = 2;
    const E_COIN_TYPE_HAS_REGISTERED_INTEGER_ADDRESS: u64 = 3;
    const E_ORIGIN_CHAIN_MISMATCH: u64 = 4;
    const E_ORIGIN_ADDRESS_MISMATCH: u64 = 5;
    const E_IS_WRAPPED_ASSET: u64 = 6;

    friend token_bridge::vaa;
    friend token_bridge::register_chain;
    friend token_bridge::wrapped;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::transfer_tokens;
    friend token_bridge::attest_token;
    #[test_only]
    friend token_bridge::bridge_state_test;
    #[test_only]
    friend token_bridge::complete_transfer_test;
    #[test_only]
    friend token_bridge::token_bridge_vaa_test;
    #[test_only]
    friend token_bridge::complete_transfer_with_payload_test;

    /// Capability for creating a bridge state object, granted to sender when this
    /// module is deployed
    struct DeployerCapability has key, store {id: UID}

    /// WrappedAssetInfo<CoinType> stores all the metadata about a wrapped asset
    struct WrappedAssetInfo<phantom CoinType> has key, store {
        id: UID,
        token_chain: U16,
        token_address: ExternalAddress,
        treasury_cap: TreasuryCap<CoinType>,
    }

    struct NativeAssetInfo<phantom CoinType> has key, store {
        id: UID,
        // Even though we can look up token_chain at any time from wormhole state,
        // it can be more efficient to store it here locally so we don't have to do lookups.
        custody: Coin<CoinType>,
        asset_meta: AssetMeta,
    }

    /// OriginInfo is a non-Sui object that stores info about a tokens native token
    /// chain and address
    struct OriginInfo<phantom CoinType> has store, copy, drop {
        token_chain: U16,
        token_address: ExternalAddress,
    }

    public fun get_token_chain_from_origin_info<CoinType>(origin_info: &OriginInfo<CoinType>): U16 {
        return origin_info.token_chain
    }

    public fun get_token_address_from_origin_info<CoinType>(origin_info: &OriginInfo<CoinType>): ExternalAddress {
        return origin_info.token_address
    }

    public fun get_origin_info_from_wrapped_asset_info<CoinType>(wrapped_asset_info: &WrappedAssetInfo<CoinType>): OriginInfo<CoinType> {
        OriginInfo { token_chain: wrapped_asset_info.token_chain, token_address: wrapped_asset_info.token_address }
    }

    public fun get_origin_info_from_native_asset_info<CoinType>(native_asset_info: &NativeAssetInfo<CoinType>): OriginInfo<CoinType> {
        let asset_meta = &native_asset_info.asset_meta;
        let token_chain = asset_meta::get_token_chain(asset_meta);
        let token_address = asset_meta::get_token_address(asset_meta);
        OriginInfo { token_chain, token_address }
    }

    public(friend) fun create_wrapped_asset_info<CoinType>(
        token_chain: U16,
        token_address: ExternalAddress,
        treasury_cap: TreasuryCap<CoinType>,
        ctx: &mut TxContext
    ): WrappedAssetInfo<CoinType> {
        return WrappedAssetInfo {
            id: object::new(ctx),
            token_chain,
            token_address,
            treasury_cap
        }
    }

    // Integer label for coin types registered with Wormhole

    struct NativeIdRegistry has key, store {
        id: UID,
        index: u64, // next index to use
    }

    fun next_native_id(registry: &mut NativeIdRegistry): ExternalAddress {
        use wormhole::serialize::serialize_u64;

        let cur_index = registry.index;
        registry.index = cur_index + 1;
        let bytes = std::vector::empty<u8>();
        serialize_u64(&mut bytes, cur_index);
        external_address::from_bytes(bytes)
    }

    // Treasury caps, token stores, consumed VAAs, registered emitters, etc.
    // are stored as dynamic fields of bridge state.
    struct BridgeState has key, store {
        id: UID,

        /// Set of consumed VAA hashes
        consumed_vaas: Set<vector<u8>>,

        /// Token bridge owned emitter capability
        emitter_cap: EmitterCapability,

        /// Mapping of bridge contracts on other chains
        registered_emitters: VecMap<U16, ExternalAddress>,

        native_id_registry: NativeIdRegistry,
    }

    fun init(ctx: &mut TxContext) {
        transfer::transfer(DeployerCapability{id: object::new(ctx)}, tx_context::sender(ctx));
    }

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        transfer::transfer(DeployerCapability{id: object::new(ctx)}, tx_context::sender(ctx));
    }

    // converts owned state object into a shared object, so that anyone can get a reference to &mut State
    // and pass it into various functions
    public entry fun init_and_share_state(
        deployer: DeployerCapability,
        emitter_cap: EmitterCapability,
        ctx: &mut TxContext
    ) {
        let DeployerCapability{ id } = deployer;
        object::delete(id);
        let state = BridgeState {
            id: object::new(ctx),
            consumed_vaas: set::new(ctx),
            emitter_cap,
            registered_emitters: vec_map::empty(),
            native_id_registry: NativeIdRegistry {
                id: object::new(ctx),
                index: 1,
            }
        };

        // permanently shares state
        transfer::share_object(state);
    }

    public(friend) fun deposit<CoinType>(
        bridge_state: &mut BridgeState,
        coin: Coin<CoinType>,
    ) {
        // TODO: create custom errors for each dynamic_set::borrow_mut
        let native_asset = dynamic_set::borrow_mut<NativeAssetInfo<CoinType>>(&mut bridge_state.id);
        coin::join<CoinType>(&mut native_asset.custody, coin);
    }

    public(friend) fun withdraw<CoinType>(
        _verified_coin_witness: VerifiedCoinType<CoinType>,
        bridge_state: &mut BridgeState,
        value: u64,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        let native_asset = dynamic_set::borrow_mut<NativeAssetInfo<CoinType>>(&mut bridge_state.id);
        coin::split<CoinType>(&mut native_asset.custody, value, ctx)
    }

    public(friend) fun mint<CoinType>(
        _verified_coin_witness: VerifiedCoinType<CoinType>,
        bridge_state: &mut BridgeState,
        value: u64,
        ctx: &mut TxContext,
    ): Coin<CoinType> {
        let wrapped_info = dynamic_set::borrow_mut<WrappedAssetInfo<CoinType>>(&mut bridge_state.id);
        coin::mint<CoinType>(&mut wrapped_info.treasury_cap, value, ctx)
    }

    public(friend) fun burn<CoinType>(
        bridge_state: &mut BridgeState,
        coin: Coin<CoinType>,
    ) {
        let wrapped_info = dynamic_set::borrow_mut<WrappedAssetInfo<CoinType>>(&mut bridge_state.id);
        coin::burn<CoinType>(&mut wrapped_info.treasury_cap, coin);
    }

    public(friend) fun publish_message(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
    ): u64 {
        wormhole::publish_message(
            &mut bridge_state.emitter_cap,
            wormhole_state,
            nonce,
            payload,
            message_fee,
        )
    }

    /// getters

    public fun vaa_is_consumed(state: &BridgeState, hash: vector<u8>): bool {
        set::contains(&state.consumed_vaas, hash)
    }

    public fun get_registered_emitter(state: &BridgeState, chain_id: &U16): Option<ExternalAddress> {
        if (vec_map::contains(&state.registered_emitters, chain_id)) {
            option::some(*vec_map::get(&state.registered_emitters, chain_id))
        } else {
            option::none()
        }
    }

    public fun is_wrapped_asset<CoinType>(bridge_state: &BridgeState): bool {
        dynamic_set::exists_<WrappedAssetInfo<CoinType>>(&bridge_state.id)
    }

    public fun is_registered_native_asset<CoinType>(bridge_state: &BridgeState): bool {
        dynamic_set::exists_<NativeAssetInfo<CoinType>>(&bridge_state.id)
    }

    /// Returns the origin information for a CoinType
    public fun origin_info<CoinType>(bridge_state: &BridgeState): OriginInfo<CoinType> {
        if (is_wrapped_asset<CoinType>(bridge_state)) {
            get_wrapped_asset_origin_info<CoinType>(bridge_state)
        } else {
            get_registered_native_asset_origin_info(bridge_state)
        }
    }

    /// A value of type `VerifiedCoinType<T>` witnesses the fact that the type
    /// `T` has been verified to correspond to a particular chain id and token
    /// address (may be either a wrapped or native asset).
    /// The verification is performed by `verify_coin_type`.
    ///
    /// This is important because the coin type is an input to several
    /// functions, and is thus untrusted. Most coin-related functionality
    /// requires passing in a coin type generic argument.
    /// When transferring tokens *out*, that type instantiation determines the
    /// token bridge's behaviour, and thus we just take whatever was supplied.
    /// When transferring tokens *in*, it's the transfer VAA that determines
    /// which coin should be used via the origin chain and origin address
    /// fields.
    ///
    /// For technical reasons, the latter case still requires a type argument to
    /// be passed in (since Move does not support existential types, so we must
    /// rely on old school universal quantification). We must thus verify that
    /// the supplied type corresponds to the origin info in the VAA.
    ///
    /// Accordingly, the `mint` and `withdraw` operations are gated by this
    /// witness type, since these two operations require a VAA to supply the
    /// token information. This ensures that those two functions can't be called
    /// without first verifying the `CoinType`.
    struct VerifiedCoinType<phantom CoinType> has copy, drop {}

    /// See the documentation for `VerifiedCoinType` above.
    public fun verify_coin_type<CoinType>(
        bridge_state: &BridgeState,
        token_chain: U16,
        token_address: ExternalAddress
    ): VerifiedCoinType<CoinType> {
        let coin_origin = origin_info<CoinType>(bridge_state);
        assert!(coin_origin.token_chain == token_chain, E_ORIGIN_CHAIN_MISMATCH);
        assert!(coin_origin.token_address == token_address, E_ORIGIN_ADDRESS_MISMATCH);
        VerifiedCoinType {}
    }

    public fun get_wrapped_asset_origin_info<CoinType>(bridge_state: &BridgeState): OriginInfo<CoinType> {
        assert!(is_wrapped_asset<CoinType>(bridge_state), E_IS_NOT_WRAPPED_ASSET);
        let wrapped_asset_info = dynamic_set::borrow<WrappedAssetInfo<CoinType>>(&bridge_state.id);
        get_origin_info_from_wrapped_asset_info(wrapped_asset_info)
    }

    public fun get_registered_native_asset_origin_info<CoinType>(bridge_state: &BridgeState): OriginInfo<CoinType> {
        let native_asset_info = dynamic_set::borrow<NativeAssetInfo<CoinType>>(&bridge_state.id);
        get_origin_info_from_native_asset_info(native_asset_info)
    }

    /// setters

    public(friend) fun set_registered_emitter(state: &mut BridgeState, chain_id: U16, emitter: ExternalAddress) {
        if (vec_map::contains<U16, ExternalAddress>(&mut state.registered_emitters, &chain_id)){
            vec_map::remove<U16, ExternalAddress>(&mut state.registered_emitters, &chain_id);
        };
        vec_map::insert<U16, ExternalAddress>(&mut state.registered_emitters, chain_id, emitter);
    }

    /// dynamic ops

    public(friend) fun store_consumed_vaa(bridge_state: &mut BridgeState, vaa: vector<u8>) {
        set::add(&mut bridge_state.consumed_vaas, vaa);
    }

    public(friend) fun register_wrapped_asset<CoinType>(bridge_state: &mut BridgeState, wrapped_asset_info: WrappedAssetInfo<CoinType>) {
        dynamic_set::add<WrappedAssetInfo<CoinType>>(&mut bridge_state.id, wrapped_asset_info);
    }

    public(friend) fun register_native_asset<CoinType>(
        wormhole_state: &WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        ctx: &mut TxContext
    ): AssetMeta {
        assert!(!is_wrapped_asset<CoinType>(bridge_state), E_IS_WRAPPED_ASSET); // TODO - test
        let asset_meta = asset_meta::create(
            next_native_id(&mut bridge_state.native_id_registry),
            wormhole_state::get_chain_id(wormhole_state), // TODO: should we just hardcode this?
            coin::get_decimals<CoinType>(coin_meta), // decimals
            string32::from_bytes(ascii::into_bytes(coin::get_symbol<CoinType>(coin_meta))), // symbol
            string32::from_string(&coin::get_name<CoinType>(coin_meta)) // name
        );
        let native_asset_info = NativeAssetInfo<CoinType> {
            id: object::new(ctx),
            custody: coin::zero(ctx),
            asset_meta,
        };
        dynamic_set::add<NativeAssetInfo<CoinType>>(&mut bridge_state.id, native_asset_info);
        asset_meta
    }

}

#[test_only]
module token_bridge::bridge_state_test{
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_from_address, take_shared, return_shared};
    use sui::coin::{CoinMetadata};

    use wormhole::state::{State};
    use wormhole::test_state::{init_wormhole_state};
    use wormhole::wormhole::{Self};
    use wormhole::external_address::{Self};

    use token_bridge::bridge_state::{Self as state, BridgeState, DeployerCapability};
    use token_bridge::native_coin_witness::{Self, NATIVE_COIN_WITNESS};
    use token_bridge::native_coin_witness_v2::{Self, NATIVE_COIN_WITNESS_V2};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    #[test]
    fun test_state_setters() {
        test_state_setters_(scenario())
    }

    #[test]
    fun test_coin_type_addressing(){
        test_coin_type_addressing_(scenario())
    }

    #[test]
    #[expected_failure(abort_code = 0, location=0000000000000000000000000000000000000002::dynamic_field)]
    fun test_coin_type_addressing_failure_case(){
        test_coin_type_addressing_failure_case_(scenario())
    }

    public fun set_up_wormhole_core_and_token_bridges(admin: address, test: Scenario): Scenario {
        // init and share wormhole core bridge
        test =  init_wormhole_state(test, admin);

        // call init for token bridge to get deployer cap
        next_tx(&mut test, admin); {
            state::test_init(ctx(&mut test));
        };

        // register for emitter cap and init_and_share token bridge
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<State>(&test);
            let my_emitter = wormhole::register_emitter(&mut wormhole_state, ctx(&mut test));
            let deployer = take_from_address<DeployerCapability>(&test, admin);
            state::init_and_share_state(deployer, my_emitter, ctx(&mut test));
            return_shared<State>(wormhole_state);
        };

        next_tx(&mut test, admin); {
            let bridge_state = take_shared<BridgeState>(&test);
            return_shared<BridgeState>(bridge_state);
        };

        return test
    }

    fun test_state_setters_(test: Scenario) {
        let (admin, _, _) = people();

        test = set_up_wormhole_core_and_token_bridges(admin, test);

        //test BridgeState setter and getter functions
        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);

            // test store consumed vaa
            state::store_consumed_vaa(&mut state, x"1234");
            assert!(state::vaa_is_consumed(&state, x"1234"), 0);

            // TODO - test store coin store
            // TODO - test store treasury cap

            return_shared<BridgeState>(state);
        };
        test_scenario::end(test);
    }

    fun test_coin_type_addressing_(test: Scenario) {
        let (admin, _, _) = people();

        test = set_up_wormhole_core_and_token_bridges(admin, test);

        //test coin type addressing
        next_tx(&mut test, admin); {
            native_coin_witness::test_init(ctx(&mut test));
            native_coin_witness_v2::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<State>(&test);
            let bridge_state = take_shared<BridgeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let origin_info = state::origin_info<NATIVE_COIN_WITNESS>(&bridge_state);
            let address = state::get_token_address_from_origin_info(&origin_info);
            assert!(address == external_address::from_bytes(x"01"), 0);

            let coin_meta_v2 = take_shared<CoinMetadata<NATIVE_COIN_WITNESS_V2>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS_V2>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta_v2,
                ctx(&mut test)
            );
            let origin_info = state::origin_info<NATIVE_COIN_WITNESS_V2>(&bridge_state);
            let address = state::get_token_address_from_origin_info(&origin_info);
            assert!(address == external_address::from_bytes(x"02"), 0);

            return_shared<State>(wormhole_state);
            return_shared<BridgeState>(bridge_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS_V2>>(coin_meta_v2);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
        };
        test_scenario::end(test);
    }


    fun test_coin_type_addressing_failure_case_(test: Scenario) {
        let (admin, _, _) = people();

        test = set_up_wormhole_core_and_token_bridges(admin, test);

        //test coin type addressing
        next_tx(&mut test, admin); {
            native_coin_witness::test_init(ctx(&mut test));
            native_coin_witness_v2::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<State>(&test);
            let bridge_state = take_shared<BridgeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let origin_info = state::origin_info<NATIVE_COIN_WITNESS>(&bridge_state);
            let address = state::get_token_address_from_origin_info(&origin_info);
            assert!(address == external_address::from_bytes(x"01"), 0);

            // aborts because trying to re-register native coin
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );

            return_shared<State>(wormhole_state);
            return_shared<BridgeState>(bridge_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
        };
        test_scenario::end(test);
    }
}
