module token_bridge::state {
    use std::ascii::{Self};
    use sui::coin::{Self, Coin, CoinMetadata, TreasuryCap};
    use sui::object::{Self, UID};
    use sui::sui::{SUI};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};
    use wormhole::emitter::{EmitterCapability};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::set::{Self, Set};
    use wormhole::state::{Self as wormhole_state, State as WormholeState};
    use wormhole::wormhole::{Self};

    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::registered_emitters::{Self};
    use token_bridge::registered_tokens::{Self, RegisteredTokens};
    use token_bridge::string32::{Self};
    use token_bridge::token_info::{TokenInfo};

    const E_UNREGISTERED_EMITTER: u64 = 0;
    const E_EMITTER_ALREADY_REGISTERED: u64 = 1;

    friend token_bridge::attest_token;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::create_wrapped;
    friend token_bridge::register_chain;
    friend token_bridge::transfer_tokens;
    friend token_bridge::transfer_tokens_with_payload;
    friend token_bridge::vaa;

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

    /// Capability for creating a bridge state object, granted to sender when
    /// this module is deployed.
    struct DeployerCapability has key, store {
        id: UID
    }

    /// Treasury caps, token stores, consumed VAAs, registered emitters, etc.
    /// are stored as dynamic fields of bridge state.
    struct State has key, store {
        id: UID,

        // Set of consumed VAA hashes.
        consumed_vaas: Set<vector<u8>>,

        // Token bridge owned emitter capability.
        emitter_cap: EmitterCapability,

        registered_tokens: RegisteredTokens,
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
    public fun init_test_only(ctx: &mut TxContext) {
        init(ctx)
    }

    /// Converts owned state object into a shared object, so that anyone can get
    /// a reference to &mut State and pass it into various functions.
    public entry fun init_and_share_state(
        deployer: DeployerCapability,
        worm_state: &mut WormholeState,
        ctx: &mut TxContext
    ) {
        let DeployerCapability{ id } = deployer;
        object::delete(id);

        let state = State {
            id: object::new(ctx),
            consumed_vaas: set::new(ctx),
            emitter_cap: wormhole::register_emitter(worm_state, ctx),
            registered_tokens: registered_tokens::new(ctx)
        };

        registered_emitters::new(&mut state.id, ctx);

        // Permanently shares state.
        transfer::share_object(state);
    }

    public(friend) fun deposit<CoinType>(
        self: &mut State,
        coin: Coin<CoinType>,
    ) {
        registered_tokens::deposit(&mut self.registered_tokens, coin)
    }

    #[test_only]
    /// Exposing method so an integrator can test redeeming native tokens.
    public fun deposit_test_only<CoinType>(
        self: &mut State,
        coin: Coin<CoinType>
    ) {
        deposit(self, coin)
    }

    public(friend) fun withdraw<CoinType>(
        self: &mut State,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        registered_tokens::withdraw(&mut self.registered_tokens, amount, ctx)
    }

    #[test_only]
    /// Exposing method so an integrator can test sending native tokens.
    public fun withdraw_test_only<CoinType>(
        self: &mut State,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        withdraw(self, amount, ctx)
    }

    public(friend) fun burn<CoinType>(
        self: &mut State,
        coin: Coin<CoinType>,
    ): u64 {
        registered_tokens::burn(&mut self.registered_tokens, coin)
    }

    #[test_only]
    /// Exposing method so an integrator can test redeeming wrapped tokens.
    public fun burn_test_only<CoinType>(
        self: &mut State,
        coin: Coin<CoinType>
    ): u64 {
        burn(self, coin)
    }

    public(friend) fun mint<CoinType>(
        self: &mut State,
        amount: u64,
        ctx: &mut TxContext,
    ): Coin<CoinType> {
        registered_tokens::mint(&mut self.registered_tokens, amount, ctx)
    }

    #[test_only]
    /// Exposing method so an integrator can test sending wrapped tokens.
    public fun mint_test_only<CoinType>(
        self: &mut State,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        mint(self, amount, ctx)
    }

    /// We only examine the balance of native assets, because the token
    /// bridge does not custody wrapped assets (only mints and burns them).
    public fun balance<CoinType>(self: &State): u64 {
        registered_tokens::balance<CoinType>(&self.registered_tokens)
    }

    public(friend) fun publish_wormhole_message(
        self: &mut State,
        worm_state: &mut WormholeState,
        nonce: u32,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
    ): u64 {
        wormhole::publish_message(
            &mut self.emitter_cap,
            worm_state,
            nonce,
            payload,
            message_fee,
        )
    }

    public fun vaa_is_consumed(state: &State, hash: vector<u8>): bool {
        set::contains(&state.consumed_vaas, hash)
    }

    public fun registered_emitter(
        state: &State,
        chain: u16
    ): ExternalAddress {
        assert!(
            registered_emitters::has(&state.id, chain),
            E_UNREGISTERED_EMITTER
        );
        registered_emitters::external_address(&state.id, chain)
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

    /// Returns the origin information for a CoinType.
    public fun token_info<CoinType>(self: &State): TokenInfo<CoinType> {
        registered_tokens::to_token_info<CoinType>(&self.registered_tokens)
    }

    public fun coin_decimals<CoinType>(self: &State): u8 {
        registered_tokens::decimals<CoinType>(&self.registered_tokens)
    }

    /// This function returns an immutable reference to the treasury cap
    /// for a wrapped coin that the token bridge manages. Note that there
    /// is no danger of the returned reference being used to mint coins
    /// outside of the bridge mint/burn mechanism, because a mutable reference
    /// to the TreasuryCap is required for mint/burn.
    ///
    /// This function is only used in create_wrapped.move to update coin
    /// metadata (only an immutable reference is needed).
    public(friend) fun treasury_cap<CoinType>(self: &State): &TreasuryCap<CoinType> {
        registered_tokens::treasury_cap<CoinType>(&self.registered_tokens)
    }

    public(friend) fun register_emitter(
        self: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        assert!(
            !registered_emitters::has(&self.id, chain),
            E_EMITTER_ALREADY_REGISTERED
        );
        registered_emitters::add(&mut self.id, chain, contract_address);
    }

    #[test_only]
    public fun register_emitter_test_only(
        self: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        register_emitter(self, chain, contract_address);
    }

    /// dynamic ops

    public(friend) fun store_consumed_vaa(
        bridge_state: &mut State,
        vaa: vector<u8>)
    {
        set::add(&mut bridge_state.consumed_vaas, vaa);
    }

    public(friend) fun register_wrapped_asset<CoinType>(
        self: &mut State,
        token_chain: u16,
        token_address: ExternalAddress,
        treasury_cap: TreasuryCap<CoinType>,
        decimals: u8,
    ) {
        registered_tokens::add_new_wrapped(&mut self.registered_tokens,
            token_chain,
            token_address,
            treasury_cap,
            decimals,
        )
    }

    public(friend) fun register_native_asset<CoinType>(
        self: &mut State,
        coin_metadata: &CoinMetadata<CoinType>,
    ): AssetMeta {
        let decimals = coin::get_decimals(coin_metadata);

        registered_tokens::add_new_native<CoinType>(
            &mut self.registered_tokens,
            decimals,
        );

        asset_meta::new(
            wormhole_state::chain_id(),
            registered_tokens::token_address<CoinType>(&self.registered_tokens),
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
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_from_address,
        take_shared, return_shared};
    use sui::coin::{CoinMetadata};

    use wormhole::state::{State as WormholeState};
    use wormhole::test_state::{init_wormhole_state};
    use wormhole::external_address::{Self};

    use token_bridge::state::{Self, State, DeployerCapability};
    use token_bridge::native_coin_10_decimals::{Self, NATIVE_COIN_10_DECIMALS};
    use token_bridge::native_coin_4_decimals::{Self, NATIVE_COIN_4_DECIMALS};
    use token_bridge::token_info::{Self};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    const ETHEREUM_TOKEN_REG: vector<u8> =
        x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    #[test]
    fun test_state_setters() {
        test_state_setters_(scenario())
    }

    #[test]
    fun test_coin_type_addressing(){
        test_coin_type_addressing_(scenario())
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::registered_tokens::E_ALREADY_REGISTERED,
        location = token_bridge::registered_tokens
    )]
    fun test_coin_type_addressing_failure_case(){
        test_coin_type_addressing_failure_case_(scenario())
    }

    public fun set_up_wormhole_core_and_token_bridges(admin: address, test: Scenario): Scenario {
        // Init and share wormhole core bridge.
        test =  init_wormhole_state(test, admin, 0);

        // Call init for token bridge to get deployer cap.
        next_tx(&mut test, admin); {
            state::init_test_only(ctx(&mut test));
        };

        // Register for emitter cap and init_and_share token bridge.
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let deployer = take_from_address<DeployerCapability>(&test, admin);
            state::init_and_share_state(
                deployer,
                &mut wormhole_state,
                ctx(&mut test)
            );
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

        // Test State setter and getter functions.
        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);

            // Test store consumed vaa.
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

        // Test coin type addressing.
        next_tx(&mut test, admin); {
            native_coin_4_decimals::test_init(ctx(&mut test));
            native_coin_10_decimals::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            let coin_meta =
                take_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(&test);

            state::register_native_asset<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &coin_meta,
            );
            let info = state::token_info<NATIVE_COIN_10_DECIMALS>(&bridge_state);
            let expected_addr = external_address::from_bytes(x"01");
            assert!(token_info::addr(&info) == expected_addr, 0);

            let coin_meta_v2 =
                take_shared<CoinMetadata<NATIVE_COIN_4_DECIMALS>>(&test);
            state::register_native_asset<NATIVE_COIN_4_DECIMALS>(
                &mut bridge_state,
                &coin_meta_v2,
            );
            let info =
                state::token_info<NATIVE_COIN_4_DECIMALS>(&bridge_state);
            let expected_addr = external_address::from_bytes(x"02");
            assert!(token_info::addr(&info) == expected_addr, 0);

            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
            return_shared<CoinMetadata<NATIVE_COIN_4_DECIMALS>>(coin_meta_v2);
            return_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(coin_meta);
        };
        test_scenario::end(test);
    }


    fun test_coin_type_addressing_failure_case_(test: Scenario) {
        let (admin, _, _) = people();

        test = set_up_wormhole_core_and_token_bridges(admin, test);

        // Test coin type addressing.
        next_tx(&mut test, admin); {
            native_coin_10_decimals::test_init(ctx(&mut test));
            native_coin_4_decimals::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(&test);
            state::register_native_asset<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &coin_meta,
            );
            let info = state::token_info<NATIVE_COIN_10_DECIMALS>(&bridge_state);
            let expected_addr = external_address::from_bytes(x"01");
            assert!(token_info::addr(&info) == expected_addr, 0);

            // aborts because trying to re-register native coin
            state::register_native_asset<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &coin_meta,
            );

            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
            return_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(coin_meta);
        };
        test_scenario::end(test);
    }
}
