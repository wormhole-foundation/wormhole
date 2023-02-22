module wormhole::state {
    use std::vector::{Self};
    use sui::coin::{Coin};
    use sui::object::{Self, UID};
    use sui::sui::{SUI};
    use sui::tx_context::{TxContext};
    use sui::vec_map::{Self, VecMap};

    use wormhole::cursor::{Self};
    use wormhole::emitter::{Self, EmitterCapability};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::fee_collector::{Self, FeeCollector};
    use wormhole::guardian::{Self};
    use wormhole::guardian_set::{Self, GuardianSet};
    use wormhole::set::{Self, Set};

    friend wormhole::update_guardian_set;
    friend wormhole::publish_message;
    friend wormhole::myvaa;
    friend wormhole::setup;
    #[test_only]
    friend wormhole::vaa_test;

    const E_ZERO_GUARDIANS: u64 = 0;

    /// Sui's chain ID is hard-coded to one value.
    const CHAIN_ID: u16 = 21;

    /// Dynamic field key for `FeeCollector`
    //const FIELD_FEE_COLLECTOR: vector<u8> = b"fee_collector";

    struct State has key, store {
        id: UID,

        /// Governance chain ID.
        governance_chain: u16,

        /// Governance contract address.
        governance_contract: ExternalAddress,

        /// Current active guardian set index.
        guardian_set_index: u32,

        /// All guardian sets (including expired ones).
        guardian_sets: VecMap<u32, GuardianSet>,

        /// Period for which a guardian set stays active after it has been
        /// replaced.
        ///
        /// Currently in terms of Sui epochs until we have access to a clock
        /// with unix timestamp.
        guardian_set_epochs_to_live: u32,

        /// Consumed governance VAAs.
        consumed_governance_actions: Set<vector<u8>>,

        /// Capability for creating new emitters
        emitter_registry: emitter::EmitterRegistry,

        /// Wormhole fee collector.
        fee_collector: FeeCollector,
    }

    public(friend) fun new(
        governance_chain: u16,
        governance_contract: vector<u8>,
        initial_guardians: vector<vector<u8>>,
        guardian_set_epochs_to_live: u32,
        message_fee: u64,
        ctx: &mut TxContext
    ): State {
        assert!(vector::length(&initial_guardians) > 0, E_ZERO_GUARDIANS);

        let governance_contract =
            external_address::from_nonzero_bytes(
                governance_contract
            );
        let guardian_set_index = 0;
        let state = State {
            id: object::new(ctx),
            governance_chain,
            governance_contract,
            guardian_set_index,
            guardian_sets: vec_map::empty(),
            guardian_set_epochs_to_live,
            consumed_governance_actions: set::new(ctx),
            emitter_registry: emitter::init_emitter_registry(),
            fee_collector: fee_collector::new(message_fee)
        };

        let guardians = {
            let out = vector::empty();
            let cur = cursor::new(initial_guardians);
            while (!cursor::is_empty(&cur)) {
                vector::push_back(
                    &mut out,
                    guardian::new(cursor::poke(&mut cur))
                );
            };
            cursor::destroy_empty(cur);
            out
        };

        // the initial guardian set with index 0
        store_guardian_set(
            &mut state,
            guardian_set::new(guardian_set_index, guardians)
        );

        state
    }

    public fun chain_id(): u16 {
        CHAIN_ID
    }

    public fun governance_chain(self: &State): u16 {
        self.governance_chain
    }

    #[test_only]
    // TODO: possibly remove
    public fun set_governance_chain(self: &mut State, chain: u16) {
        self.governance_chain = chain;
    }

    public fun governance_contract(self: &State): ExternalAddress {
        self.governance_contract
    }

    #[test_only]
    // TODO: possibly remove
    public fun set_governance_contract(self: &mut State, contract: vector<u8>) {
        self.governance_contract = external_address::from_bytes(contract);
    }

    public fun guardian_set_index(self: &State): u32 {
        self.guardian_set_index
    }

    public fun guardian_set_epochs_to_live(self: &State): u32 {
        self.guardian_set_epochs_to_live
    }

    public fun message_fee(self: &State): u64 {
        return fee_collector::amount(&self.fee_collector)
    }

    public fun deposit_fee(self: &mut State, coin: Coin<SUI>) {
        fee_collector::deposit(&mut self.fee_collector, coin);
    }

    public(friend) fun set_governance_action_consumed(self: &mut State, hash: vector<u8>){
        set::add<vector<u8>>(&mut self.consumed_governance_actions, hash);
    }

    public(friend) fun update_guardian_set_index(self: &mut State, new_index: u32) {
        self.guardian_set_index = new_index;
    }

    public(friend) fun expire_guardian_set(self: &mut State, ctx: &TxContext) {
        let set =
            vec_map::get_mut<u32, GuardianSet>(
                &mut self.guardian_sets,
                &self.guardian_set_index
            );
        guardian_set::set_expiration(
            set,
            self.guardian_set_epochs_to_live,
            ctx
        );
    }

    public(friend) fun store_guardian_set(self: &mut State, set: GuardianSet) {
        vec_map::insert<u32, GuardianSet>(
            &mut self.guardian_sets, guardian_set::index(&set),
            set
        );
    }

    public fun guardian_set_at(self: &State, index: &u32): &GuardianSet {
        vec_map::get(&self.guardian_sets, index)
    }

    public fun is_guardian_set_active(
        self: &State,
        set: &GuardianSet,
        ctx: &TxContext
    ): bool {
        (
            self.guardian_set_index == guardian_set::index(set) ||
            guardian_set::is_active(set, ctx)
        )
    }

    public fun new_emitter(
        self: &mut State,
        ctx: &mut TxContext
    ): EmitterCapability{
        emitter::new_emitter_cap(&mut self.emitter_registry, ctx)
    }

}
