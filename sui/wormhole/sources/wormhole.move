module wormhole::wormhole {
    use sui::sui::{SUI};
    use sui::coin::{Self, Coin};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer::{Self};

    //use wormhole::structs::{create_guardian, create_guardian_set};
    use wormhole::state::{Self, State};
    use wormhole::emitter::{Self};

    const E_INSUFFICIENT_FEE: u64 = 3;
    const E_TOO_MUCH_FEE: u64 = 4;

// -----------------------------------------------------------------------------
// Sending messages
    public fun publish_message(
        emitter_cap: &mut emitter::EmitterCapability,
        state: &mut State,
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
    ): u64 {
        // ensure that provided fee is sufficient to cover message fees
        let expected_fee = state::get_message_fee(state);
        let val = coin::value(&message_fee);
        if (expected_fee != val){
            if (expected_fee < val){
                assert!(expected_fee > val, E_TOO_MUCH_FEE);
            } else {
                assert!(expected_fee < val, E_INSUFFICIENT_FEE);
            }
        };
        // deposit the fees into wormhole
        state::deposit_fee_coins<SUI>(state, message_fee);

        // get sequence number
        let sequence = emitter::use_sequence(emitter_cap);

        // emit event
        state::publish_event(
            emitter::get_emitter(emitter_cap),
            sequence,
            nonce,
            payload,
        );
        return sequence
    }

    public entry fun publish_message_entry(
        emitter_cap: &mut emitter::EmitterCapability,
        state: &mut State,
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
    ) {
        publish_message(emitter_cap, state, nonce, payload, message_fee);
    }

    public entry fun publish_message_free(
        emitter_cap: &mut emitter::EmitterCapability,
        state: &mut State,
        nonce: u64,
        payload: vector<u8>,
    ) {
        // ensure that provided fee is sufficient to cover message fees
        let expected_fee = state::get_message_fee(state);
        assert!(expected_fee == 0, E_INSUFFICIENT_FEE);

        // get sender and sequence number
        let sequence = emitter::use_sequence(emitter_cap);

        // emit event
        state::publish_event(
            emitter::get_emitter(emitter_cap),
            sequence,
            nonce,
            payload,
        );
    }

    // -----------------------------------------------------------------------------
    // Emitter registration

    public fun register_emitter(state: &mut State, ctx: &mut TxContext): emitter::EmitterCapability {
        state::new_emitter(state, ctx)
    }

    // -----------------------------------------------------------------------------
    // get_new_emitter
    //
    // Honestly, unsure if this should survive once we get into code review but it
    // sure makes writing my test script work quite well
    //
    // This creates a new emitter object and stores it away into the senders context.
    //
    // You can then use this to call publish_message_free and generate a vaa

    public entry fun get_new_emitter(state: &mut State, ctx: &mut TxContext) {
        transfer::transfer(state::new_emitter(state, ctx), tx_context::sender(ctx));
    }
}

#[test_only]
module wormhole::test_wormhole{
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_shared, return_shared};
    use sui::coin::{Self};
    use sui::sui::{SUI};
    use sui::transfer::{Self};

    use wormhole::test_state::{init_wormhole_state};
    use wormhole::state::{State};
    use wormhole::wormhole::{Self, E_TOO_MUCH_FEE, E_INSUFFICIENT_FEE};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test] // precisely the right amount of fee
    public fun test_publish_wormhole_message_nonzero_fee(){
        let test = scenario();
        let (admin, _, _) = people();
        test = init_wormhole_state(test, admin, 100000000); // wormhole fee set to 100000000 SUI
        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let emitter = wormhole::register_emitter(&mut state, ctx(&mut test));
            let message_fee = coin::mint_for_testing<SUI>(100000000, ctx(&mut test)); // fee amount == expected amount
            wormhole::publish_message(
                &mut emitter,
                &mut state,
                0,
                x"11223344556677889900",
                message_fee
            );
            return_shared<State>(state);
            transfer::transfer(emitter, admin);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = E_TOO_MUCH_FEE)] // E_TOO_MUCH_FEE
    public fun test_publish_wormhole_message_too_much_fee(){
        let test = scenario();
        let (admin, _, _) = people();
        test = init_wormhole_state(test, admin, 100000000); // wormhole fee set to 100000000 SUI
        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let emitter = wormhole::register_emitter(&mut state, ctx(&mut test));
            let message_fee = coin::mint_for_testing<SUI>(100000001, ctx(&mut test)); // fee amount > expected amount
            wormhole::publish_message(
                &mut emitter,
                &mut state,
                0,
                x"11223344556677889900",
                message_fee
            );
            return_shared<State>(state);
            transfer::transfer(emitter, admin);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = E_INSUFFICIENT_FEE)] // E_INSUFFICIENT_FEE
    public fun test_publish_wormhole_message_insufficient_fee(){
        let test = scenario();
        let (admin, _, _) = people();
        test = init_wormhole_state(test, admin, 100000000); // wormhole fee set to 100000000 SUI
        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let emitter = wormhole::register_emitter(&mut state, ctx(&mut test));
            let message_fee = coin::mint_for_testing<SUI>(99999999, ctx(&mut test)); // fee amount < expected amount
            wormhole::publish_message(
                &mut emitter,
                &mut state,
                0,
                x"11223344556677889900",
                message_fee
            );
            return_shared<State>(state);
            transfer::transfer(emitter, admin);
        };
        test_scenario::end(test);
    }
}
