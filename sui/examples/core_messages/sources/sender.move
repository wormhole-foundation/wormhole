/// A simple contracts that demonstrates how to send messages with wormhole.
module core_messages::sender {
    use sui::coin::{Self};
    use sui::object::{Self, UID};
    use sui::transfer::{Self};
    use sui::tx_context::{TxContext};

    use wormhole::state::{State as WormholeState};

    struct State has key, store {
        id: UID,
        emitter_cap: wormhole::emitter::EmitterCap,
    }

    /// Register ourselves as a wormhole emitter. This gives back an
    /// `EmitterCap` which will be required to send messages through
    /// wormhole.
    public entry fun init_with_params(
        wormhole_state: &mut WormholeState,
        ctx: &mut TxContext
    ) {
        let state = State {
            id: object::new(ctx),
            emitter_cap: wormhole::state::new_emitter(wormhole_state, ctx)
        };
        transfer::share_object(state);
    }

    public entry fun send_message_entry(
        state: &mut State,
        wormhole_state: &mut WormholeState,
        payload: vector<u8>,
        ctx: &mut TxContext
    ) {
        send_message(
            state,
            wormhole_state,
            payload,
            ctx
        );
    }

    public entry fun send_message(
        state: &mut State,
        wormhole_state: &mut WormholeState,
        payload: vector<u8>,
        ctx: &mut TxContext
    ): u64 {
        wormhole::publish_message::publish_message(
            wormhole_state,
            &mut state.emitter_cap,
            0, // Set nonce to 0, intended for batch VAAs.
            payload,
            coin::zero(ctx),
        )
    }
}

#[test_only]
module core_messages::sender_test {
    use sui::test_scenario::{
        Self,
        return_shared,
        take_shared,
    };

    use wormhole::state::{State as WormholeState};
    use wormhole::wormhole_scenario::{
        set_up_wormhole,
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
        let wormhole_message_fee = 100000000;
        set_up_wormhole(scenario, wormhole_message_fee);

        // Initialize sender module.
        test_scenario::next_tx(scenario, admin);
        {
            let wormhole_state = take_shared<WormholeState>(scenario);
            init_with_params(&mut wormhole_state, test_scenario::ctx(scenario));
            return_shared<WormholeState>(wormhole_state);
        };

        // Send message as an ordinary user.
        test_scenario::next_tx(scenario, user);
        {
            let state = take_shared<State>(scenario);
            let wormhole_state = take_shared<WormholeState>(scenario);

            let first_message_sequence = send_message(
                &mut state,
                &mut wormhole_state,
                b"Hello",
                test_scenario::ctx(scenario)
            );
            assert!(first_message_sequence == 0, 0);

            let second_message_sequence = send_message(
                &mut state,
                &mut wormhole_state,
                b"World",
                test_scenario::ctx(scenario)
            );
            assert!(second_message_sequence == 1, 0);

            // Clean up.
            return_shared<State>(state);
            return_shared<WormholeState>(wormhole_state);
        };

        // Check effects.
        let effects = test_scenario::next_tx(scenario, user);
        assert!(test_scenario::num_user_events(&effects) == 2, 0);

        // End test.
        test_scenario::end(my_scenario);
    }
}
