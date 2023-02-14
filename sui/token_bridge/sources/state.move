module token_bridge::state {
    use std::option::{Self, Option};
    use std::ascii::{Self};

    use sui::object::{Self, UID};
    use sui::vec_map::{Self, VecMap};
    use sui::tx_context::{TxContext};
    use sui::coin::{Self, Coin, CoinMetadata};
    use sui::transfer::{Self};
    use sui::tx_context::{Self};
    use sui::sui::{SUI};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::native_asset::{Self, NativeAsset};
    use token_bridge::native_id_registry::{Self, NativeIdRegistry};
    use token_bridge::string32::{Self};
    use token_bridge::token_info::{Self, TokenInfo};
    use token_bridge::wrapped_asset::{Self, WrappedAsset};

    use wormhole::dynamic_set::{Self};
    use wormhole::external_address::{ExternalAddress};
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
    friend token_bridge::create_wrapped;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::transfer_tokens;
    friend token_bridge::transfer_tokens_with_payload;
    friend token_bridge::attest_token;
    #[test_only]
    friend token_bridge::bridge_state_test;
    #[test_only]
    friend token_bridge::complete_transfer_test;
    #[test_only]
    friend token_bridge::token_bridge_vaa_test;
    #[test_only]
    friend token_bridge::complete_transfer_with_payload_test;
    #[test_only]
    friend token_bridge::transfer_token_test;

    /// Capability for creating a bridge state object, granted to sender when this
    /// module is deployed
    struct DeployerCapability has key, store {
        id: UID
    }

    // Treasury caps, token stores, consumed VAAs, registered emitters, etc.
    // are stored as dynamic fields of bridge state.
    struct State has key, store {
        id: UID,

        /// Set of consumed VAA hashes
        consumed_vaas: Set<vector<u8>>,

        /// Token bridge owned emitter capability
        emitter_cap: EmitterCapability,

        /// Mapping of bridge contracts on other chains
        registered_emitters: VecMap<u16, ExternalAddress>,

        native_id_registry: NativeIdRegistry,
    }

    fun init(ctx: &mut TxContext) {
        transfer::transfer(
            DeployerCapability{
                id: object::new(ctx)
            },
            tx_context::sender(ctx)
        );
    }

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        transfer::transfer(
            DeployerCapability{
                id: object::new(ctx)
            },
            tx_context::sender(ctx)
        );
    }

    // converts owned state object into a shared object, so that anyone can get
    // a reference to &mut State and pass it into various functions
    public entry fun init_and_share_state(
        deployer: DeployerCapability,
        worm_state: &mut WormholeState,
        ctx: &mut TxContext
    ) {
        let DeployerCapability{ id } = deployer;
        let emitter_cap = wormhole::register_emitter(worm_state, ctx);
        object::delete(id);
        let state = State {
            id: object::new(ctx),
            consumed_vaas: set::new(ctx),
            emitter_cap,
            registered_emitters: vec_map::empty(),
            native_id_registry: native_id_registry::new(ctx)
        };

        // permanently shares state
        transfer::share_object(state);
    }

    public(friend) fun deposit<CoinType>(
        self: &mut State,
        coin: Coin<CoinType>,
    ) {
        // TODO: create custom errors for each dynamic_set::borrow_mut
        let asset =
            dynamic_set::borrow_mut<NativeAsset<CoinType>>(
                &mut self.id
            );
        native_asset::deposit(asset, coin);
    }

    #[test_only]
    public fun test_deposit<CoinType>(
        self: &mut State,
        coin: Coin<CoinType>
    ) {
        deposit(self, coin);
    }

    public(friend) fun withdraw<CoinType>(
        self: &mut State,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        let asset =
            dynamic_set::borrow_mut<NativeAsset<CoinType>>(
                &mut self.id
            );
        native_asset::withdraw(asset, amount, ctx)
    }

    public(friend) fun burn<CoinType>(
        self: &mut State,
        coin: Coin<CoinType>,
    ) {
        let asset =
            dynamic_set::borrow_mut<WrappedAsset<CoinType>>(
                &mut self.id
            );
        wrapped_asset::burn(asset, coin);
    }

    public(friend) fun mint<CoinType>(
        self: &mut State,
        amount: u64,
        ctx: &mut TxContext,
    ): Coin<CoinType> {
        let asset =
            dynamic_set::borrow_mut<WrappedAsset<CoinType>>(
                &mut self.id
            );
        wrapped_asset::mint(asset, amount, ctx)
    }

    // Note: we only examine the balance of native assets, because the token
    // bridge does not custody wrapped assets (only mints and burns them)
    #[test_only]
    public fun balance<CoinType>(
        bridge_state: &mut State
    ): u64 {
        let asset =
            dynamic_set::borrow_mut<NativeAsset<CoinType>>(
                &mut bridge_state.id
            );
        native_asset::balance(asset)
    }

    public(friend) fun publish_message(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut State,
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

    public fun vaa_is_consumed(state: &State, hash: vector<u8>): bool {
        set::contains(&state.consumed_vaas, hash)
    }

    public fun get_registered_emitter(
        state: &State,
        chain_id: u16
    ): Option<ExternalAddress> {
        if (vec_map::contains(&state.registered_emitters, &chain_id)) {
            option::some(*vec_map::get(&state.registered_emitters, &chain_id))
        } else {
            option::none()
        }
    }

    public fun is_wrapped_asset<CoinType>(bridge_state: &State): bool {
        dynamic_set::exists_<WrappedAsset<CoinType>>(&bridge_state.id)
    }

    public fun is_registered_native_asset<CoinType>(bridge_state: &State): bool {
        dynamic_set::exists_<NativeAsset<CoinType>>(&bridge_state.id)
    }

    /// Returns the origin information for a CoinType
    public fun origin_info<CoinType>(bridge_state: &State): TokenInfo<CoinType> {
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
        self: &State,
        token_chain: u16,
        token_address: ExternalAddress
    ): VerifiedCoinType<CoinType> {
        let info = origin_info<CoinType>(self);
        assert!(
            token_info::chain(&info) == token_chain,
            E_ORIGIN_CHAIN_MISMATCH
        );
        assert!(
            token_info::addr(&info) == token_address,
            E_ORIGIN_ADDRESS_MISMATCH
        );
        VerifiedCoinType {}
    }

    public fun get_wrapped_decimals<CoinType>(bridge_state: &State): u8 {
        let asset =
            dynamic_set::borrow<WrappedAsset<CoinType>>(&bridge_state.id);
        wrapped_asset::decimals(asset)
    }

    public fun get_native_decimals<CoinType>(bridge_state: &State): u8 {
        let asset =
            dynamic_set::borrow<NativeAsset<CoinType>>(&bridge_state.id);
        native_asset::decimals(asset)
    }

    public fun get_wrapped_asset_origin_info<CoinType>(bridge_state: &State): TokenInfo<CoinType> {
        assert!(is_wrapped_asset<CoinType>(bridge_state), E_IS_NOT_WRAPPED_ASSET);
        let asset =
            dynamic_set::borrow<WrappedAsset<CoinType>>(&bridge_state.id);
        wrapped_asset::to_token_info(asset)
    }

    public fun get_registered_native_asset_origin_info<CoinType>(bridge_state: &State): TokenInfo<CoinType> {
        let asset = dynamic_set::borrow<NativeAsset<CoinType>>(&bridge_state.id);
        token_info::new(
            false, // is_wrapped
            native_asset::token_chain(asset),
            native_asset::token_address(asset)
        )
    }

    /// setters

    public(friend) fun set_registered_emitter(state: &mut State, chain_id: u16, emitter: ExternalAddress) {
        if (vec_map::contains<u16, ExternalAddress>(&mut state.registered_emitters, &chain_id)){
            vec_map::remove<u16, ExternalAddress>(&mut state.registered_emitters, &chain_id);
        };
        vec_map::insert<u16, ExternalAddress>(&mut state.registered_emitters, chain_id, emitter);
    }

    #[test_only]
    public fun test_set_registered_emitter(state: &mut State, chain_id: u16, emitter: ExternalAddress) {
        set_registered_emitter(state, chain_id, emitter);
    }

    /// dynamic ops

    public(friend) fun store_consumed_vaa(bridge_state: &mut State, vaa: vector<u8>) {
        set::add(&mut bridge_state.consumed_vaas, vaa);
    }

    public(friend) fun register_wrapped_asset<CoinType>(bridge_state: &mut State, wrapped_asset_info: WrappedAsset<CoinType>) {
        dynamic_set::add<WrappedAsset<CoinType>>(&mut bridge_state.id, wrapped_asset_info);
    }

    public(friend) fun register_native_asset<CoinType>(
        worm_state: &WormholeState,
        bridge_state: &mut State,
        coin_metadata: &CoinMetadata<CoinType>,
        ctx: &mut TxContext
    ): AssetMeta {
        assert!(!is_wrapped_asset<CoinType>(bridge_state), E_IS_WRAPPED_ASSET); // TODO - test

        let chain = wormhole_state::get_chain_id(worm_state);
        let addr = native_id_registry::next_id(&mut bridge_state.native_id_registry);
        let decimals = coin::get_decimals(coin_metadata);

        let asset = native_asset::new(
            chain,
            addr,
            decimals,
            ctx
        );
        dynamic_set::add<NativeAsset<CoinType>>(&mut bridge_state.id, asset);

        asset_meta::create(
            addr,
            chain, // TODO: should we just hardcode this?
            decimals,
            string32::from_bytes(
                ascii::into_bytes(coin::get_symbol<CoinType>(coin_metadata))
            ),
            string32::from_string(&coin::get_name<CoinType>(coin_metadata))
        )
    }

}

#[test_only]
module token_bridge::bridge_state_test{
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_from_address, take_shared, return_shared};
    use sui::coin::{CoinMetadata};

    use wormhole::state::{State as WormholeState};
    use wormhole::test_state::{init_wormhole_state};
    use wormhole::external_address::{Self};

    use token_bridge::state::{Self, State, DeployerCapability};
    use token_bridge::native_coin_witness::{Self, NATIVE_COIN_WITNESS};
    use token_bridge::native_coin_witness_v2::{Self, NATIVE_COIN_WITNESS_V2};
    use token_bridge::token_info::{Self};

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
    #[expected_failure(abort_code = 0, location=sui::dynamic_field)]
    fun test_coin_type_addressing_failure_case(){
        test_coin_type_addressing_failure_case_(scenario())
    }

    public fun set_up_wormhole_core_and_token_bridges(admin: address, test: Scenario): Scenario {
        // init and share wormhole core bridge
        test =  init_wormhole_state(test, admin, 0);

        // call init for token bridge to get deployer cap
        next_tx(&mut test, admin); {
            state::test_init(ctx(&mut test));
        };

        // register for emitter cap and init_and_share token bridge
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let deployer = take_from_address<DeployerCapability>(&test, admin);
            state::init_and_share_state(deployer, &mut wormhole_state, ctx(&mut test));
            return_shared<WormholeState>(wormhole_state);
        };

        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            return_shared<State>(bridge_state);
        };

        return test
    }

    fun test_state_setters_(test: Scenario) {
        let (admin, _, _) = people();

        test = set_up_wormhole_core_and_token_bridges(admin, test);

        //test State setter and getter functions
        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);

            // test store consumed vaa
            state::store_consumed_vaa(&mut state, x"1234");
            assert!(state::vaa_is_consumed(&state, x"1234"), 0);

            // TODO - test store coin store
            // TODO - test store treasury cap

            return_shared<State>(state);
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
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            let coin_meta =
                take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);

            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let info = state::origin_info<NATIVE_COIN_WITNESS>(&bridge_state);
            let expected_addr = external_address::from_bytes(x"01");
            assert!(token_info::addr(&info) == expected_addr, 0);

            let coin_meta_v2 =
                take_shared<CoinMetadata<NATIVE_COIN_WITNESS_V2>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS_V2>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta_v2,
                ctx(&mut test)
            );
            let info =
                state::origin_info<NATIVE_COIN_WITNESS_V2>(&bridge_state);
            let expected_addr = external_address::from_bytes(x"02");
            assert!(token_info::addr(&info) == expected_addr, 0);

            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
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
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let info = state::origin_info<NATIVE_COIN_WITNESS>(&bridge_state);
            let expected_addr = external_address::from_bytes(x"01");
            assert!(token_info::addr(&info) == expected_addr, 0);

            // aborts because trying to re-register native coin
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut wormhole_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );

            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
        };
        test_scenario::end(test);
    }
}
