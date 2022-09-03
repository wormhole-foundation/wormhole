/// TODO(csongor): document
module wormhole::emitter {

    friend wormhole::state;

    #[test_only]
    friend wormhole::emitter_test;

    struct EmitterRegistry has store {
        next_id: u128
    }

    // TODO(csongor): document that this has to be globally unique.
    // The friend modifier is very important here.
    public(friend) fun init_emitter_registry(): EmitterRegistry {
        EmitterRegistry { next_id: 0 }
    }

    #[test_only]
    public fun destroy_emitter_registry(registry: EmitterRegistry) {
        let EmitterRegistry { next_id: _ } = registry;
    }

    public fun new_emitter(registry: &mut EmitterRegistry): EmitterCapability {
        let emitter = registry.next_id;
        registry.next_id = emitter + 1;
        EmitterCapability { emitter, sequence: 0 }
    }

    struct EmitterCapability has store {
        emitter: u128,
        sequence: u64
    }

    public fun destroy_emitter_cap(emitter_cap: EmitterCapability) {
        let EmitterCapability { emitter: _, sequence: _ } = emitter_cap;
    }

    public fun get_emitter(emitter_cap: &EmitterCapability): u128 {
        emitter_cap.emitter
    }

    public fun use_sequence(emitter_cap: &mut EmitterCapability): u64 {
        let sequence = emitter_cap.sequence;
        emitter_cap.sequence = sequence + 1;
        sequence
    }

}

#[test_only]
module wormhole::emitter_test {
    use wormhole::emitter;

    #[test]
    public fun test_increasing_emitters() {
        let registry = emitter::init_emitter_registry();
        let emitter1 = emitter::new_emitter(&mut registry);
        let emitter2 = emitter::new_emitter(&mut registry);

        assert!(emitter::get_emitter(&emitter1) == 0, 0);
        assert!(emitter::get_emitter(&emitter2) == 1, 0);

        emitter::destroy_emitter_cap(emitter1);
        emitter::destroy_emitter_cap(emitter2);
        emitter::destroy_emitter_registry(registry);
    }
}
