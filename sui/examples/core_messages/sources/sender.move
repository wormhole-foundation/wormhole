/// A simple contracts that demonstrates how to send messages with wormhole.
module core_messages::sender {
    use sui::clock::{Clock};
    use sui::coin::{Self};
    use sui::object::{Self, UID};
    use sui::transfer::{Self};
    use sui::tx_context::{TxContext};
    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::state::{State as WormholeState};

    struct State has key, store {
        id: UID,
        emitter_cap: EmitterCap,
    }

    /// Register ourselves as a wormhole emitter. This gives back an
    /// `EmitterCap` which will be required to send messages through
    /// wormhole.
    public fun init_with_params(
        wormhole_state: &WormholeState,
        ctx: &mut TxContext
    ) {
        transfer::share_object(
            State {
                id: object::new(ctx),
                emitter_cap: emitter::new(wormhole_state, ctx)
            }
        );
    }

    public fun send_message_entry(
        state: &mut State,
        wormhole_state: &mut WormholeState,
        payload: vector<u8>,
        the_clock: &Clock,
        ctx: &mut TxContext
    ) {
        send_message(
            state,
            wormhole_state,
            payload,
            the_clock,
            ctx
        );
    }

    /// NOTE: This is NOT the proper way of using the `prepare_message` and
    /// `publish_message` workflow. This example app is meant for testing for
    /// observing Wormhole messages via the guardian.
    ///
    /// See `publish_message` module for more info.
    public fun send_message(
        state: &mut State,
        wormhole_state: &mut WormholeState,
        payload: vector<u8>,
        the_clock: &Clock,
        ctx: &mut TxContext
    ): u64 {
        use wormhole::publish_message::{prepare_message, publish_message};

        // NOTE AGAIN: Integrators should NEVER call this within their contract.
        publish_message(
            wormhole_state,
            coin::zero(ctx),
            prepare_message(
                &mut state.emitter_cap,
                0, // Set nonce to 0, intended for batch VAAs.
                payload
            ),
            the_clock
        )
    }
}

#[test_only]
module core_messages::sender_test {
    use sui::test_scenario::{Self};
    use wormhole::wormhole_scenario::{
        return_clock,
        return_state,
        set_up_wormhole,
        take_clock,
        take_state,
        two_people,
    };

    use core_messages::sender::{
        State,
        init_with_params,
        send_message,
    };

    #[test]
    public fun test_send_message() {
        let (user, admin) = two_people();
        let my_scenario = test_scenario::begin(admin);
        let scenario = &mut my_scenario;

        // Initialize Wormhole.
        let wormhole_message_fee = 0;
        set_up_wormhole(scenario, wormhole_message_fee);

        // Initialize sender module.
        test_scenario::next_tx(scenario, admin);
        {
            let wormhole_state = take_state(scenario);
            init_with_params(&wormhole_state, test_scenario::ctx(scenario));
            return_state(wormhole_state);
        };

        // Send message as an ordinary user.
        test_scenario::next_tx(scenario, user);
        {
            let state = test_scenario::take_shared<State>(scenario);
            let wormhole_state = take_state(scenario);
            let the_clock = take_clock(scenario);

            let first_message_sequence = send_message(
                &mut state,
                &mut wormhole_state,
                b"Hello",
                &the_clock,
                test_scenario::ctx(scenario)
            );
            assert!(first_message_sequence == 0, 0);

            let second_message_sequence = send_message(
                &mut state,
                &mut wormhole_state,
                b"World",
                &the_clock,
                test_scenario::ctx(scenario)
            );
            assert!(second_message_sequence == 1, 0);

            // Clean up.
            test_scenario::return_shared(state);
            return_state(wormhole_state);
            return_clock(the_clock);
        };

        // Check effects.
        let effects = test_scenario::next_tx(scenario, user);
        assert!(test_scenario::num_user_events(&effects) == 2, 0);

        // End test.
        test_scenario::end(my_scenario);
    }
}
