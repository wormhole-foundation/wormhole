module wormhole::state {
    use 0x1::table::{Self, Table};
    use 0x1::event::{Self, EventHandle};
    use 0x1::signer::{address_of};
    use 0x1::vector::{Self};
    use 0x1::account::{Self};
    use wormhole::structs::{GuardianSet};
    use wormhole::u16::{Self, U16};
    use wormhole::u32::{Self, U32};
    use wormhole::u256::{Self, U256};

    friend wormhole::governance;
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

    struct Provider has key, store {
        chain_id: U16,
        governance_chain_id: U16,
        governance_contract: vector<u8>, //bytes32 (TODO: create custom type for wormhole addresses)
    }

    struct WormholeState has key {
        provider: Provider,

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

        // Mapping of initialized implementations
        initialized_implementations: Table<address, bool>,

        message_fee: U256,
    }

    //create some empty tables and stuff...
    public(friend) fun init_wormhole_state(admin: &signer) {
        move_to(admin, WormholeState {
            provider: Provider {
                chain_id: u16::from_u64(0),
                governance_chain_id: u16::from_u64(0),
                governance_contract: vector::empty<u8>()
            },
            guardian_sets: table::new<u64, GuardianSet>(),
            guardian_set_index: u32::from_u64(0),
            guardian_set_expiry: u32::from_u64(0),
            sequences: table::new<address, u64>(),
            consumed_governance_actions: table::new<vector<u8>, bool>(),
            initialized_implementations: table::new<address, bool>(),
            message_fee: u256::from_u64(0)
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

    public(friend) fun update_guardian_set_index(newIndex: U32) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        state.guardian_set_index= newIndex;
    }

    public(friend) fun expire_guardian_set(_index: u64) acquires WormholeState {
        let _state = borrow_global_mut<WormholeState>(@wormhole);
        //TODO: expire guardian set, when we can index into guardian_sets with state.guardian_set_index(a U32)
        //let guardian_set = table::borrow_mut<u64, GuardianSet>(&mut state.guardian_sets, state.guardian_set_index);
        //Structs::expire_guardian_set(guardian_set);
    }

    public(friend) fun store_guardian_set(_set: GuardianSet, _index: U32) acquires WormholeState {
        let _state = borrow_global_mut<WormholeState>(@wormhole);
        //TODO: store guardian set under index (U32)
        //table::add(&mut state.guardian_sets, index, set);
    }

    // TODO: setInitialized?

    public(friend) fun set_governance_action_consumed(hash: vector<u8>) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        table::add(&mut state.consumed_governance_actions, hash, true);
    }

    public(friend) fun set_chain_id(chaind_id: U16) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        let provider = &mut state.provider;
        provider.chain_id = chaind_id;
    }

    public(friend) fun set_governance_chain_id(chain_id: U16) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        let provider = &mut state.provider;
        provider.governance_chain_id = chain_id;
    }

    public(friend) fun set_governance_contract(governance_contract: vector<u8>) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        let provider = &mut state.provider;
        provider.governance_contract = governance_contract;
    }

    public(friend) fun set_message_fee(new_fee: U256) acquires WormholeState {
        let state = borrow_global_mut<WormholeState>(@wormhole);
        state.message_fee = new_fee;
    }

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
        let _ind = state.guardian_set_index;
        //TODO: fetch ind instead of 0
        //*table::borrow(&state.guardianSets, ind)
        *table::borrow(&state.guardian_sets, 0)
    }
}
