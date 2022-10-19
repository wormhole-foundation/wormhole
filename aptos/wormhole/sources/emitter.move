/// TODO(csongor): document
/// TODO(csongor): should we rename emitter to something else? It's used in a
/// couple of places to authenticate contracts
module wormhole::emitter {

    use wormhole::serialize;
    use wormhole::external_address::{Self, ExternalAddress};

    friend wormhole::state;
    friend wormhole::wormhole;

    #[test_only]
    friend wormhole::emitter_test;

    const E_INVALID_EMITTER_UPGRADE: u64 = 0;

    struct EmitterRegistry has store {
        next_id: u64
    }

    // TODO(csongor): document that this has to be globally unique.
    // The friend modifier is very important here.
    public(friend) fun init_emitter_registry(): EmitterRegistry {
        EmitterRegistry { next_id: 1 }
    }

    #[test_only]
    public fun destroy_emitter_registry(registry: EmitterRegistry) {
        let EmitterRegistry { next_id: _ } = registry;
    }

    public(friend) fun new_emitter(registry: &mut EmitterRegistry): EmitterCapability {
        let emitter = registry.next_id;
        registry.next_id = emitter + 1;
        EmitterCapability { emitter, sequence: 0 }
    }

    struct EmitterCapability has store {
        /// Unique identifier of the emitter
        emitter: u64,
        /// Sequence number of the next wormhole message
        sequence: u64
    }

    /// Destroys an emitter capability.
    ///
    /// Note that this operation removes the ability to send messages using the
    /// emitter id, and is irreversible.
    public fun destroy_emitter_cap(emitter_cap: EmitterCapability) {
        let EmitterCapability { emitter: _, sequence: _ } = emitter_cap;
    }

    public fun get_emitter(emitter_cap: &EmitterCapability): u64 {
        emitter_cap.emitter
    }

    /// Returns the external address of the emitter.
    ///
    /// The 16 byte (u128) emitter id left-padded to u256
    public fun get_external_address(emitter_cap: &EmitterCapability): ExternalAddress {
        let emitter_bytes = vector<u8>[];
        serialize::serialize_u64(&mut emitter_bytes, emitter_cap.emitter);
        external_address::from_bytes(emitter_bytes)
    }

    public(friend) fun use_sequence(emitter_cap: &mut EmitterCapability): u64 {
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

        assert!(emitter::get_emitter(&emitter1) == 1, 0);
        assert!(emitter::get_emitter(&emitter2) == 2, 0);

        emitter::destroy_emitter_cap(emitter1);
        emitter::destroy_emitter_cap(emitter2);
        emitter::destroy_emitter_registry(registry);
    }
}
