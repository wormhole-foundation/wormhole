module wormhole::publish_message {
    use sui::sui::{SUI};
    use sui::coin::{Coin};
    use sui::event::{Self};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer::{Self};

    //use wormhole::structs::{create_guardian, create_guardian_set};
    use wormhole::state::{Self, State};
    use wormhole::emitter::{Self, EmitterCapability};
    use wormhole::external_address::{ExternalAddress};

    const E_INSUFFICIENT_FEE: u64 = 3;
    const E_TOO_MUCH_FEE: u64 = 4;

    struct WormholeMessage has store, copy, drop {
        sender: ExternalAddress,
        sequence: u64,
        nonce: u32,
        payload: vector<u8>,
        consistency_level: u8 // do we need this if Sui is instant finality?
    }

    public fun publish_message(
        wormhole_state: &mut State,
        emitter_cap: &mut EmitterCapability,
        nonce: u32,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
    ): u64 {
        // deposit the fees into wormhole
        state::deposit_fee(wormhole_state, message_fee);

        // get sequence number
        let sequence = emitter::use_sequence(emitter_cap);

        // emit event
        event::emit(
            WormholeMessage {
                sender: emitter::get_external_address(emitter_cap),
                sequence,
                nonce,
                payload: payload,
                // Sui is an instant finality chain, so we don't need
                // confirmations
                consistency_level: 0,
            }
        );

        sequence
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
module wormhole::publish_message_test{
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_shared, return_shared};
    use sui::coin::{Self};
    use sui::sui::{SUI};
    use sui::transfer::{Self};

    use wormhole::fee_collector::{Self};
    use wormhole::test_state::{init_wormhole_state};
    use wormhole::state::{State};
    use wormhole::publish_message::{Self as wormhole};

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
                &mut state,
                &mut emitter,
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
    #[expected_failure(abort_code = fee_collector::E_INCORRECT_FEE)]
    public fun test_publish_wormhole_message_too_much_fee(){
        let test = scenario();
        let (admin, _, _) = people();
        test = init_wormhole_state(test, admin, 100000000); // wormhole fee set to 100000000 SUI
        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let emitter = wormhole::register_emitter(&mut state, ctx(&mut test));
            let message_fee = coin::mint_for_testing<SUI>(100000001, ctx(&mut test)); // fee amount > expected amount
            wormhole::publish_message(
                &mut state,
                &mut emitter,
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
    #[expected_failure(abort_code = fee_collector::E_INCORRECT_FEE)]
    public fun test_publish_wormhole_message_insufficient_fee(){
        let test = scenario();
        let (admin, _, _) = people();
        test = init_wormhole_state(test, admin, 100000000); // wormhole fee set to 100000000 SUI
        next_tx(&mut test, admin); {
            let state = take_shared<State>(&test);
            let emitter = wormhole::register_emitter(&mut state, ctx(&mut test));
            let message_fee = coin::mint_for_testing<SUI>(99999999, ctx(&mut test)); // fee amount < expected amount
            wormhole::publish_message(
                &mut state,
                &mut emitter,
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
