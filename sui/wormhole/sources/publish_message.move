// SPDX-License-Identifier: Apache 2

/// This module implements the method `publish_message` which emits a
/// `WormholeMessage` event. This event is observed by the Guardian network and
/// must have information relating to emitter (`sender`), message sequence
/// relative to the emitter, nonce (i.e. batch ID), consistency level
/// (i.e. finality) and arbitrary message payload.
module wormhole::publish_message {
    use sui::balance::{Balance};
    use sui::clock::{Self, Clock};
    use sui::event::{Self};
    use sui::object::{Self};
    use sui::sui::{SUI};

    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::state::{Self, State};
    use wormhole::version_control::{PublishMessage as PublishMessageControl};

    /// `WormholeMessage` to be emitted via sui::event::emit.
    struct WormholeMessage has drop, copy {
        /// Underlying bytes of `EmitterCap` external address.
        sender: vector<u8>,
        /// From `EmitterCap`.
        sequence: u64,
        /// A.K.A. Batch ID.
        nonce: u32,
        /// Arbitrary message data relevant to integrator.
        payload: vector<u8>,
        /// This will always be `0`.
        consistency_level: u8,
        /// `Clock` timestamp.
        timestamp: u64
    }

    /// `publish_message` emits a message as a Sui event. This method uses the
    /// input `EmitterCap` as the registered sender of the
    /// `WormholeMessage`. It also produces a new sequence for this emitter.
    ///
    /// NOTE: This method is guarded by a minimum build version check. This
    /// method could break backward compatibility on an upgrade.
    public fun publish_message(
        wormhole_state: &mut State,
        emitter_cap: &mut EmitterCap,
        nonce: u32,
        payload: vector<u8>,
        message_fee: Balance<SUI>,
        the_clock: &Clock
    ): u64 {
        state::check_minimum_requirement<PublishMessageControl>(wormhole_state);

        // Deposit `message_fee`. This method interacts with the `FeeCollector`,
        // which will abort if `message_fee` does not equal the collector's
        // expected fee amount.
        state::deposit_fee(wormhole_state, message_fee);

        // Produce sequence number for this message. This will also be the
        // return value for this method.
        let sequence = emitter::use_sequence(emitter_cap);

        // Truncate to seconds.
        let timestamp = clock::timestamp_ms(the_clock) / 1000;
        // Emit Sui event with `WormholeMessage`.
        event::emit(
            WormholeMessage {
                sender: object::id_to_bytes(&object::id(emitter_cap)),
                sequence,
                nonce,
                payload: payload,
                // Sui is an instant finality chain, so we don't need
                // confirmations. Do we even need to specify this?
                consistency_level: 0,
                timestamp
            }
        );

        // Done.
        sequence
    }
}

#[test_only]
module wormhole::publish_message_tests {
    use sui::balance::{Self};
    use sui::test_scenario::{Self};

    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::fee_collector::{Self};
    use wormhole::required_version::{Self};
    use wormhole::state::{Self};
    use wormhole::publish_message::{publish_message};
    use wormhole::version_control::{Self as control};
    use wormhole::wormhole_scenario::{
        person,
        return_clock,
        return_state,
        set_up_wormhole,
        take_clock,
        take_state,
        upgrade_wormhole
    };

    #[test]
    /// This test verifies that `publish_message` is successfully called when
    /// the specified message fee is used.
    public fun test_publish_message() {
        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        let wormhole_message_fee = 100000000;

        // Initialize Wormhole.
        set_up_wormhole(scenario, wormhole_message_fee);

        // Next transaction should be conducted as an ordinary user.
        test_scenario::next_tx(scenario, user);

        {
            let worm_state = take_state(scenario);
            let the_clock = take_clock(scenario);

            // User needs an `EmitterCap` so he can send a message.
            let emitter_cap =
                wormhole::emitter::new(
                    &worm_state,
                    test_scenario::ctx(scenario)
                );

            // Finally publish Wormhole message.
            let sequence =
                publish_message(
                    &mut worm_state,
                    &mut emitter_cap,
                    0, // nonce
                    b"Hello World",
                    balance::create_for_testing(wormhole_message_fee),
                    &the_clock
                );
            assert!(sequence == 0, 0);

            // Publish again to check sequence uptick.
            let another_sequence =
                publish_message(
                    &mut worm_state,
                    &mut emitter_cap,
                    0, // nonce
                    b"Hello World... again",
                    balance::create_for_testing(wormhole_message_fee),
                    &the_clock
                );
            assert!(another_sequence == 1, 0);

            // Clean up.
            return_state(worm_state);
            return_clock(the_clock);
            sui::transfer::public_transfer(emitter_cap, user);
        };

        // Grab the `TransactionEffects` of the previous transaction.
        let effects = test_scenario::next_tx(scenario, user);

        // We expect two events (the Wormhole messages). `test_scenario` does
        // not give us an in-depth view of the event specifically. But we can
        // check that there was an event associated with the previous
        // transaction.
        assert!(test_scenario::num_user_events(&effects) == 2, 0);

        // Simulate upgrade and confirm that publish message still works.
        {
            upgrade_wormhole(scenario);

            // Ignore effects from upgrade.
            test_scenario::next_tx(scenario, user);

            let worm_state = take_state(scenario);
            let the_clock = take_clock(scenario);
            let emitter_cap =
                test_scenario::take_from_sender<EmitterCap>(scenario);

            let sequence =
                publish_message(
                    &mut worm_state,
                    &mut emitter_cap,
                    0, // nonce
                    b"Hello?",
                    balance::create_for_testing(wormhole_message_fee),
                    &the_clock
                );
            assert!(sequence == 2, 0);

            // Clean up.
            test_scenario::return_to_sender(scenario, emitter_cap);
            return_state(worm_state);
            return_clock(the_clock);
        };

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = fee_collector::E_INCORRECT_FEE)]
    /// This test verifies that `publish_message` fails when the fee is not the
    /// correct amount. `FeeCollector` will be the reason for this abort.
    public fun test_cannot_publish_message_with_incorrect_fee() {
        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        let wormhole_message_fee = 100000000;
        let wrong_fee_amount = wormhole_message_fee - 1;

        // Initialize Wormhole.
        set_up_wormhole(scenario, wormhole_message_fee);

        // Next transaction should be conducted as an ordinary user.
        test_scenario::next_tx(scenario, user);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // User needs an `EmitterCap` so he can send a message.
        let emitter_cap =
            emitter::new(&worm_state, test_scenario::ctx(scenario));

        // You shall not pass!
        publish_message(
            &mut worm_state,
            &mut emitter_cap,
            0, // nonce
            b"Hello World",
            balance::create_for_testing(wrong_fee_amount),
            &the_clock
        );

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);
        emitter::destroy(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = required_version::E_OUTDATED_VERSION)]
    /// This test verifies that `publish_message` will fail if the minimum
    /// required version is greater than the current build's.
    public fun test_cannot_publish_message_outdated_build() {
        let user = person();
        let my_scenario = test_scenario::begin(user);
        let scenario = &mut my_scenario;

        let wormhole_message_fee = 100000000;

        // Initialize Wormhole.
        set_up_wormhole(scenario, wormhole_message_fee);

        // Next transaction should be conducted as an ordinary user.
        test_scenario::next_tx(scenario, user);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        state::set_required_version<control::PublishMessage>(
            &mut worm_state,
            control::version() + 1
        );

        // User needs an `EmitterCap` so he can send a message.
        let emitter_cap =
            emitter::new(&worm_state, test_scenario::ctx(scenario));

        // You shall not pass!
        publish_message(
            &mut worm_state,
            &mut emitter_cap,
            0, // nonce
            b"Hello World",
            balance::create_for_testing(wormhole_message_fee),
            &the_clock
        );

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);
        emitter::destroy(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }
}
