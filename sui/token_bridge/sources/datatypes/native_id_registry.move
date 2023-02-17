module token_bridge::native_id_registry {
    use wormhole::external_address::{Self, ExternalAddress};

    struct NativeIdRegistry has store {
        /// Integer label for coin types registered with Wormhole.
        index: u64
    }

    public fun new(): NativeIdRegistry {
        NativeIdRegistry { index: 0 }
    }

    public fun next_id(self: &mut NativeIdRegistry): ExternalAddress {
        self.index = self.index + 1;
        external_address::from_u64_be(self.index)
    }

    #[test_only]
    public fun destroy(r: NativeIdRegistry): u64 {
        let NativeIdRegistry { index } = r;
        return index
    }
}

#[test_only]
module token_bridge::native_registry_test{
    use wormhole::bytes::{Self};
    use wormhole::external_address::{Self};
    use wormhole::cursor::{Self};

    use token_bridge::native_id_registry::{Self, destroy};

    #[test]
    fun test_native_id_registry(){
        let registry = native_id_registry::new();
        let i = 1;
        // Generate a large number of IDs using native_id_registry::next_id
        // and check that they are indeed generated in monotonic increasing
        // order in increments of one.
        while (i < 2000){
            let addr = native_id_registry::next_id(&mut registry);
            let cursor = cursor::new<u8>(external_address::to_bytes(addr));

            // Deserialize the 32-byte representation of the ID into an integer.
            let w = bytes::deserialize_u256_be(&mut cursor);
            cursor::destroy_empty<u8>(cursor);
            assert!(w==i, 0);
            i = i + 1;
        };
        destroy(registry);
    }
}
