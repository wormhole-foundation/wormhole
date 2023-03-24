/// A simple contracts that demonstrates how to send messages with wormhole.
module core_messages::sender {
    use sui::coin::{Self, Coin};
    use sui::object::{Self, UID};
    use sui::sui::{SUI};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use wormhole::state::{State as WormholeState};

    struct DeployerCap has key, store {
        id: UID
    }

    struct State has key, store {
        id: UID,
        emitter_cap: wormhole::emitter::EmitterCap,
    }

    fun init(ctx: &mut TxContext) {
        transfer::transfer(
            DeployerCap { id: object::new(ctx) },
            tx_context::sender(ctx)
        );
    }

    public entry fun init_with_params(
        deployer: DeployerCap,
        wormhole_state: &mut WormholeState,
        ctx: &mut TxContext
    ) {
        let DeployerCap { id } = deployer;
        object::delete(id);

        // Register ourselves as a wormhole emitter. This gives back an
        // `EmitterCap` which will be required to send messages through
        // wormhole.
        let state = State {
            id: object::new(ctx),
            emitter_cap: wormhole::state::new_emitter(wormhole_state, ctx)
        };
        transfer::transfer(state, @core_messages);
    }

    public entry fun send_message_entry(
        state: &mut State,
        wormhole_state: &mut WormholeState,
        payload: vector<u8>,
        fee_with_potential_surplus: Coin<SUI>,
        ctx: &mut TxContext
    ) {
        send_message(
            state,
            wormhole_state,
            payload,
            fee_with_potential_surplus,
            ctx
        );
    }

    public fun send_message(
        state: &mut State,
        wormhole_state: &mut WormholeState,
        payload: vector<u8>,
        fee_with_potential_surplus: Coin<SUI>,
        ctx: &mut TxContext
    ): u64 {
        let fee_amount = wormhole::state::message_fee(wormhole_state);
        let fee_coins = coin::split(
            &mut fee_with_potential_surplus,
            fee_amount,
            ctx
        );
        transfer::transfer(fee_with_potential_surplus, tx_context::sender(ctx));
        wormhole::publish_message::publish_message(
            wormhole_state,
            &mut state.emitter_cap,
            0, // Set nonce to 0, only used for batch VAAs.
            payload,
            fee_coins,
        )
    }

    #[test_only]
    public fun init_test_only(ctx: &mut TxContext) {
        init(ctx);
    }
}

#[test_only]
module core_messages::sender_test {
    use sui::coin::{Self};
    use sui::sui::{SUI};
    use sui::test_scenario::{
        Self,
        return_shared,
        return_to_address,
        take_from_address,
        take_from_sender,
        take_shared,
    };

    use wormhole::state::{State as WormholeState};
    use wormhole::wormhole_scenario::{
        set_up_wormhole,
        two_people
    };

    use core_messages::sender::{
        DeployerCap,
        State,
        init_test_only,
        init_with_params,
        send_message
    };

    #[test]
    public fun test_send_message() {
        let (user, admin) = two_people();
        let my_scenario = test_scenario::begin(admin);
        let scenario = &mut my_scenario;

        // Initialize Wormhole.
        let wormhole_message_fee = 100000000;
        set_up_wormhole(scenario, wormhole_message_fee);

        // Mock deploy sender module.
        test_scenario::next_tx(scenario, admin);
        {
            init_test_only(test_scenario::ctx(scenario));
        };

        // Initialize sender module.
        test_scenario::next_tx(scenario, admin);
        {
            let wormhole_state = take_shared<WormholeState>(scenario);
            let deployer_cap = take_from_sender<DeployerCap>(scenario);

            init_with_params(
                deployer_cap,
                &mut wormhole_state,
                test_scenario::ctx(scenario)
            );

            test_scenario::return_shared<WormholeState>(wormhole_state);
        };

        // Send message as an ordinary user.
        test_scenario::next_tx(scenario, user);
        {
            let state = take_from_address<State>(scenario, @core_messages);
            let wormhole_state = take_shared<WormholeState>(scenario);

            let first_message_sequence = send_message(
                &mut state,
                &mut wormhole_state,
                b"Hello",
                coin::mint_for_testing<SUI>(
                    wormhole_message_fee,
                    test_scenario::ctx(scenario)
                ),
                test_scenario::ctx(scenario)
            );
            assert!(first_message_sequence == 0, 0);

            let second_message_sequence = send_message(
                &mut state,
                &mut wormhole_state,
                b"World",
                coin::mint_for_testing<SUI>(
                    wormhole_message_fee,
                    test_scenario::ctx(scenario)
                ),
                test_scenario::ctx(scenario)
            );
            assert!(second_message_sequence == 1, 0);

            // Clean up.
            return_to_address<State>(@core_messages, state);
            return_shared<WormholeState>(wormhole_state);
        };

        // Check effects.
        let effects = test_scenario::next_tx(scenario, user);
        assert!(test_scenario::num_user_events(&effects) == 2, 0);

        // End test.
        test_scenario::end(my_scenario);
    }
}
