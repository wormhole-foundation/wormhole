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

    friend wormhole::guardian_set_upgrade;
    //friend wormhole::contract_upgrade;
    //friend wormhole::wormhole;
    friend wormhole::myvaa;

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

        /// wormhole message fee
        message_fee: u64,
    }

    public entry fun init_state(ctx: &mut TxContext){
        transfer::share_object(State {
            id: object::new(ctx),
            chain_id: u16::from_u64(1234),
            governance_chain_id: u16::from_u64(1234),
            governance_contract: external_address::from_bytes(vector::empty<u8>()),
            guardian_set_index: u32::from_u64(1234),
            guardian_sets: vec_map::empty<U32, GuardianSet>(),
            guardian_set_expiry: u32::from_u64(0),
            consumed_governance_actions: vec_set::empty<vector<u8>>(),
            message_fee: 0,
        });
    }

    public(friend) entry fun publish_event(
        //sender: u128,
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
                // Aptos is an instant finality chain, so we don't need
                // confirmations
                consistency_level: 0,
                timestamp: now // this is current epoch and not seconds
            }
        );
    }

    // setters

    public fun set_chain_id(state: &mut State, id: u64){
        state.chain_id = u16::from_u64(id);
    }

    public fun set_governance_chain_id(state: &mut State, id: u64){
        state.governance_chain_id = u16::from_u64(id);
    }

    public fun set_governance_action_consumed(state: &mut State, hash: vector<u8>){
        vec_set::insert<vector<u8>>(&mut state.consumed_governance_actions, hash);
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

}
