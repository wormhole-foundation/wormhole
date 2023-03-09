module wormhole::id_registry {
    use wormhole::external_address::{Self, ExternalAddress};

    /// Resource to keep track of an increasing `index` value, which is used to
    /// generate a new `ExternalAddress`.
    struct IdRegistry has store {
        index: u64
    }

    public fun new(): IdRegistry {
        IdRegistry { index: 0 }
    }

    public fun next_address(self: &mut IdRegistry): ExternalAddress {
        self.index = self.index + 1;
        external_address::from_u64_be(self.index)
    }

    #[test_only]
    public fun destroy(registry: IdRegistry) {
        let IdRegistry { index: _ } = registry;
    }

    #[test_only]
    public fun skip_to(self: &mut IdRegistry, value: u64) {
        self.index = value;
    }

    #[test_only]
    public fun index(self: &IdRegistry): u64 {
        self.index
    }

}

#[test_only]
module wormhole::id_registry_tests {
    use wormhole::bytes32::{Self};
    use wormhole::id_registry::{Self};
    use wormhole::external_address::{Self};

    #[test]
    fun test_native_id_registry() {
        let registry = id_registry::new();
        let i = 0;
        assert!(id_registry::index(&registry) == i, 0);

        // Generate multiple IDs using `next_address` and check that they are
        // indeed generated in monotonic increasing order in increments of one.
        while (i < 10) {
            let left = external_address::to_bytes32(
                id_registry::next_address(&mut registry)
            );

            i = i + 1;
            assert!(id_registry::index(&registry) == i, 0);

            let right = bytes32::from_u64_be(i);
            assert!(left == right, 0);
        };

        // Skip ahead by some arbitrary amount and repeat.
        let i = 1000;
        id_registry::skip_to(&mut registry, i);
        while (i < 10) {
            let left = external_address::to_bytes32(
                id_registry::next_address(&mut registry)
            );

            i = i + 1;
            assert!(id_registry::index(&registry) == i, 0);

            let right = bytes32::from_u64_be(i);
            assert!(left == right, 0);
        };

        // Clean up.
        id_registry::destroy(registry);
    }
}
