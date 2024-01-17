// SPDX-License-Identifier: Apache 2

/// This module implements two methods: `prepare_message` and `publish_message`,
/// which are to be executed in a transaction block in this order.
///
/// `prepare_message` allows a contract to pack Wormhole message info (payload
/// that has meaning to an integrator plus nonce) in preparation to publish a
/// `WormholeMessage` event via `publish_message`. Only the owner of an
/// `EmitterCap` has the capability of creating this `MessageTicket`.
///
/// `publish_message` unpacks the `MessageTicket` and emits a
/// `WormholeMessage` with this message info and timestamp. This event is
/// observed by the Guardian network.
///
/// The purpose of splitting this message publishing into two steps is in case
/// Wormhole needs to be upgraded and there is a breaking change for this
/// module, an integrator would not be left broken. It is discouraged to put
/// `publish_message` in an integrator's package logic. Otherwise, this
/// integrator needs to be prepared to upgrade his contract to handle the latest
/// version of `publish_message`.
///
/// Instead, an integtrator is encouraged to execute a transaction block, which
/// executes `publish_message` using the latest Wormhole package ID and to
/// implement `prepare_message` in his contract to produce `MessageTicket`,
/// which `publish_message` consumes.
module wormhole::publish_message {
    use sui::coin::{Self, Coin};
    use sui::clock::{Self, Clock};
    use sui::object::{Self, ID};
    use sui::sui::{SUI};

    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::state::{Self, State};

    /// This type is emitted via `sui::event` module. Guardians pick up this
    /// observation and attest to its existence.
    struct WormholeMessage has drop, copy {
        /// `EmitterCap` object ID.
        sender: ID,
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

    /// This type represents Wormhole message data. The sender is the object ID
    /// of an `EmitterCap`, who acts as the capability of creating this type.
    /// The only way to destroy this type is calling `publish_message` with
    /// a fee to emit a `WormholeMessage` with the unpacked members of this
    /// struct.
    struct MessageTicket {
        /// `EmitterCap` object ID.
        sender: ID,
        /// From `EmitterCap`.
        sequence: u64,
        /// A.K.A. Batch ID.
        nonce: u32,
        /// Arbitrary message data relevant to integrator.
        payload: vector<u8>
    }

    /// `prepare_message` constructs Wormhole message parameters. An
    /// `EmitterCap` provides the capability to send an arbitrary payload.
    ///
    /// NOTE: Integrators of Wormhole should be calling only this method from
    /// their contracts. This method is not guarded by version control (thus not
    /// requiring a reference to the Wormhole `State` object), so it is intended
    /// to work for any package version.
    public fun prepare_message(
        emitter_cap: &mut EmitterCap,
        nonce: u32,
        payload: vector<u8>
    ): MessageTicket {
        // Produce sequence number for this message. This will also be the
        // return value for this method.
        let sequence = emitter::use_sequence(emitter_cap);

        MessageTicket {
            sender: object::id(emitter_cap),
            sequence,
            nonce,
            payload
        }
    }

    /// `publish_message` emits a message as a Sui event. This method uses the
    /// input `EmitterCap` as the registered sender of the
    /// `WormholeMessage`. It also produces a new sequence for this emitter.
    ///
    /// NOTE: This method is guarded by a minimum build version check. This
    /// method could break backward compatibility on an upgrade.
    ///
    /// It is important for integrators to refrain from calling this method
    /// within their contracts. This method is meant to be called in a
    /// transaction block after receiving a `MessageTicket` from calling
    /// `prepare_message` within a contract. If in a circumstance where this
    /// module has a breaking change in an upgrade, `prepare_message` will not
    /// be affected by this change.
    ///
    /// See `prepare_message` for more details.
    public fun publish_message(
        wormhole_state: &mut State,
        message_fee: Coin<SUI>,
        prepared_msg: MessageTicket,
        the_clock: &Clock
    ): u64 {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(wormhole_state);

        // Deposit `message_fee`. This method interacts with the `FeeCollector`,
        // which will abort if `message_fee` does not equal the collector's
        // expected fee amount.
        state::deposit_fee(
            &latest_only,
            wormhole_state,
            coin::into_balance(message_fee)
        );

        let MessageTicket {
            sender,
            sequence,
            nonce,
            payload
        } = prepared_msg;

        // Truncate to seconds.
        let timestamp = clock::timestamp_ms(the_clock) / 1000;

        // Sui is an instant finality chain, so we don't need confirmations.
        let consistency_level = 0;

        // Emit Sui event with `WormholeMessage`.
        sui::event::emit(
            WormholeMessage {
                sender,
                sequence,
                nonce,
                payload,
                consistency_level,
                timestamp
            }
        );

        // Done.
        sequence
    }

    #[test_only]
    public fun destroy(prepared_msg: MessageTicket) {
        let MessageTicket {
            sender: _,
            sequence: _,
            nonce: _,
            payload: _
        } = prepared_msg;
    }
}

#[test_only]
module wormhole::publish_message_tests {
    use sui::coin::{Self};
    use sui::test_scenario::{Self};

    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::fee_collector::{Self};
    use wormhole::state::{Self};
    use wormhole::version_control::{Self};
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
    fun test_publish_message() {
        use wormhole::publish_message::{prepare_message, publish_message};

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

            // Check for event corresponding to new emitter.
            let effects = test_scenario::next_tx(scenario, user);
            assert!(test_scenario::num_user_events(&effects) == 1, 0);

            // Prepare message.
            let msg =
                prepare_message(
                    &mut emitter_cap,
                    0, // nonce
                    b"Hello World"
                );

            // Finally publish Wormhole message.
            let sequence =
                publish_message(
                    &mut worm_state,
                    coin::mint_for_testing(
                        wormhole_message_fee,
                        test_scenario::ctx(scenario)
                    ),
                    msg,
                    &the_clock
                );
            assert!(sequence == 0, 0);

            // Prepare another message.
            let msg =
                prepare_message(
                    &mut emitter_cap,
                    0, // nonce
                    b"Hello World... again"
                );

            // Publish again to check sequence uptick.
            let another_sequence =
                publish_message(
                    &mut worm_state,
                    coin::mint_for_testing(
                        wormhole_message_fee,
                        test_scenario::ctx(scenario)
                    ),
                    msg,
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

            let msg =
                prepare_message(
                    &mut emitter_cap,
                    0, // nonce
                    b"Hello?"
                );

            let sequence =
                publish_message(
                    &mut worm_state,
                    coin::mint_for_testing(
                        wormhole_message_fee,
                        test_scenario::ctx(scenario)
                    ),
                    msg,
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
    fun test_cannot_publish_message_with_incorrect_fee() {
        use wormhole::publish_message::{prepare_message, publish_message};

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

        let msg =
            prepare_message(
                &mut emitter_cap,
                0, // nonce
                b"Hello World"
            );
        // You shall not pass!
        publish_message(
            &mut worm_state,
            coin::mint_for_testing(
                wrong_fee_amount,
                test_scenario::ctx(scenario)
            ),
            msg,
            &the_clock
        );

        // Clean up.
        emitter::destroy_test_only(emitter_cap);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    /// This test verifies that `publish_message` will fail if the minimum
    /// required version is greater than the current build's.
    fun test_cannot_publish_message_outdated_version() {
        use wormhole::publish_message::{prepare_message, publish_message};

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

        // User needs an `EmitterCap` so he can send a message.
        let emitter_cap =
            emitter::new(&worm_state, test_scenario::ctx(scenario));

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut worm_state);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        state::migrate_version_test_only(
            &mut worm_state,
            version_control::previous_version_test_only(),
            version_control::next_version()
        );

        let msg =
            prepare_message(
                &mut emitter_cap,
                0, // nonce
                b"Hello World",
            );

        // You shall not pass!
        publish_message(
            &mut worm_state,
            coin::mint_for_testing(
                wormhole_message_fee,
                test_scenario::ctx(scenario)
            ),
            msg,
            &the_clock
        );

        // Clean up.
        emitter::destroy_test_only(emitter_cap);

        abort 42
    }
}
