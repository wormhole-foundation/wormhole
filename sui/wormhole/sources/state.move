module wormhole::state {
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use sui::transfer::{Self};
    use sui::vec_map::{Self, VecMap};

    use std::vector::{Self};

    use wormhole::myu16::{Self as u16, U16};
    use wormhole::myu32::{Self as u32, U32};
    use wormhole::structs::{Self, Guardian, GuardianSet};
    use wormhole::external_address::{Self, ExternalAddress};

    friend wormhole::guardian_set_upgrade;
    //friend wormhole::contract_upgrade;
    //friend wormhole::wormhole;
    friend wormhole::myvaa;

    struct State has key, store {
        id: UID,
        chain_id: U16,
        governance_chain_id: U16,
        guardian_set_index: U32,
        guardian_sets: VecMap<U32, GuardianSet>,
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
            guardian_sets: vec_map::empty<U32, GuardianSet>(),
        });
    }

    public fun set_chain_id(state: &mut State, id: u64){
        state.chain_id = u16::from_u64(id);
    }

    public fun set_governance_chain_id(state: &mut State, id: u64){
        state.governance_chain_id = u16::from_u64(id);
    }

    public fun set_governance_action_consumed(_hash: vector<u8>){
    }

    public entry fun create_governance_set(set_id: u64, state: &mut State, ctx: &mut TxContext){
        transfer::transfer_to_object(GovernanceSet {id: object::new(ctx), set_id}, state);
    }

    // TODO change the following

    //public entry fun guardian_set_is_active()
    public(friend) fun update_guardian_set_index(_new_index: U32) {
        //let state = borrow_global_mut<WormholeState>(@wormhole);
        //state.guardian_set_index= new_index;
    }

    public fun get_guardian_set(_index: U32): GuardianSet {
        //let state = borrow_global_mut<WormholeState>(@wormhole);
        //*table::borrow<u64, GuardianSet>(&mut state.guardian_sets, u32::to_u64(index))
        return structs::create_guardian_set(u32::from_u64(0), vector::empty<Guardian>())
    }

    public(friend) fun expire_guardian_set(_index: U32) {
        //let state = borrow_global_mut<WormholeState>(@wormhole);
        //let guardian_set: &mut GuardianSet = table::borrow_mut<u64, GuardianSet>(&mut state.guardian_sets, u32::to_u64(index));
        //let expiry = state.guardian_set_expiry;
        //structs::expire_guardian_set(guardian_set, expiry);
    }

    public(friend) fun store_guardian_set(state: &mut State, index: U32, set: GuardianSet) {
        vec_map::insert<U32, GuardianSet>(&mut state.guardian_sets, index, set);
        //let state = borrow_global_mut<WormholeState>(@wormhole);
        //let index: u64 = u32::to_u64(structs::get_guardian_set_index(&set));
        //table::add(&mut state.guardian_sets, index, set);
    }

    public fun guardian_set_is_active(_guardian_set: &GuardianSet): bool {
        // let index = structs::get_guardian_set_index(guardian_set);
        // let current_index = get_current_guardian_set_index();
        // let now = timestamp::now_seconds();

        // index == current_index ||
        //     u32::to_u64(structs::get_guardian_set_expiry(guardian_set)) > now
        return true
    }

    public fun get_current_guardian_set_index(): U32 {
        return u32::from_u64(0)
    }

    public fun get_governance_chain(): U16 {
        return u16::from_u64(0)
    }

     public fun get_governance_contract(): ExternalAddress {
        return external_address::from_bytes(x"8c82b2fd82faed2711d59af0f2499d16e726f6b2")
    }

}
