module wormhole::publish_message {
    use sui::coin::{Coin};
    use sui::event::{Self};
    use sui::sui::{SUI};

    use wormhole::version_control::{PublishMessage as PublishMessageControl};
    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::state::{Self, State};

    /// `WormholeMessage` to be emitted via sui::event::emit.
    struct WormholeMessage has store, copy, drop {
        sender: vector<u8>,
        sequence: u64,
        nonce: u32,
        payload: vector<u8>,
        consistency_level: u8 // do we need this if Sui is instant finality?
    }

    /// `publish_message` emits a message as a Sui event. This method uses the
    /// input `EmitterCap` as the registered sender of the
    /// `WormholeMessage`. It also produces a new sequence for this emitter.
    public fun publish_message(
        wormhole_state: &mut State,
        emitter_cap: &mut EmitterCap,
        nonce: u32,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
    ): u64 {
        state::check_minimum_requirement<PublishMessageControl>(wormhole_state);

        // Deposit `message_fee`. This method interacts with the `FeeCollector`,
        // which will abort if `message_fee` does not equal the collector's
        // expected fee amount.
        state::deposit_fee(wormhole_state, message_fee);

        // Produce sequence number for this message. This will also be the
        // return value for this method.
        let sequence = state::use_emitter_sequence(emitter_cap);

        // Emit Sui event with `WormholeMessage`.
        event::emit(
            WormholeMessage {
                sender: emitter::emitter_address(emitter_cap),
                sequence,
                nonce,
                payload: payload,
                // Sui is an instant finality chain, so we don't need
                // confirmations. Do we even need to specify this?
                consistency_level: 0,
            }
        );

        // Done.
        sequence
    }
}

#[test_only]
module wormhole::publish_message_test{
    use sui::coin::{Self};
    use sui::sui::{SUI};
    use sui::test_scenario::{Self};

    use wormhole::emitter::{Self};
    use wormhole::fee_collector::{Self};
    use wormhole::state::{State};
    use wormhole::publish_message::{publish_message};
    use wormhole::wormhole_scenario::{set_up_wormhole, three_people as people};

    #[test]
    /// This test verifies that `publish_message` is successfully called when
    /// the specified message fee is used.
    public fun test_publish_message() {
        let (admin, user, _) = people();
        let my_scenario = test_scenario::begin(admin);
        let scenario = &mut my_scenario;

        let wormhole_message_fee = 100000000;

        // Initialize Wormhole.
        set_up_wormhole(scenario, wormhole_message_fee);

        // Next transaction should be conducted as an ordinary user.
        test_scenario::next_tx(scenario, user);

        {
            let worm_state = test_scenario::take_shared<State>(scenario);

            // User needs an `EmitterCap` so he can send a message.
            let emitter =
                wormhole::state::new_emitter(
                    &mut worm_state,
                    test_scenario::ctx(scenario)
                );

            // Finally publish Wormhole message.
            let sequence = publish_message(
                &mut worm_state,
                &mut emitter,
                0, // nonce
                b"Hello World",
                coin::mint_for_testing<SUI>(
                    wormhole_message_fee,
                    test_scenario::ctx(scenario)
                )
            );
            assert!(sequence == 0, 0);

            // Publish again to check sequence uptick.
            let another_sequence = publish_message(
                &mut worm_state,
                &mut emitter,
                0, // nonce
                b"Hello World... again",
                coin::mint_for_testing<SUI>(
                    wormhole_message_fee,
                    test_scenario::ctx(scenario)
                )
            );
            assert!(another_sequence == 1, 0);

            // Clean up.
            test_scenario::return_shared<State>(worm_state);
            emitter::destroy_cap(emitter);
        };

        // Grab the `TransactionEffects` of the previous transaction.
        let effects = test_scenario::next_tx(scenario, user);

        // We expect two events (the Wormhole messages). `test_scenario` does
        // not give us an in-depth view of the event specifically. But we can
        // check that there was an event associated with the previous
        // transaction.
        assert!(test_scenario::num_user_events(&effects) == 2, 0);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = fee_collector::E_INCORRECT_FEE)]
    /// This test verifies that `publish_message` fails when the fee is not the
    /// correct amount. `FeeCollector` will be the reason for this abort.
    public fun test_cannot_publish_message_with_incorrect_fee() {
        let (admin, user, _) = people();
        let my_scenario = test_scenario::begin(admin);
        let scenario = &mut my_scenario;

        let wormhole_message_fee = 100000000;
        let wrong_fee_amount = wormhole_message_fee - 1;

        // Initialize Wormhole.
        set_up_wormhole(scenario, wormhole_message_fee);

        // Next transaction should be conducted as an ordinary user.
        test_scenario::next_tx(scenario, user);

        {
            let worm_state = test_scenario::take_shared<State>(scenario);

            // User needs an `EmitterCap` so he can send a message.
            let emitter =
                wormhole::state::new_emitter(
                    &mut worm_state,
                    test_scenario::ctx(scenario)
                );

            // Finally publish Wormhole message.
            publish_message(
                &mut worm_state,
                &mut emitter,
                0, // nonce
                b"Hello World",
                coin::mint_for_testing<SUI>(
                    wrong_fee_amount,
                    test_scenario::ctx(scenario)
                )
            );

            // Clean up.
            test_scenario::return_shared<State>(worm_state);
            emitter::destroy_cap(emitter);
        };

        // Done.
        test_scenario::end(my_scenario);
    }
}
