module token_bridge::state {
    use sui::balance::{Balance, Supply};
    use sui::coin::{CoinMetadata};
    use sui::object::{Self, UID};
    use sui::sui::{SUI};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::emitter::{EmitterCap};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::set::{Self, Set};
    use wormhole::state::{State as WormholeState};

    use token_bridge::asset_meta::{AssetMeta};
    use token_bridge::emitter_registry::{Self, EmitterRegistry};
    use token_bridge::registered_tokens::{Self, RegisteredTokens};

    const E_UNREGISTERED_EMITTER: u64 = 0;
    const E_EMITTER_ALREADY_REGISTERED: u64 = 1;
    const E_VAA_ALREADY_CONSUMED: u64 = 2;
    const E_CANONICAL_TOKEN_INFO_MISMATCH: u64 = 3;

    friend token_bridge::attest_token;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::create_wrapped;
    friend token_bridge::register_chain;
    friend token_bridge::transfer_tokens;
    friend token_bridge::transfer_tokens_with_payload;
    friend token_bridge::vaa;

    /// Capability for creating a bridge state object, granted to sender when this
    /// module is deployed
    struct DeployerCap has key, store {
        id: UID
    }

    /// Treasury caps, token stores, consumed VAAs, registered emitters, etc.
    /// are stored as dynamic fields of bridge state.
    struct State has key, store {
        id: UID,

        /// Set of consumed VAA hashes
        consumed_vaa_hashes: Set<Bytes32>,

        /// Token bridge owned emitter capability
        emitter_cap: EmitterCap,

        emitter_registry: EmitterRegistry,

        registered_tokens: RegisteredTokens,
    }

    fun init(ctx: &mut TxContext) {
        transfer::transfer(
            DeployerCap {
                id: object::new(ctx)
            },
            tx_context::sender(ctx)
        );
    }

    #[test_only]
    public fun init_test_only(ctx: &mut TxContext) {
        init(ctx)
    }

    /// Converts owned state object into a shared object, so that anyone can get
    /// a reference to &mut State and pass it into various functions.
    public entry fun init_and_share_state(
        deployer: DeployerCap,
        worm_state: &mut WormholeState,
        ctx: &mut TxContext
    ) {
        let DeployerCap { id } = deployer;
        object::delete(id);

        let state = State {
            id: object::new(ctx),
            consumed_vaa_hashes: set::new(ctx),
            emitter_cap: wormhole::state::new_emitter(worm_state, ctx),
            emitter_registry: emitter_registry::new(ctx),
            registered_tokens: registered_tokens::new(ctx)
        };

        // Permanently shares state.
        transfer::share_object(state);
    }

    public fun governance_module(): Bytes32 {
        // A.K.A. "TokenBridge".
        bytes32::new(
            x"000000000000000000000000000000000000000000546f6b656e427269646765"
        )
    }

    public(friend) fun take_from_circulation<CoinType>(
        self: &mut State,
        removed: Balance<CoinType>
    ) {
        let registry = &mut self.registered_tokens;
        if (registered_tokens::is_wrapped<CoinType>(registry)) {
            registered_tokens::burn(registry, removed);
        } else {
            registered_tokens::deposit(registry, removed);
        }
    }

    #[test_only]
    /// Exposing method so an integrator can test redeeming native tokens.
    public fun take_from_circulation_test_only<CoinType>(
        self: &mut State,
        removed: Balance<CoinType>
    ) {
        take_from_circulation(self, removed)
    }

    public(friend) fun put_into_circulation<CoinType>(
        self: &mut State,
        amount: u64
    ): Balance<CoinType> {
        let registry = &mut self.registered_tokens;
        if (registered_tokens::is_wrapped<CoinType>(registry)) {
            registered_tokens::mint(&mut self.registered_tokens, amount)
        } else {
            registered_tokens::withdraw(&mut self.registered_tokens, amount)
        }
    }

    #[test_only]
    /// Exposing method so an integrator can test sending native tokens.
    public fun put_into_circulation_test_only<CoinType>(
        self: &mut State,
        amount: u64
    ): Balance<CoinType> {
        put_into_circulation(self, amount)
    }

    /// We only examine the balance of native assets.
    public fun custody_balance<CoinType>(self: &State): u64 {
        registered_tokens::balance<CoinType>(&self.registered_tokens)
    }

    /// We only examine the total supply of wrapped assets.
    public fun wrapped_supply<CoinType>(self: &State): u64 {
        registered_tokens::total_supply<CoinType>(&self.registered_tokens)
    }

    public(friend) fun publish_wormhole_message(
        self: &mut State,
        worm_state: &mut WormholeState,
        nonce: u32,
        payload: vector<u8>,
        message_fee: Balance<SUI>,
    ): u64 {
        use wormhole::publish_message::{publish_message};

        publish_message(
            worm_state,
            &mut self.emitter_cap,
            nonce,
            payload,
            message_fee,
        )
    }

    public(friend) fun consume_vaa_hash(self: &mut State, vaa_hash: Bytes32) {
        let consumed = &mut self.consumed_vaa_hashes;
        assert!(!set::contains(consumed, vaa_hash), E_VAA_ALREADY_CONSUMED);
        set::add(consumed, vaa_hash);
    }

    public fun registered_emitter(
        self: &State,
        chain: u16
    ): ExternalAddress {
        emitter_registry::emitter_address(&self.emitter_registry, chain)
    }

    public fun is_registered_asset<CoinType>(self: &State): bool {
        registered_tokens::has<CoinType>(&self.registered_tokens)
    }

    public fun is_native_asset<CoinType>(self: &State): bool {
        registered_tokens::is_native<CoinType>(&self.registered_tokens)
    }

    public fun is_wrapped_asset<CoinType>(self: &State): bool {
        registered_tokens::is_wrapped<CoinType>(&self.registered_tokens)
    }

    /// Retrieves canonical token info from the registry, which are the native
    /// chain ID and token address.
    public fun token_info<CoinType>(self: &State): (u16, ExternalAddress) {
        registered_tokens::canonical_info<CoinType>(&self.registered_tokens)
    }

    /// Assert that given canonical token info agrees with what exists in the
    /// registry for this particular `CoinType`.
    public fun assert_registered_token<CoinType>(
        self: &State,
        token_chain: u16,
        token_address: ExternalAddress
    ) {
        let (expected_chain, expected_addr) = token_info<CoinType>(self);
        assert!(
            token_chain == expected_chain && token_address == expected_addr,
            E_CANONICAL_TOKEN_INFO_MISMATCH
        );

    }

    /// Retrieve decimals for coins (wrapped and native) in registry.
    public fun coin_decimals<CoinType>(self: &State): u8 {
        registered_tokens::decimals<CoinType>(&self.registered_tokens)
    }

    /// Add a new Token Bridge emitter to the registry. This method will abort
    /// if an emitter is already registered for a particular chain ID.
    ///
    /// See `register_chain` module for more info.
    public(friend) fun register_new_emitter(
        self: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        emitter_registry::add(&mut self.emitter_registry, chain, contract_address);
    }

    #[test_only]
    public fun register_new_emitter_test_only(
        self: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        register_new_emitter(self, chain, contract_address);
    }

    /// Add a new wrapped asset to the registry.
    ///
    /// See `create_wrapped` module for more info.
    public(friend) fun register_wrapped_asset<CoinType>(
        self: &mut State,
        token_meta: AssetMeta,
        supply: Supply<CoinType>,
        ctx: &mut TxContext
    ) {
        registered_tokens::add_new_wrapped(&mut self.registered_tokens,
            token_meta,
            supply,
            ctx
        )
    }

    #[test_only]
    public fun register_wrapped_asset_test_only<CoinType>(
        self: &mut State,
        token_meta: AssetMeta,
        supply: Supply<CoinType>,
        ctx: &mut TxContext
    ) {
        register_wrapped_asset(self, token_meta, supply, ctx)
    }

    /// Add a new native asset to the registry.
    ///
    /// See `attest_token` module for more info.
    public(friend) fun register_native_asset<CoinType>(
        self: &mut State,
        metadata: &CoinMetadata<CoinType>,
    ) {
        registered_tokens::add_new_native(
            &mut self.registered_tokens,
            metadata,
        );
    }

    #[test_only]
    public fun register_native_asset_test_only<CoinType>(
        self: &mut State,
        metadata: &CoinMetadata<CoinType>,
    ) {
        register_native_asset(self, metadata)
    }

}

#[test_only]
module token_bridge::bridge_state_test {
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_from_address,
        take_shared, return_shared};
    use sui::coin::{CoinMetadata};

    use wormhole::state::{State as WormholeState, chain_id};
    use wormhole::wormhole_scenario::{set_up_wormhole, three_people as people};
    use wormhole::external_address::{Self};

    use token_bridge::state::{Self, State, DeployerCap};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::coin_native_4::{Self};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::registered_tokens::E_ALREADY_REGISTERED,
        location = token_bridge::registered_tokens
    )]
    fun test_coin_type_addressing_failure_case(){
        test_coin_type_addressing_failure_case_(scenario())
    }

    public fun set_up_wormhole_core_and_token_bridges(admin: address, test: Scenario): Scenario {
        // init and share wormhole core bridge
        set_up_wormhole(&mut test, 0);

        // Call init for token bridge to get deployer cap.
        next_tx(&mut test, admin); {
            state::init_test_only(ctx(&mut test));
        };

        // Register for emitter cap and init_and_share token bridge.
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let deployer = take_from_address<DeployerCap>(&test, admin);
            state::init_and_share_state(deployer, &mut wormhole_state, ctx(&mut test));
            return_shared<WormholeState>(wormhole_state);
        };

        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            return_shared<State>(bridge_state);
        };

        return test
    }

    fun test_coin_type_addressing_failure_case_(test: Scenario) {
        let (admin, _, _) = people();

        test = set_up_wormhole_core_and_token_bridges(admin, test);

        // Test coin type addressing.
        next_tx(&mut test, admin); {
            coin_native_10::init_test_only(ctx(&mut test));
            coin_native_4::init_test_only(ctx(&mut test));
        };
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<COIN_NATIVE_10>>(&test);
            state::register_native_asset_test_only(
                &mut bridge_state,
                &coin_meta,
            );
            let (token_chain, token_address) = state::token_info<COIN_NATIVE_10>(&bridge_state);
            assert!(token_chain == chain_id(), 0);
            let expected_addr = external_address::from_any_bytes(x"01");
            assert!(token_address == expected_addr, 0);

            // aborts because trying to re-register native coin
            state::register_native_asset_test_only(
                &mut bridge_state,
                &coin_meta,
            );

            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
            return_shared<CoinMetadata<COIN_NATIVE_10>>(coin_meta);
        };
        test_scenario::end(test);
    }
}
