module wormhole::emitter {
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};

    use wormhole::external_address::{ExternalAddress};
    use wormhole::id_registry::{Self, IdRegistry};

    friend wormhole::state;

    #[test_only]
    friend wormhole::emitter_test;

    struct EmitterRegistry has store {
        registry: IdRegistry
    }

    struct EmitterCapability has key, store {
        id: UID,

        /// Unique identifier of the emitter
        addr: ExternalAddress,

        /// Sequence number of the next wormhole message
        sequence: u64
    }

    // TODO(csongor): document that this has to be globally unique.
    // The friend modifier is very important here.
    public(friend) fun new_registry(): EmitterRegistry {
        EmitterRegistry { registry: id_registry::new() }
    }

    public(friend) fun new_emitter(
        self: &mut EmitterRegistry,
        ctx: &mut TxContext
    ): EmitterCapability {
        EmitterCapability {
            id: object::new(ctx),
            addr: id_registry::next_address(&mut self.registry),
            sequence: 0
        }
    }

    /// Destroys an emitter capability.
    ///
    /// Note that this operation removes the ability to send messages using the
    /// emitter id, and is irreversible.
    public fun destroy_emitter(emitter_cap: EmitterCapability) {
        let EmitterCapability { id: id, addr: _, sequence: _ } = emitter_cap;
        object::delete(id);
    }

    /// Returns the external address of the emitter.
    ///
    /// The 16 byte (u128) emitter id left-padded to u256
    public fun get_external_address(
        emitter_cap: &EmitterCapability
    ): ExternalAddress {
        emitter_cap.addr
    }

    public(friend) fun use_sequence(emitter_cap: &mut EmitterCapability): u64 {
        let sequence = emitter_cap.sequence;
        emitter_cap.sequence = sequence + 1;
        sequence
    }

    #[test_only]
    public fun destroy_registry(registry: EmitterRegistry) {
        let EmitterRegistry { registry } = registry;
        id_registry::destroy(registry);
    }
}

#[test_only]
module wormhole::emitter_test {
    // use wormhole::emitter;
    // use sui::tx_context;

    #[test]
    public fun test_increasing_emitters() {
        // let ctx = tx_context::dummy();

        // let registry = emitter::init_emitter_registry();
        // let emitter1 = emitter::new_emitter(&mut registry, &mut ctx);
        // let emitter2 = emitter::new_emitter(&mut registry, &mut ctx);

        // assert!(emitter::get_emitter(&emitter1) == 1, 0);
        // assert!(emitter::get_emitter(&emitter2) == 2, 0);

        // emitter::destroy_emitter_cap(emitter1);
        // emitter::destroy_emitter_cap(emitter2);
        // emitter::destroy_emitter_registry(registry);
    }
}
