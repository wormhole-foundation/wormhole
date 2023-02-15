module wormhole::state {
    use std::vector::{Self};

    use sui::object::{Self, UID};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer::{Self};
    use sui::vec_map::{Self, VecMap};
    use sui::event::{Self};
    use sui::coin::{Coin};
    use sui::sui::{SUI};

    use wormhole::fee_collector::{Self};
    use wormhole::set::{Self, Set};
    use wormhole::structs::{Self, create_guardian, Guardian, GuardianSet};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::emitter::{Self};

    friend wormhole::update_guardian_set;
    friend wormhole::wormhole;
    friend wormhole::myvaa;
    #[test_only]
    friend wormhole::vaa_test;

    const CHAIN_ID: u16 = 21;

    struct DeployerCapability has key, store {id: UID}

    struct WormholeMessage has store, copy, drop {
        sender: u64,
        sequence: u64,
        nonce: u32,
        payload: vector<u8>,
        consistency_level: u8
    }

    struct FeeCustody<phantom CoinType> has key, store {
        id: UID,
        custody: Coin<CoinType>
    }

    struct State has key, store {
        id: UID,

        /// guardian chain ID
        governance_chain_id: u16,

        /// Address of governance contract on governance chain
        governance_contract: ExternalAddress,

        /// Current active guardian set index
        guardian_set_index: u32,

        /// guardian sets
        guardian_sets: VecMap<u32, GuardianSet>,

        /// Period for which a guardian set stays active after it has been replaced
        guardian_set_expiry: u32,

        /// Consumed governance actions
        consumed_governance_actions: Set<vector<u8>>,

        /// Capability for creating new emitters
        emitter_registry: emitter::EmitterRegistry,

        /// wormhole message fee
        message_fee: u64,
    }

    /// Called automatically when module is first published. Transfers a deployer cap to sender.
    fun init(ctx: &mut TxContext) {
        transfer::transfer(DeployerCapability{id: object::new(ctx)}, tx_context::sender(ctx));
    }

    // creates a shared state object, so that anyone can get a reference to &mut State
    // and pass it into various functions
    public entry fun init_and_share_state(
        deployer: DeployerCapability,
        governance_chain_id: u16,
        governance_contract: vector<u8>,
        initial_guardians: vector<vector<u8>>,
        message_fee: u64,
        ctx: &mut TxContext
    ) {
        let DeployerCapability{ id } = deployer;
        object::delete(id);
        let state = State {
            id: object::new(ctx),
            governance_chain_id,
            governance_contract: external_address::from_bytes(
                governance_contract
            ),
            guardian_set_index: 0,
            guardian_sets: vec_map::empty<u32, GuardianSet>(),
            guardian_set_expiry: 2, // TODO - what is the right #epochs to set this to?
            consumed_governance_actions: set::new(ctx),
            emitter_registry: emitter::init_emitter_registry(),
            message_fee,
        };

        let guardians = vector::empty<Guardian>();
        vector::reverse(&mut initial_guardians);
        while (!vector::is_empty(&initial_guardians)) {
            vector::push_back(
                &mut guardians,
                create_guardian(vector::pop_back(&mut initial_guardians))
            );
        };

        // the initial guardian set with index 0
        let initial_index = 0;
        store_guardian_set(
            &mut state,
            initial_index,
            structs::create_guardian_set(initial_index, guardians)
        );

        // add wormhole fee collector
        fee_collector::new(&mut state.id, ctx);

        // permanently shares state
        transfer::share_object<State>(state);
    }

    public fun deposit_fee_coins(state: &mut State, coin: Coin<SUI>) {
        fee_collector::deposit(&mut state.id, coin);
    }

    // TODO - later on, can perform contract upgrade and add a governance-gated withdraw function to
    //        extract fee coins from the store

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        transfer::transfer(DeployerCapability{id: object::new(ctx)}, tx_context::sender(ctx));
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

    public(friend) fun set_governance_chain_id(
        state: &mut State,
        chain_id: u16
    ) {
        state.governance_chain_id = chain_id;
    }

    #[test_only]
    public fun test_set_governance_chain_id(state: &mut State, chain_id: u16) {
        set_governance_chain_id(state, chain_id);
    }

    public(friend) fun set_governance_action_consumed(state: &mut State, hash: vector<u8>){
        set::add<vector<u8>>(&mut state.consumed_governance_actions, hash);
    }

    public(friend) fun set_governance_contract(state: &mut State, contract: vector<u8>) {
        state.governance_contract = external_address::from_bytes(contract);
    }

    public(friend) fun update_guardian_set_index(state: &mut State, new_index: u32) {
        state.guardian_set_index = new_index;
    }

    public(friend) fun expire_guardian_set(state: &mut State, index: u32, ctx: &TxContext) {
        let expiry = state.guardian_set_expiry;
        let guardian_set = vec_map::get_mut<u32, GuardianSet>(&mut state.guardian_sets, &index);
        structs::expire_guardian_set(guardian_set, expiry, ctx);
    }

    public(friend) fun store_guardian_set(state: &mut State, index: u32, set: GuardianSet) {
        vec_map::insert<u32, GuardianSet>(&mut state.guardian_sets, index, set);
    }

    // getters

    public fun get_current_guardian_set_index(state: &State): u32 {
        return state.guardian_set_index
    }

    public fun get_guardian_set(state: &State, index: u32): GuardianSet {
        return *vec_map::get<u32, GuardianSet>(&state.guardian_sets, &index)
    }

    public fun guardian_set_is_active(
        state: &State,
        guardian_set: &GuardianSet,
        ctx: &TxContext
    ): bool {
        let cur_epoch = (tx_context::epoch(ctx) as u32);
        let index = structs::get_guardian_set_index(guardian_set);
        let current_index = get_current_guardian_set_index(state);
        index == current_index ||
             structs::get_guardian_set_expiry(guardian_set) > cur_epoch
    }

    public fun get_governance_chain(state: &State): u16 {
        return state.governance_chain_id
    }

    public fun get_governance_contract(state: &State): ExternalAddress {
        return state.governance_contract
    }

    public fun chain_id(): u16 {
        CHAIN_ID
    }

    public fun get_message_fee(state: &State): u64 {
        return state.message_fee
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

    use wormhole::state::{Self, test_init, DeployerCapability};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    public fun init_wormhole_state(test: Scenario, admin: address, message_fee: u64): Scenario {
        next_tx(&mut test, admin); {
            test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin); {
            let deployer = take_from_address<DeployerCapability>(&test, admin);
            state::init_and_share_state(
                deployer,
                1, // governance chain
                x"0000000000000000000000000000000000000000000000000000000000000004", // governance_contract
                vector[x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"], // initial_guardian(s)
                message_fee, // message fee = 0
                ctx(&mut test));
        };
        return test
    }
}
