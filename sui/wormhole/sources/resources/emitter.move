module wormhole::emitter {
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};

    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::id_registry::{Self, IdRegistry};

    friend wormhole::state;

    /// `EmitterRegistry` keeps track of auto-assigned IDs using the
    /// `IdRegistry` resource. For every new `EmitterCapability` created, its
    /// registry will uptick an internal value for the next call to
    /// `new_emitter`.
    ///
    /// Other contracts can leverage this registry (instead of having to
    /// implement his own) for their own cross-chain emitter registry. An
    /// example of this is Token Bridge's contract-controlled transfer methods.
    struct EmitterRegistry has store {
        registry: IdRegistry
    }

    /// `EmitterCapability` gives a user or smart contract the capability to
    /// send Wormhole messages. For every Wormhole message emitted, a unique
    /// `sequence` is used.
    struct EmitterCapability has key, store {
        id: UID,

        /// Unique identifier of the emitter
        addr: ExternalAddress,

        /// Sequence number of the next wormhole message
        sequence: u64
    }

    /// We only allow `wormhole::state` to call this method. The `State` is
    /// the sole controller of the `EmitterRegistry`.
    public(friend) fun new_registry(): EmitterRegistry {
        EmitterRegistry { registry: id_registry::new() }
    }

    #[test_only]
    public fun new_registry_test_only(): EmitterRegistry {
        new_registry()
    }

    public fun registry_index(self: &EmitterRegistry): u64 {
        id_registry::index(&self.registry)
    }

    /// Generate a new `EmitterCapability` via the registry.
    public fun new_emitter(
        self: &mut EmitterRegistry,
        ctx: &mut TxContext
    ): EmitterCapability {
        EmitterCapability {
            id: object::new(ctx),
            addr: id_registry::next_address(&mut self.registry),
            sequence: 0
        }
    }

    /// Destroys an `EmitterCapability`.
    ///
    /// Note that this operation removes the ability to send messages using the
    /// emitter id, and is irreversible.
    public fun destroy_emitter(emitter_cap: EmitterCapability) {
        let EmitterCapability { id, addr: _, sequence: _ } = emitter_cap;
        object::delete(id);
    }

    /// Returns the `ExternalAddress` of the emitter (32-bytes).
    public fun external_address(
        emitter_cap: &EmitterCapability
    ): ExternalAddress {
        emitter_cap.addr
    }

    /// Returns the address of the emitter as 32-element vector<u8>.
    public fun emitter_address(
        emitter_cap: &EmitterCapability
    ): vector<u8> {
        external_address::to_bytes(emitter_cap.addr)
    }

    /// Once a Wormhole message is emitted, an `EmitterCapability` upticks its
    /// internal `sequence` for the next message.
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

    #[test_only]
    public fun skip_to(self: &mut EmitterRegistry, value: u64) {
        id_registry::skip_to(&mut self.registry, value);
    }
}

#[test_only]
module wormhole::emitter_test {
    use sui::tx_context::{Self};

    use wormhole::emitter::{Self};

    #[test]
    public fun test_emitter_registry_and_capability() {
        let ctx = &mut tx_context::dummy();

        let registry = emitter::new_registry_test_only();
        assert!(emitter::registry_index(&registry) == 0, 0);

        // Generate new emitter and check that the registry value upticked.
        let cap = emitter::new_emitter(&mut registry, ctx);
        assert!(emitter::registry_index(&registry) == 1, 0);

        // And check emitter cap's address.
        let expected =
            x"0000000000000000000000000000000000000000000000000000000000000001";
        assert!(emitter::emitter_address(&cap) == expected, 0);
        emitter::destroy_emitter(cap);

        // Skip ahead to ID = 256, create new emitter and check registry value
        // again.
        emitter::skip_to(&mut registry, 256);
        let cap = emitter::new_emitter(&mut registry, ctx);
        assert!(emitter::registry_index(&registry) == 257, 0);

        // And check emitter cap's address.
        let expected =
            x"0000000000000000000000000000000000000000000000000000000000000101";
        assert!(emitter::emitter_address(&cap) == expected, 0);
        emitter::destroy_emitter(cap);

        // Clean up.
        emitter::destroy_registry(registry);
    }
}
