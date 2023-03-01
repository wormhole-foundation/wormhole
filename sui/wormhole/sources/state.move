module wormhole::state {
    use std::vector::{Self};
    use sui::coin::{Coin};
    use sui::object::{Self, UID};
    use sui::sui::{SUI};
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};

    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::cursor::{Self};
    use wormhole::emitter::{Self, EmitterCapability, EmitterRegistry};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::fee_collector::{Self, FeeCollector};
    use wormhole::guardian::{Self};
    use wormhole::guardian_set::{Self, GuardianSet};
    use wormhole::set::{Self, Set};

    friend wormhole::publish_message;
    friend wormhole::set_fee;
    friend wormhole::setup;
    friend wormhole::transfer_fee;
    friend wormhole::update_guardian_set;
    friend wormhole::vaa;

    const E_ZERO_GUARDIANS: u64 = 0;
    const E_VAA_ALREADY_CONSUMED: u64 = 1;

    /// Sui's chain ID is hard-coded to one value.
    const CHAIN_ID: u16 = 21;

    struct State has key, store {
        id: UID,

        /// Governance chain ID.
        governance_chain: u16,

        /// Governance contract address.
        governance_contract: ExternalAddress,

        /// Current active guardian set index.
        guardian_set_index: u32,

        /// All guardian sets (including expired ones).
        guardian_sets: Table<u32, GuardianSet>,

        /// Period for which a guardian set stays active after it has been
        /// replaced.
        ///
        /// Currently in terms of Sui epochs until we have access to a clock
        /// with unix timestamp.
        guardian_set_epochs_to_live: u32,

        /// Consumed VAA hashes to protect against replay. VAAs relevant to
        /// Wormhole are just governance VAAs.
        consumed_vaa_hashes: Set<Bytes32>,

        /// Registry for new emitter caps (`EmitterCapability`).
        emitter_registry: EmitterRegistry,

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

        // First guardian set index is zero. New guardian sets must increment
        // from the last recorded index.
        let guardian_set_index = 0;

        let governance_contract =
            external_address::from_nonzero_bytes(
                governance_contract
            );
        let state = State {
            id: object::new(ctx),
            governance_chain,
            governance_contract,
            guardian_set_index,
            guardian_sets: table::new(ctx),
            guardian_set_epochs_to_live,
            consumed_vaa_hashes: set::new(ctx),
            emitter_registry: emitter::new_registry(),
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

    public fun governance_module(): Bytes32 {
        // A.K.A. "Core".
        bytes32::new(
            x"00000000000000000000000000000000000000000000000000000000436f7265"
        )
    }

    public fun governance_chain(self: &State): u16 {
        self.governance_chain
    }

    public fun governance_contract(self: &State): ExternalAddress {
        self.governance_contract
    }

    public fun guardian_set_index(self: &State): u32 {
        self.guardian_set_index
    }

    public fun guardian_set_epochs_to_live(self: &State): u32 {
        self.guardian_set_epochs_to_live
    }

    public fun message_fee(self: &State): u64 {
        return fee_collector::fee_amount(&self.fee_collector)
    }

    public fun deposit_fee(self: &mut State, coin: Coin<SUI>) {
        fee_collector::deposit(&mut self.fee_collector, coin);
    }

    public(friend) fun withdraw_fee(
        self: &mut State,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<SUI> {
        fee_collector::withdraw(&mut self.fee_collector, amount, ctx)
    }

    public fun fees_collected(self: &State): u64 {
        fee_collector::balance_value(&self.fee_collector)
    }

    public(friend) fun consume_vaa_hash(self: &mut State, vaa_hash: Bytes32) {
        let consumed = &mut self.consumed_vaa_hashes;
        assert!(!set::contains(consumed, vaa_hash), E_VAA_ALREADY_CONSUMED);
        set::add(consumed, vaa_hash);
    }

    public(friend) fun expire_guardian_set(self: &mut State, ctx: &TxContext) {
        let expiring =
            table::borrow_mut(&mut self.guardian_sets, self.guardian_set_index);
        guardian_set::set_expiration(
            expiring,
            self.guardian_set_epochs_to_live,
            ctx
        );
    }

    public(friend) fun store_guardian_set(
        self: &mut State,
        new_guardian_set: GuardianSet
    ) {
        self.guardian_set_index = guardian_set::index(&new_guardian_set);
        table::add(
            &mut self.guardian_sets,
            self.guardian_set_index,
            new_guardian_set
        );
    }

    public(friend) fun set_message_fee(self: &mut State, amount: u64) {
        fee_collector::change_fee(&mut self.fee_collector, amount);
    }

    public fun guardian_set_at(self: &State, index: u32): &GuardianSet {
        table::borrow(&self.guardian_sets, index)
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
        emitter::new_emitter(&mut self.emitter_registry, ctx)
    }

    public(friend) fun use_emitter_sequence(
        emitter_cap: &mut EmitterCapability
    ): u64 {
        emitter::use_sequence(emitter_cap)
    }
}
