module wormhole::state {
    use std::table::{Self, Table};
    use std::event::{Self, EventHandle};
    use std::signer::{address_of};
    use std::account;
    use std::timestamp;
    use wormhole::structs::{Self, GuardianSet};
    use wormhole::u16::{U16};
    use wormhole::u32::{Self, U32};
    use wormhole::u256::{Self, U256};

    friend wormhole::guardian_set_upgrade;
    friend wormhole::contract_upgrade;
    friend wormhole::wormhole;
    friend wormhole::vaa;

    struct GuardianSetChanged has store, drop {
        oldGuardianIndex: U32,
        newGuardianIndex: U32,
    }

    struct WormholeMessage has store, drop {
        sender: address,
        sequence: u64,
        nonce: u64,
        payload: vector<u8>,
        consistency_level: u8,
        timestamp: u64,
    }

    struct WormholeMessageHandle has key, store {
        event: EventHandle<WormholeMessage>
    }

    struct GuardianSetChangedHandle has key, store {
        event: EventHandle<GuardianSetChanged>
    }

    struct WormholeState has key {
        // This chain's id
        chain_id: U16,

        // Governance chain's id
        governance_chain_id: U16,

        // Address of governance contract on governance chain
        governance_contract: vector<u8>, //(TODO: create custom type for wormhole addresses)

        // Mapping of guardian_set_index => guardian set
        guardian_sets: Table<u64, GuardianSet>,

        // Current active guardian set index
        guardian_set_index: U32,

        // Period for which a guardian set stays active after it has been replaced
        guardian_set_expiry: U32,

        // Sequence numbers per emitter
        sequences: Table<address, u64>,

        // Mapping of consumed governance actions
        consumed_governance_actions: Table<vector<u8>, bool>,

        message_fee: U256,

        // The signer capability for wormhole itself
        signer_cap: account::SignerCapability
    }

    //create some empty tables and stuff...
    public(friend) fun init_wormhole_state(
        wormhole: &signer,
        chain_id: U16,
        governance_chain_id: U16,
        governance_contract: vector<u8>,
        guardian_set_expiry: U32,
        signer_cap: account::SignerCapability
    ) {
        move_to(wormhole, WormholeState {
            chain_id,
            governance_chain_id,
            governance_contract,
            guardian_sets: table::new<u64, GuardianSet>(),
            guardian_set_index: u32::from_u64(0),
            guardian_set_expiry,
            sequences: table::new<address, u64>(),
            consumed_governance_actions: table::new<vector<u8>, bool>(),
            message_fee: u256::from_u64(0),
            signer_cap
        });
    }

    public fun create_wormhole_message_handle(e: EventHandle<WormholeMessage>): WormholeMessageHandle {
        WormholeMessageHandle{
            event: e
        }
    }

    public fun create_guardian_set_changed_handle(e: EventHandle<GuardianSetChanged>): GuardianSetChangedHandle {
        GuardianSetChangedHandle{
            event: e
        }
    }

    public(friend) fun init_message_handles(admin: &signer) {
        move_to(admin, create_wormhole_message_handle(account::new_event_handle<WormholeMessage>(admin)));
        move_to(admin, create_guardian_set_changed_handle(account::new_event_handle<GuardianSetChanged>(admin)));
    }

    fun use_sequence(emitter: address): u64 acquires WormholeState {
        let sequence = next_sequence(emitter);
        set_next_sequence(emitter, sequence + 1);
        sequence
    }

    public entry fun publish_message(
        //TODO(csongor): don't require a signer here and instead take an emitter
        // capability
        sender: &signer,
        nonce: u64,
        payload: vector<u8>,
        consistency_level: u8,
     ) acquires WormholeState, WormholeMessageHandle{
        let addr = address_of(sender);
        let sequence = use_sequence(addr);
        let event_handle = borrow_global_mut<WormholeMessageHandle>(@wormhole);
        let now = aptos_framework::timestamp::now_seconds();

        event::emit_event<WormholeMessage>(
            &mut event_handle.event,
            WormholeMessage {
                sender: addr,
                sequence,
                nonce: nonce,
                payload,
                consistency_level,
                timestamp: now
            }
        );
    }

    public(friend) fun update_guardian_set_index(new_index: U32) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        state.guardian_set_index= new_index;
    }

    public fun get_guardian_set(index: U32): GuardianSet acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        *table::borrow<u64, GuardianSet>(&mut state.guardian_sets, u32::to_u64(index))
    }

    public(friend) fun expire_guardian_set(index: U32) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        let guardian_set: &mut GuardianSet = table::borrow_mut<u64, GuardianSet>(&mut state.guardian_sets, u32::to_u64(index));
        let expiry = state.guardian_set_expiry;
        structs::expire_guardian_set(guardian_set, expiry);
    }

    public(friend) fun store_guardian_set(set: GuardianSet) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        let index: u64 = u32::to_u64(structs::get_guardian_set_index(&set));
        table::add(&mut state.guardian_sets, index, set);
    }

    public fun guardian_set_is_active(guardian_set: &GuardianSet): bool acquires WormholeState {
        let index = structs::get_guardian_set_index(guardian_set);
        let current_index = get_current_guardian_set_index();
        let now = timestamp::now_seconds();

        index == current_index ||
            u32::to_u64(structs::get_guardian_set_expiry(guardian_set)) > now
    }

    public(friend) fun set_governance_action_consumed(hash: vector<u8>) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        table::add(&mut state.consumed_governance_actions, hash, true);
    }

    public(friend) fun set_chain_id(chain_id: U16) acquires WormholeState {
        borrow_global_mut<WormholeState>(@wormhole).chain_id = chain_id;
    }

    public(friend) fun set_governance_chain_id(chain_id: U16) acquires WormholeState {
        borrow_global_mut<WormholeState>(@wormhole).governance_chain_id = chain_id;
    }

    public(friend) fun set_governance_contract(governance_contract: vector<u8>) acquires WormholeState {
        borrow_global_mut<WormholeState>(@wormhole).governance_contract = governance_contract;
    }

    public(friend) fun set_message_fee(new_fee: U256) acquires WormholeState {
        borrow_global_mut<WormholeState>(@wormhole).message_fee = new_fee;
    }

    // TODO: we could move the sequence counter into the emitters instead of
    // having a shared object. That would be more efficient
    public entry fun set_next_sequence(emitter: address, sequence: u64) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        if (table::contains(&state.sequences, emitter)){
            table::remove(&mut state.sequences, emitter);
        };
        table::add(&mut state.sequences, emitter, sequence);
    }

    // getters

    public entry fun next_sequence(emitter: address):u64 acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        if (table::contains(&state.sequences, emitter)){
            return *table::borrow(&state.sequences, emitter)
        };
        return 0
    }

    public fun get_current_guardian_set_index(): U32 acquires WormholeState {
        let state = borrow_global<WormholeState>(@wormhole);
        state.guardian_set_index
    }

    public fun get_current_guardian_set(): GuardianSet acquires WormholeState {
        let state = borrow_global<WormholeState>(@wormhole);
        let ind = u32::to_u64(state.guardian_set_index);
        *table::borrow(&state.guardian_sets, ind)
    }

    public fun get_governance_chain(): U16 acquires WormholeState {
        borrow_global<WormholeState>(@wormhole).governance_chain_id
    }

    /// Provide access to the wormhole contract signer. Be *very* careful who
    /// gets access to this!
    public(friend) fun wormhole_signer(): signer acquires WormholeState {
        account::create_signer_with_capability(&borrow_global<WormholeState>(@wormhole).signer_cap)
    }
}
