module wormhole::id_registry {
    use wormhole::external_address::{Self, ExternalAddress};

    struct IdRegistry has store {
        /// Integer label for coin types registered with Wormhole
        index: u64
    }

    public fun new(): IdRegistry {
        IdRegistry { index: 0 }
    }

    public fun value(self: &IdRegistry): u64 {
        self.index
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
}

#[test_only]
module wormhole::id_registry_test{
    use wormhole::bytes::{Self};
    use wormhole::external_address::{Self};
    use wormhole::cursor::{Self};

    use wormhole::id_registry::{Self, destroy};

    #[test]
    fun test_native_id_registry(){
        let registry = id_registry::new();
        let i = 1;
        // generate a large number of IDs using native_id_registry::next_id
        // and check that they are indeed generated in monotonic increasing
        // order in increments of one
        while (i < 2000){
            let addr = id_registry::next_address(&mut registry);
            let cursor = cursor::new<u8>(external_address::to_bytes(addr));

            // deserialize the 32-byte representation of the ID into an integer
            let w = bytes::deserialize_u256_be(&mut cursor);
            cursor::destroy_empty<u8>(cursor);
            assert!(w==i, 0);
            i = i + 1;
        };
        destroy(registry);
    }
}
