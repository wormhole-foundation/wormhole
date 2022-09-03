/// A simple contracts that demonstrates how to send messages with wormhole.
module core_messages::sender {
    use wormhole::wormhole;

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

    public entry fun send_message(payload: vector<u8>) acquires State {
        // Retrieve emitter capability from the state
        let emitter_cap = &mut borrow_global_mut<State>(@core_messages).emitter_cap;

        // Set nonce to 0 (this field is not interesting for regular messages,
        // only batch VAAs)
        let nonce: u64 = 0;

        // How many block confirmations to wait before observing this message.
        // Since aptos has instant-finality, 0 is fine.
        let consistency_level: u8 = 0;

        wormhole::publish_message(
            emitter_cap,
            nonce,
            payload,
            consistency_level
        )
    }
}
