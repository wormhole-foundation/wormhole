module token_bridge::registered_emitters {
    use sui::dynamic_object_field::{Self};
    use sui::object::{UID};
    use sui::table::{Self};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};

    const KEY: vector<u8> = b"registered_emitters";

    const E_EMITTER_EXTERNAL_ADDRESS_ALREADY_EXISTS_FOR_CHAIN: u64 = 0;
    const E_EMITTER_EXTERNAL_ADDRESS_DOES_NOT_EXIST_FOR_CHAIN: u64 = 1;
    const E_REGISTERED_EMITTER_TABLE_ALREADY_EXISTS: u64 = 2;

    public fun new(parent_id: &mut UID, ctx: &mut TxContext) {
        assert!(
            !dynamic_object_field::exists_(parent_id, KEY),
            E_REGISTERED_EMITTER_TABLE_ALREADY_EXISTS
        );
        dynamic_object_field::add(
            parent_id,
            KEY,
            table::new<u16, ExternalAddress>(ctx)
        )
    }

    public fun add(parent_id: &mut UID, chain: u16, addr: ExternalAddress) {
        assert!(
            !has(parent_id, chain),
            E_EMITTER_EXTERNAL_ADDRESS_ALREADY_EXISTS_FOR_CHAIN
        );
        table::add(
            dynamic_object_field::borrow_mut(parent_id, KEY),
            chain,
            addr
        )
    }

    public fun has(parent_id: &UID, chain: u16): bool {
        let ref = dynamic_object_field::borrow(parent_id, KEY);
        table::contains<u16, ExternalAddress>(ref, chain)
    }

    public fun external_address(parent_id: &UID, chain: u16): ExternalAddress {
        assert!(
            has(parent_id, chain),
            E_EMITTER_EXTERNAL_ADDRESS_DOES_NOT_EXIST_FOR_CHAIN
        );
        *table::borrow(dynamic_object_field::borrow(parent_id, KEY), chain)
    }
}

#[test_only]
module token_bridge::registered_emitters_test {
    use std::vector::{Self};
    use sui::object::{Self, UID};
    use sui::tx_context::{dummy};

    use wormhole::external_address::{from_any_bytes};
    use wormhole::bytes::{Self};

    use token_bridge::registered_emitters::{
        new,
        add,
        has,
        external_address,
        E_EMITTER_EXTERNAL_ADDRESS_ALREADY_EXISTS_FOR_CHAIN,
        E_EMITTER_EXTERNAL_ADDRESS_DOES_NOT_EXIST_FOR_CHAIN,
        E_REGISTERED_EMITTER_TABLE_ALREADY_EXISTS
    };

    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    struct MockState has key {
        id: UID
    }

    public fun destroy_mock_state(m: MockState) {
        let MockState {id} = m;
        object::delete(id);
    }

    // write a test exercising the creation of a new registered emitters table
    // and adding several (chain_id, external_address) key-value pairs to it
    #[test]
    fun test_registered_emitters(){
        let mock_state = MockState {id: object::new(&mut dummy())};

        // create table of registered emitters as a dynamic field of our
        // mock state
        new(&mut mock_state.id, &mut dummy());

        // add many (chain id, external address) key-value pairs to the
        // registered emitters table attached to mock state
        let i = 1;
        while (i < 1000) {
            let cur_external_addr = vector::empty<u8>();
            bytes::push_u16_be(&mut cur_external_addr, i);
            add(
                &mut mock_state.id,
                i,
                from_any_bytes(cur_external_addr)
            );
            i = i + 1;
        };

        // check that the key-value pairs we just added to the registered
        // emitters table were indeed added
        i = 1;
        while (i < 1000) {
            // check that key (chain id) exists
            assert!(
                has(
                    &mock_state.id,
                    i
                ),
                0
            );

            // check that values (external addresses) are correct
            let cur_external_addr = vector::empty<u8>();
            bytes::push_u16_be(&mut cur_external_addr, i);
            assert!(
                external_address(&mock_state.id,i) ==
                from_any_bytes(cur_external_addr),
            0);
            i = i + 1;
        };
        destroy_mock_state(mock_state);
    }

    // test that creating two registered emitter tables for the same UID
    // fails
    #[test]
    #[expected_failure(
        abort_code = E_REGISTERED_EMITTER_TABLE_ALREADY_EXISTS,
        location=token_bridge::registered_emitters
    )]
    fun test_registered_emitters_table_already_exists(){
        let mock_state = MockState {id: object::new(&mut dummy())};

        // create table of registered emitters as a dynamic field of our
        // mock state
        new(&mut mock_state.id, &mut dummy());

        // attempt to create another table for storing registered
        // emitters and attach it to mock_state under key b"registered_emitters"
        // resulting in failure, because that key already exists
        new(&mut mock_state.id, &mut dummy());

        destroy_mock_state(mock_state);
    }

    // test that only one external address can be registered for a chain
    #[test]
    #[expected_failure(
        abort_code = E_EMITTER_EXTERNAL_ADDRESS_ALREADY_EXISTS_FOR_CHAIN,
        location=token_bridge::registered_emitters
    )]
    fun test_register_chain_id_twice(){
        let mock_state = MockState {id: object::new(&mut dummy())};

        // create table of registered emitters as a dynamic field of our
        // mock state
        new(&mut mock_state.id, &mut dummy());

        // try to add chain_id 1 more than once, resulting in failure
        let i = 1;
        while (i < 2) {
            let cur_external_addr = vector::empty<u8>();
            bytes::push_u16_be(&mut cur_external_addr, i);
            add(
                &mut mock_state.id,
                i,
                from_any_bytes(cur_external_addr)
            );
        };
        destroy_mock_state(mock_state);
    }

    #[test]
    #[expected_failure(
        abort_code = E_EMITTER_EXTERNAL_ADDRESS_DOES_NOT_EXIST_FOR_CHAIN,
        location=token_bridge::registered_emitters
    )]
    fun test_registered_emitters_nonexistent_external_address(){
        let mock_state = MockState {id: object::new(&mut dummy())};

        // create table of registered emitters as a dynamic field of our
        // mock state
        new(&mut mock_state.id, &mut dummy());

        // register chain ids 1-100
        let i = 1;
        while (i < 100) {
            let cur_external_addr = vector::empty<u8>();
            bytes::push_u16_be(&mut cur_external_addr, i);
            add(
                &mut mock_state.id,
                i,
                from_any_bytes(cur_external_addr)
            );
            i = i + 1;
        };
        let _nonexistent_external_address = external_address(
            &mut mock_state.id,
            10022 // this chain id is not registered (only 1-100 are registered)
        );
        destroy_mock_state(mock_state);
    }
}
