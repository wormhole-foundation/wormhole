/// A simple contracts that demonstrates how to send messages with wormhole.
module core_messages::sender {
    use wormhole::wormhole;
    use aptos_framework::coin;

    struct State has key {
        emitter_cap: wormhole::emitter::EmitterCapability,
    }

    entry fun init_module(core_messages: &signer) {
        // Register ourselves as a wormhole emitter. This gives back an
        // `EmitterCapability` which will be required to send messages through
        // wormhole.
        let emitter_cap = wormhole::register_emitter();
        move_to(core_messages, State { emitter_cap });
    }

    #[test_only]
    /// Initialise module for testing.
    public fun init_module_test() {
        use aptos_framework::account;
        // recover the signer for the module's account
        let signer_cap = account::create_test_signer_cap(@core_messages);
        let signer = account::create_signer_with_capability(&signer_cap);
        // then call the initialiser
        init_module(&signer)
    }

    public entry fun send_message(user: &signer, payload: vector<u8>) acquires State {
        // Retrieve emitter capability from the state
        let emitter_cap = &mut borrow_global_mut<State>(@core_messages).emitter_cap;

        // Set nonce to 0 (this field is not interesting for regular messages,
        // only batch VAAs)
        let nonce: u64 = 0;

        let message_fee = wormhole::state::get_message_fee();
        let fee_coins = coin::withdraw(user, message_fee);

        let _sequence = wormhole::publish_message(
            emitter_cap,
            nonce,
            payload,
            fee_coins
        );
    }
}

#[test_only]
module core_messages::sender_test {
    use wormhole::wormhole;
    use core_messages::sender;
    use aptos_framework::account;
    use aptos_framework::aptos_coin::{Self, AptosCoin};
    use aptos_framework::coin;
    use aptos_framework::signer;
    use aptos_framework::timestamp;

    #[test(aptos_framework = @aptos_framework, user = @0x111)]
    public fun test_send_message(aptos_framework: &signer, user: &signer) {
        let message_fee = 100;
        timestamp::set_time_has_started_for_testing(aptos_framework);
        wormhole::init_test(
            22,
            1,
            x"0000000000000000000000000000000000000000000000000000000000000004",
            x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
            message_fee
        );
        sender::init_module_test();

        let (burn_cap, mint_cap) = aptos_coin::initialize_for_test(aptos_framework);

        // create user account and airdrop coins
        account::create_account_for_test(signer::address_of(user));
        coin::register<AptosCoin>(user);
        coin::deposit(signer::address_of(user), coin::mint(message_fee, &mint_cap));

        sender::send_message(user, b"hi mom");

        coin::destroy_mint_cap(mint_cap);
        coin::destroy_burn_cap(burn_cap);
    }
}
