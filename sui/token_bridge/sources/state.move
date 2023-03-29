module token_bridge::state {
    use sui::balance::{Balance};
    use sui::clock::{Clock};
    use sui::object::{Self, ID, UID};
    use sui::package::{UpgradeCap};
    use sui::sui::{SUI};
    use sui::tx_context::{TxContext};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::set::{Self, Set};
    use wormhole::state::{State as WormholeState};

    use token_bridge::emitter_registry::{Self, EmitterRegistry};
    use token_bridge::token_registry::{Self, TokenRegistry};

    const E_UNREGISTERED_EMITTER: u64 = 0;
    const E_EMITTER_ALREADY_REGISTERED: u64 = 1;
    const E_VAA_ALREADY_CONSUMED: u64 = 2;

    friend token_bridge::attest_token;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::create_wrapped;
    friend token_bridge::register_chain;
    friend token_bridge::setup;
    friend token_bridge::transfer_tokens;
    friend token_bridge::transfer_tokens_with_payload;
    friend token_bridge::vaa;

    // TODO: For version 0.28, emit this after `commit_upgrade`.
    struct ContractUpgraded has drop, copy {
        old_contract: ID,
        new_contract: ID
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

        token_registry: TokenRegistry,

        upgrade_cap: UpgradeCap,
    }

    public(friend) fun new(
        worm_state: &WormholeState,
        upgrade_cap: UpgradeCap,
        ctx: &mut TxContext
    ): State {
        // TODO: Make sure upgrade cap belongs to this package.
        State {
            id: object::new(ctx),
            consumed_vaa_hashes: set::new(ctx),
            emitter_cap: emitter::new(worm_state, ctx),
            emitter_registry: emitter_registry::new(ctx),
            token_registry: token_registry::new(ctx),
            upgrade_cap
        }
    }

    public fun governance_module(): Bytes32 {
        // A.K.A. "TokenBridge".
        bytes32::new(
            x"000000000000000000000000000000000000000000546f6b656e427269646765"
        )
    }

    public(friend) fun publish_wormhole_message(
        self: &mut State,
        worm_state: &mut WormholeState,
        nonce: u32,
        payload: vector<u8>,
        message_fee: Balance<SUI>,
        the_clock: &Clock
    ): u64 {
        use wormhole::publish_message::{publish_message};

        publish_message(
            worm_state,
            &mut self.emitter_cap,
            nonce,
            payload,
            message_fee,
            the_clock
        )
    }

    public fun borrow_token_registry(self: &State): &TokenRegistry {
        &self.token_registry
    }

    public(friend) fun borrow_token_registry_mut(
        self: &mut State
    ): &mut TokenRegistry {
        &mut self.token_registry
    }

    #[test_only]
    public fun borrow_token_registry_mut_test_only(
        self: &mut State
    ): &mut TokenRegistry {
        borrow_token_registry_mut(self)
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

    /// Add a new Token Bridge emitter to the registry. This method will abort
    /// if an emitter is already registered for a particular chain ID.
    ///
    /// See `register_chain` module for more info.
    public(friend) fun register_new_emitter(
        self: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        emitter_registry::add(
            &mut self.emitter_registry,
            chain,
            contract_address
        );
    }

    #[test_only]
    public fun register_new_emitter_test_only(
        self: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        register_new_emitter(self, chain, contract_address);
    }
}
