module wormhole::state {
    use std::vector::{Self};
    use sui::dynamic_field::{Self};
    use sui::object::{Self, UID};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer::{Self};
    use sui::vec_map::{Self, VecMap};
    use sui::event::{Self};
    use sui::coin::{Coin};
    use sui::sui::{SUI};

    use wormhole::fee_collector::{Self};
    use wormhole::guardian::{Self, Guardian};
    use wormhole::set::{Self, Set};
    use wormhole::guardian_set::{Self, GuardianSet};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::emitter::{Self};

    friend wormhole::update_guardian_set;
    friend wormhole::wormhole;
    friend wormhole::myvaa;
    #[test_only]
    friend wormhole::vaa_test;

    /// Sui's chain ID is hard-coded to one value.
    const CHAIN_ID: u16 = 21;

    /// Dynamic field key for `FeeCollector`
    const FIELD_FEE_COLLECTOR: vector<u8> = b"fee_collector";

    /// Capability created at `init`, which will be destroyed once
    /// `init_and_share_state` is called. This ensures only the deployer can
    /// create the shared `State`.
    struct DeployerCapability has key, store {
        id: UID
    }

    struct WormholeMessage has store, copy, drop {
        sender: u64,
        sequence: u64,
        nonce: u32,
        payload: vector<u8>,
        consistency_level: u8
    }

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

        /// Wormhole message fee.
        message_fee: u64,
    }

    /// Called automatically when module is first published. Transfers deployer
    /// cap to sender.
    fun init(ctx: &mut TxContext) {
        let cap = DeployerCapability{ id: object::new(ctx) };
        transfer::transfer(cap, tx_context::sender(ctx));
    }

    // creates a shared state object, so that anyone can get a reference to &mut State
    // and pass it into various functions
    public entry fun init_and_share_state(
        deployer: DeployerCapability,
        governance_chain: u16,
        governance_contract: vector<u8>,
        initial_guardians: vector<vector<u8>>,
        guardian_set_epochs_to_live: u32,
        message_fee: u64,
        ctx: &mut TxContext
    ) {
        let DeployerCapability{ id } = deployer;
        object::delete(id);

        let state = State {
            id: object::new(ctx),
            governance_chain,
            governance_contract: external_address::from_nonzero_bytes(
                governance_contract
            ),
            guardian_set_index: 0,
            guardian_sets: vec_map::empty<u32, GuardianSet>(),
            guardian_set_epochs_to_live,
            consumed_governance_actions: set::new(ctx),
            emitter_registry: emitter::init_emitter_registry(),
            message_fee,
        };

        let guardians = vector::empty<Guardian>();
        vector::reverse(&mut initial_guardians);
        while (!vector::is_empty(&initial_guardians)) {
            vector::push_back(
                &mut guardians,
                guardian::new(vector::pop_back(&mut initial_guardians))
            );
        };

        // the initial guardian set with index 0
        store_guardian_set(&mut state, guardian_set::new(0, guardians));

        // add wormhole fee collector
        dynamic_field::add(
            &mut state.id,
            FIELD_FEE_COLLECTOR,
            fee_collector::new(message_fee, ctx)
        );

        // permanently shares state
        transfer::share_object<State>(state);
    }

    public fun chain_id(): u16 {
        CHAIN_ID
    }

    public fun governance_chain(self: &State): u16 {
        self.governance_chain
    }

    #[test_only]
    public fun set_governance_chain(self: &mut State, chain: u16) {
        self.governance_chain = chain;
    }

    public fun governance_contract(self: &State): ExternalAddress {
        self.governance_contract
    }

    #[test_only]
    public fun set_governance_contract(self: &mut State, contract: vector<u8>) {
        self.governance_contract = external_address::from_bytes(contract);
    }

    public fun guardian_set_index(self: &State): u32 {
        self.guardian_set_index
    }

    public fun message_fee(self: &State): u64 {
        return self.message_fee
    }

    public fun deposit_fee(self: &mut State, coin: Coin<SUI>) {
        fee_collector::deposit(
            dynamic_field::borrow_mut(&mut self.id, FIELD_FEE_COLLECTOR),
            coin
        );
    }

    // TODO - later on, can perform contract upgrade and add a governance-gated withdraw function to
    //        extract fee coins from the store

    #[test_only]
    public fun init_test_only(ctx: &mut TxContext) {
        init(ctx)
    }

    public(friend) entry fun publish_event(
        sender: u64,
        sequence: u64,
        nonce: u32,
        payload: vector<u8>
     ) {
        event::emit(
            WormholeMessage {
                sender,
                sequence,
                nonce,
                payload: payload,
                // Sui is an instant finality chain, so we don't need
                // confirmations
                consistency_level: 0,
            }
        );
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

    public fun guardian_set_at(self: &State, index: u32): GuardianSet {
        return *vec_map::get<u32, GuardianSet>(&self.guardian_sets, &index)
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

    public(friend) fun new_emitter(
        state: &mut State,
        ctx: &mut TxContext
    ): emitter::EmitterCapability{
        emitter::new_emitter(&mut state.emitter_registry, ctx)
    }

}

#[test_only]
module wormhole::test_state{
    use sui::test_scenario::{
        Self,
        Scenario,
        next_tx,
        ctx,
        take_from_address,
    };

    use wormhole::state::{Self, init_test_only, DeployerCapability};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    public fun init_wormhole_state(test: Scenario, admin: address, message_fee: u64): Scenario {
        next_tx(&mut test, admin); {
            init_test_only(ctx(&mut test));
        };
        next_tx(&mut test, admin); {
            let deployer = take_from_address<DeployerCapability>(&test, admin);
            state::init_and_share_state(
                deployer,
                1, // governance chain
                x"0000000000000000000000000000000000000000000000000000000000000004", // governance_contract
                vector[x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"], // initial_guardian(s)
                2, // guardian_set_epochs_to_live
                message_fee, // message fee
                ctx(&mut test));
        };
        return test
    }
}
