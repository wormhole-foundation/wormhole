module wormhole::state {
    use std::vector::{Self};

    use sui::object::{Self, UID};
    use sui::tx_context::{Self, sender, TxContext};
    use sui::transfer::{Self};
    use sui::vec_map::{Self, VecMap};
    use sui::vec_set::{Self, VecSet};
    use sui::event::{Self};

    use wormhole::myu16::{Self as u16, U16};
    use wormhole::myu32::{Self as u32, U32};
    use wormhole::structs::{Self, GuardianSet};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::emitter::{Self};

    friend wormhole::guardian_set_upgrade;
    //friend wormhole::contract_upgrade;
    friend wormhole::wormhole;
    friend wormhole::myvaa;
    #[test_only]
    friend wormhole::vaa_test;

    struct WormholeMessage has store, copy, drop {
        sender: address,
        sequence: u64,
        nonce: u64,
        payload: vector<u8>,
        consistency_level: u8,
        timestamp: u64,
    }

    struct State has key, store {
        id: UID,

        /// chain id
        chain_id: U16,

        /// guardian chain ID
        governance_chain_id: U16,

        /// Address of governance contract on governance chain
        governance_contract: ExternalAddress,

        /// Current active guardian set index
        guardian_set_index: U32,

        /// guardian sets
        guardian_sets: VecMap<U32, GuardianSet>,

        /// Period for which a guardian set stays active after it has been replaced
        guardian_set_expiry: U32,

        /// Consumed governance actions
        consumed_governance_actions: VecSet<vector<u8>>,

        /// Capability for creating new emitters
        emitter_registry: emitter::EmitterRegistry,

        /// wormhole message fee
        message_fee: u64,
    }

    // Called automatically when module is first published. Transfers an empty State object to owner,
    // so that owner can then initialize it with specific fields. (init doesn't take any args).
    // The reason why State is not shared immediately is because we only want the sender to be able to modify it.
    //
    // Only one State ever exists, because only the init function creates a State object
    fun init(ctx: &mut TxContext) {
        transfer::transfer(State {
            id: object::new(ctx),
            chain_id: u16::from_u64(0),
            governance_chain_id: u16::from_u64(0),
            governance_contract: external_address::from_bytes(vector::empty<u8>()),
            guardian_set_index: u32::from_u64(0),
            guardian_sets: vec_map::empty<U32, GuardianSet>(),
            guardian_set_expiry: u32::from_u64(0),
            consumed_governance_actions: vec_set::empty<vector<u8>>(),
            emitter_registry: emitter::init_emitter_registry(),
            message_fee: 0,
        }, tx_context::sender(ctx));
    }

    // converts owned state object into a shared object, so that anyone can get a reference to &mut State
    // and pass it into various functions
    public entry fun init_and_share_state(
        state: State,
        chain_id: u64,
        governance_chain_id: u64,
        governance_contract: vector<u8>,
        initial_guardian: vector<u8>,
        _ctx: &mut TxContext
    ) {
        set_chain_id(&mut state, chain_id);
        set_governance_chain_id(&mut state, governance_chain_id);
        set_governance_contract(&mut state, governance_contract);
        let initial_guardian = vector[structs::create_guardian(initial_guardian)];
        store_guardian_set(&mut state, u32::from_u64(0), structs::create_guardian_set(u32::from_u64(0), initial_guardian));

        // permanently shares state
        transfer::share_object(state);
    }

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(ctx)
    }

    public(friend) entry fun publish_event(
        sequence: u64,
        nonce: u64,
        payload: vector<u8>,
        ctx: &mut TxContext
     ) {
        let now = tx_context::epoch(ctx);
        event::emit(
            WormholeMessage {
                sender: sender(ctx),
                sequence: sequence,
                nonce: nonce,
                payload: payload,
                // Sui is an instant finality chain, so we don't need
                // confirmations
                consistency_level: 0,
                timestamp: now // this is current epoch and not seconds
            }
        );
    }

    // setters

    public(friend) fun set_chain_id(state: &mut State, id: u64){
        state.chain_id = u16::from_u64(id);
    }

    #[test_only]
    public fun test_set_chain_id(state: &mut State, id: u64) {
        set_chain_id(state, id);
    }

    public(friend) fun set_governance_chain_id(state: &mut State, id: u64){
        state.governance_chain_id = u16::from_u64(id);
    }

    #[test_only]
    public fun test_set_governance_chain_id(state: &mut State, id: u64) {
        set_governance_chain_id(state, id);
    }

    public(friend) fun set_governance_action_consumed(state: &mut State, hash: vector<u8>){
        vec_set::insert<vector<u8>>(&mut state.consumed_governance_actions, hash);
    }

    public(friend) fun set_governance_contract(state: &mut State, contract: vector<u8>) {
        state.governance_contract = external_address::from_bytes(contract);
    }

    public(friend) fun update_guardian_set_index(state: &mut State, new_index: U32) {
        state.guardian_set_index = new_index;
    }

    public(friend) fun expire_guardian_set(state: &mut State, index: U32, ctx: &TxContext) {
        let expiry = state.guardian_set_expiry;
        let guardian_set = vec_map::get_mut<U32, GuardianSet>(&mut state.guardian_sets, &index);
        structs::expire_guardian_set(guardian_set, expiry, ctx);
    }

    public(friend) fun store_guardian_set(state: &mut State, index: U32, set: GuardianSet) {
        vec_map::insert<U32, GuardianSet>(&mut state.guardian_sets, index, set);
    }

    // getters

    public fun get_current_guardian_set_index(state: &State): U32 {
        return state.guardian_set_index
    }

    public fun get_guardian_set(state: &State, index: U32): GuardianSet {
        return *vec_map::get<U32, GuardianSet>(&state.guardian_sets, &index)
    }

    public fun guardian_set_is_active(state: &State, guardian_set: &GuardianSet, ctx: &TxContext): bool {
        let cur_epoch = tx_context::epoch(ctx);
        let index = structs::get_guardian_set_index(guardian_set);
        let current_index = get_current_guardian_set_index(state);
        index == current_index ||
             u32::to_u64(structs::get_guardian_set_expiry(guardian_set)) > cur_epoch
    }

    public fun get_governance_chain(state: &State): U16 {
        return state.governance_chain_id
    }

    public fun get_governance_contract(state: &State): ExternalAddress {
        return state.governance_contract
    }

    public fun get_chain_id(state: &State): U16 {
        return state.chain_id
    }

    public fun get_message_fee(state: &State): u64 {
        return state.message_fee
    }

    public(friend) fun new_emitter(state: &mut State, ctx: &mut TxContext): emitter::EmitterCapability{
        emitter::new_emitter(&mut state.emitter_registry, ctx)
    }

}

#[test_only]
module wormhole::test_state{
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_from_address, return_to_address, take_shared};
    use std::vector::{Self};

    use wormhole::state::{Self, test_init, State};
    use wormhole::myu16::{Self as u16};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    fun test_state_setters() {
        test_state_setters_(scenario())
    }

    fun test_state_setters_(test: Scenario) {
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            test_init(ctx(&mut test));
        };

        // test State setter and getter functions
        next_tx(&mut test, admin); {
            let state = take_from_address<State>(&test, admin);

            // test set chain id
            state::test_set_chain_id(&mut state, 5);
            assert!(state::get_chain_id(&state) == u16::from_u64(5), 0);

            // test set governance chain id
            state::test_set_governance_chain_id(&mut state, 100);
            assert!(state::get_governance_chain(&state) == u16::from_u64(100), 0);

            return_to_address<State>(admin, state);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_init_and_share_state() {
        test_init_and_share_state_(scenario())
    }

    fun test_init_and_share_state_(test: Scenario) {
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            let state = take_from_address<State>(&mut test, admin);
            // initialize state with desired parameters and initial guardian address
            state::init_and_share_state(state, 0, 0, vector::empty<u8>(), x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe", ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            // confirm that state is indeed a shared object, and mutate it
            let state = take_shared<State>(&mut test);
            //let mut_ref = test_scenario::borrow_mut(&mut state);
            let mut_ref = &mut state;
            state::test_set_chain_id(mut_ref, 9);
            assert!(state::get_chain_id(mut_ref)==u16::from_u64(9), 0);
            test_scenario::return_shared(state);
        };
        test_scenario::end(test);
    }
}
