module wormhole::state {
    use std::vector::{Self};

    use sui::object::{Self, UID};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer::{Self};
    use sui::vec_map::{Self, VecMap};
    use sui::event::{Self};
    use sui::coin::{Self, Coin};
    use sui::sui::SUI;

    use wormhole::myu16::{Self as u16, U16};
    use wormhole::myu32::{Self as u32, U32};
    use wormhole::dynamic_set::{Self};
    use wormhole::set::{Self, Set};
    use wormhole::structs::{Self, create_guardian, Guardian, GuardianSet};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::emitter::{Self};

    friend wormhole::guardian_set_upgrade;
    //friend wormhole::contract_upgrade;
    friend wormhole::wormhole;
    friend wormhole::myvaa;
    #[test_only]
    friend wormhole::vaa_test;

    struct DeployerCapability has key, store {id: UID}

    struct WormholeMessage has store, copy, drop {
        sender: u64,
        sequence: u64,
        nonce: u64,
        payload: vector<u8>,
        consistency_level: u8
    }

    struct FeeCustody<phantom CoinType> has key, store {
        id: UID,
        custody: Coin<CoinType>
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
        chain_id: u64,
        governance_chain_id: u64,
        governance_contract: vector<u8>,
        initial_guardians: vector<vector<u8>>,
        message_fee: u64,
        ctx: &mut TxContext
    ) {
        let DeployerCapability{id} = deployer;
        object::delete(id);
        let state = State {
            id: object::new(ctx),
            chain_id: u16::from_u64(chain_id),
            governance_chain_id: u16::from_u64(governance_chain_id),
            governance_contract: external_address::from_bytes(governance_contract),
            guardian_set_index: u32::from_u64(0),
            guardian_sets: vec_map::empty<U32, GuardianSet>(),
            guardian_set_expiry: u32::from_u64(2), // TODO - what is the right #epochs to set this to?
            consumed_governance_actions: set::new(ctx),
            emitter_registry: emitter::init_emitter_registry(),
            message_fee: message_fee,
        };

        let guardians = vector::empty<Guardian>();
        vector::reverse(&mut initial_guardians);
        while (!vector::is_empty(&initial_guardians)) {
            vector::push_back(&mut guardians, create_guardian(vector::pop_back(&mut initial_guardians)));
        };

        // the initial guardian set with index 0
        let initial_index = u32::from_u64(0);
        store_guardian_set(&mut state, initial_index, structs::create_guardian_set(initial_index, guardians));

        // add wormhole fee store FeeCustody<SUI> as a dynamic child of state
        dynamic_set::add<FeeCustody<SUI>>(&mut state.id, FeeCustody<SUI>{id: object::new(ctx), custody: coin::zero<SUI>(ctx)});

        // permanently shares state
        transfer::share_object<State>(state);
    }

    public fun deposit_fee_coins<CoinType>(state: &mut State, coin: Coin<CoinType>){
        let fee_custody = dynamic_set::borrow_mut<FeeCustody<CoinType>>(&mut state.id);
        coin::join<CoinType>(&mut fee_custody.custody, coin);
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
        nonce: u64,
        payload: vector<u8>
     ) {
        event::emit(
            WormholeMessage {
                sender: sender,
                sequence: sequence,
                nonce: nonce,
                payload: payload,
                // Sui is an instant finality chain, so we don't need
                // confirmations
                consistency_level: 0,
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
        set::add<vector<u8>>(&mut state.consumed_governance_actions, hash);
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
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_from_address, take_shared, return_shared};

    use wormhole::state::{Self, test_init, State, DeployerCapability};
    use wormhole::myu16::{Self as u16};

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
                21,
                1, // governance chain
                x"0000000000000000000000000000000000000000000000000000000000000004", // governance_contract
                vector[x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"], // initial_guardian(s)
                message_fee, // message fee = 0
                ctx(&mut test));
        };
        return test
    }

    #[test]
    fun test_state_setters() {
        test_state_setters_(scenario())
    }

    fun test_state_setters_(test: Scenario) {
        let (admin, _, _) = people();
        test = init_wormhole_state(test, admin, 0);

        // test setters
        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);

            // test set chain id
            state::test_set_chain_id(&mut state, 5);
            assert!(state::get_chain_id(&state) == u16::from_u64(5), 0);

            // test set governance chain id
            state::test_set_governance_chain_id(&mut state, 100);
            assert!(state::get_governance_chain(&state) == u16::from_u64(100), 0);

            return_shared<State>(state);
        };
        test_scenario::end(test);
    }
}
