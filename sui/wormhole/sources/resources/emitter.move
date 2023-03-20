// SPDX-License-Identifier: Apache 2

/// This module implements a capability (`EmitterCap`), which allows one to send
/// Wormhole messages, and a registry (`EmitterRegistry`), which provides a
/// mechanism to generate new emitters. The capability's address is derived by
/// an arbitrary index value (warehoused by `IdRegistry` found in
/// `wormhole::id_registry`).
module wormhole::emitter {
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};

    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::id_registry::{Self, IdRegistry};

    // Needs `new_registry`.
    friend wormhole::state;

    /// `EmitterRegistry` keeps track of auto-assigned IDs using the
    /// `IdRegistry` resource. For every new `EmitterCap` created, its
    /// registry will use the current index to generate an `ExternalAddress`,
    /// then uptick its index.
    ///
    /// Other contracts can leverage this registry (instead of having to
    /// implement his own) for their own cross-chain emitter registry. An
    /// example of this is Token Bridge's contract-controlled transfer methods.
    struct EmitterRegistry has store {
        registry: IdRegistry
    }

    /// `EmitterCap` is a Sui object that gives a user or smart contract the
    /// capability to send Wormhole messages. For every Wormhole message
    /// emitted, a unique `sequence` is used.
    struct EmitterCap has key, store {
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

    public fun registry_index(self: &EmitterRegistry): u256 {
        id_registry::index(&self.registry)
    }

    /// Generate a new `EmitterCap` via the registry.
    public fun new_cap(
        self: &mut EmitterRegistry,
        ctx: &mut TxContext
    ): EmitterCap {
        EmitterCap {
            id: object::new(ctx),
            addr: id_registry::next_address(&mut self.registry),
            sequence: 0
        }
    }

    /// Returns the `ExternalAddress` of the emitter (32-bytes).
    public fun addr(cap: &EmitterCap): ExternalAddress {
        cap.addr
    }

    /// Returns current sequence (which will be used in the next Wormhole
    /// message emitted).
    public fun sequence(cap: &EmitterCap): u64 {
        cap.sequence
    }

    /// Destroys an `EmitterCap`.
    ///
    /// Note that this operation removes the ability to send messages using the
    /// emitter id, and is irreversible.
    public fun destroy_cap(cap: EmitterCap) {
        let EmitterCap { id, addr: _, sequence: _ } = cap;
        object::delete(id);
    }

    /// Returns the address of the emitter as 32-element vector<u8>.
    public fun emitter_address(cap: &EmitterCap): vector<u8> {
        external_address::to_bytes(cap.addr)
    }

    /// Once a Wormhole message is emitted, an `EmitterCap` upticks its
    /// internal `sequence` for the next message.
    public(friend) fun use_sequence(cap: &mut EmitterCap): u64 {
        let sequence = cap.sequence;
        cap.sequence = sequence + 1;
        sequence
    }

    #[test_only]
    public fun destroy_registry(registry: EmitterRegistry) {
        let EmitterRegistry { registry } = registry;
        id_registry::destroy(registry);
    }

    #[test_only]
    public fun skip_to(self: &mut EmitterRegistry, value: u256) {
        id_registry::skip_to(&mut self.registry, value);
    }
}

#[test_only]
module wormhole::emitter_tests {
    use sui::tx_context::{Self};

    use wormhole::emitter::{Self};

    #[test]
    public fun test_emitter_registry_and_capability() {
        let ctx = &mut tx_context::dummy();

        let registry = emitter::new_registry_test_only();
        assert!(emitter::registry_index(&registry) == 1, 0);

        // Generate new emitter and check that the registry value upticked.
        let cap = emitter::new_cap(&mut registry, ctx);
        assert!(emitter::registry_index(&registry) == 2, 0);

        // And check emitter cap's address.
        let expected =
            x"0000000000000000000000000000000000000000000000000000000000000001";
        assert!(emitter::emitter_address(&cap) == expected, 0);
        emitter::destroy_cap(cap);

        // Skip ahead to ID = 256, create new emitter and check registry value
        // again.
        emitter::skip_to(&mut registry, 257);
        let cap = emitter::new_cap(&mut registry, ctx);
        assert!(emitter::registry_index(&registry) == 258, 0);

        // And check emitter cap's address.
        let expected =
            x"0000000000000000000000000000000000000000000000000000000000000101";
        assert!(emitter::emitter_address(&cap) == expected, 0);
        emitter::destroy_cap(cap);

        // Clean up.
        emitter::destroy_registry(registry);
    }
}
