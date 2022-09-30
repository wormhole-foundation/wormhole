module wormhole::state {
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use sui::transfer::{Self};
    use sui::vec_set::{Self, VecSet};

    use wormhole::myu16::{Self as u16, U16};
    use wormhole::myu32::{Self as u32, U32};

    struct State has key, store {
        id: UID,
        chain_id: U16,
        governance_chain_id: U16,
        guardian_set_index: U32,
        my_vec_set: VecSet<u64>,
    }

    struct GovernanceSet has key, store {
        id: UID,
        set_id: u64,
    }

    fun init(ctx: &mut TxContext){
        // share object
        transfer::share_object(State {
            id: object::new(ctx),
            chain_id: u16::from_u64(1234),
            governance_chain_id: u16::from_u64(1234),
            guardian_set_index: u32::from_u64(1234),
            my_vec_set: vec_set::empty<u64>(),
        });
    }

    public entry fun set_chain_id(state: &mut State, id: u64){
        state.chain_id = u16::from_u64(id);
    }

    public entry fun set_governance_chain_id(state: &mut State, id: u64){
        state.governance_chain_id = u16::from_u64(id);
    }

    public entry fun add_to_vec_set(state: &mut State, elem: u64){
        vec_set::insert<u64>(&mut state.my_vec_set, elem);
    }

    public entry fun create_governance_set(set_id: u64, state: &mut State, ctx: &mut TxContext){
        transfer::transfer_to_object(GovernanceSet {id: object::new(ctx), set_id}, state);
    }

}
