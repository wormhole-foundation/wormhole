/// This module implements a container for foreign emitters registered with the
/// Token Bridge. Token Bridge only cares about other Token Bridge contracts on
/// other networks.
///
/// NOTE: Once an emitter is registered, it cannot be removed or updated.
module token_bridge::emitter_registry {
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};

    /// Emitter for a chain ID already exists.
    const E_ALREADY_REGISTERED: u64 = 0;
    /// Emitter does not exist for chain ID.
    const E_UNREGISTERED: u64 = 1;
    /// Emitter for chain ID == 0 is not valid.
    const E_INVALID_CHAIN: u64 = 2;

    /// Container for chain ID (`u16`) to emitter address (`ExternalAddress`)
    /// mapping.
    struct EmitterRegistry has store {
        emitters: Table<u16, ExternalAddress>
    }

    /// Create new `EmitterRegistry`.
    public fun new(ctx: &mut TxContext): EmitterRegistry {
        EmitterRegistry { emitters: table::new(ctx) }
    }

    /// Add new emitter address for an unregistered chain ID.
    public fun add(
        self: &mut EmitterRegistry,
        chain: u16,
        emitter_addr: ExternalAddress
    ) {
        assert!(chain != 0, E_INVALID_CHAIN);
        assert!(!table::contains(&self.emitters, chain), E_ALREADY_REGISTERED);
        table::add(&mut self.emitters, chain, emitter_addr);
    }

    /// Retrieve emitter address for given chain ID.
    public fun emitter_address(
        self: &EmitterRegistry,
        chain: u16
    ): ExternalAddress {
        let emitters = &self.emitters;
        assert!(table::contains(emitters, chain), E_UNREGISTERED);
        *table::borrow(emitters, chain)
    }

    #[test_only]
    public fun destroy(registry: EmitterRegistry) {
        let EmitterRegistry { emitters } = registry;
        table::drop(emitters);
    }
}

#[test_only]
module token_bridge::emitter_registry_tests {
    use sui::tx_context::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::external_address::{Self};

    use token_bridge::emitter_registry::{Self};

    #[test]
    fun test_emitter_registry() {
        let ctx = &mut tx_context::dummy();
        let registry = emitter_registry::new(ctx);

        // Add many chain ID -> emitter address key-value pairs to the emitter
        // registry.
        let (start, n) = (1, 10);
        let i = start;
        while (i <= n) {
            let emitter_addr =
                external_address::new(bytes32::from_u256_be((i as u256)));
            emitter_registry::add(&mut registry, i, emitter_addr);
            i = i + 1;
        };

        // Check that added emitters are accessible.
        let i = start;
        while (i <= n) {
            let addr = emitter_registry::emitter_address(&registry, i);
            let expected =
                external_address::new(bytes32::from_u256_be((i as u256)));
            assert!(addr == expected, 0);
            i = i + 1;
        };

        // Clean up.
        emitter_registry::destroy(registry);
    }

    #[test]
    #[expected_failure(abort_code = emitter_registry::E_ALREADY_REGISTERED)]
    public fun test_cannot_add_already_registered() {
        let ctx = &mut tx_context::dummy();
        let registry = emitter_registry::new(ctx);

        // Add unregistered emitter.
        let chain = 42;
        let emitter_addr =
            external_address::from_bytes(
                x"deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
            );
        emitter_registry::add(&mut registry, chain, emitter_addr);

        // Make another address for the heck of it.
        let another_addr =
            external_address::from_bytes(
                x"feedbeeffeedbeeffeedbeeffeedbeeffeedbeeffeedbeeffeedbeeffeedbeef"
            );

        // You shall not pass!
        emitter_registry::add(&mut registry, chain, another_addr);

        // Clean up.
        emitter_registry::destroy(registry);
    }

    #[test]
    #[expected_failure(abort_code = emitter_registry::E_INVALID_CHAIN)]
    public fun test_cannot_add_chain_zero() {
        let ctx = &mut tx_context::dummy();
        let registry = emitter_registry::new(ctx);

        // You shall not pass!
        let chain = 0;
        let emitter_addr = external_address::new(bytes32::default());
        emitter_registry::add(&mut registry, chain, emitter_addr);

        // Clean up.
        emitter_registry::destroy(registry);
    }

    #[test]
    #[expected_failure(abort_code = emitter_registry::E_UNREGISTERED)]
    public fun test_cannot_get_emitter_address_unregistered() {
        let ctx = &mut tx_context::dummy();
        let registry = emitter_registry::new(ctx);

        // Add an emitter.
        let chain = 42;
        let emitter_addr = external_address::new(bytes32::default());
        emitter_registry::add(&mut registry, chain, emitter_addr);

        // You shall not pass!
        emitter_registry::emitter_address(&registry, chain + 1);

        // Clean up.
        emitter_registry::destroy(registry);
    }
}
