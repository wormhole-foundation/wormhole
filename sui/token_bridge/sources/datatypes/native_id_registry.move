module token_bridge::native_id_registry {
    use std::vector::{Self};
    use wormhole::external_address::{Self, ExternalAddress};

    // Needs `next_id`
    friend token_bridge::state;

    struct NativeIdRegistry has store {
        /// Integer label for coin types registered with Wormhole
        index: u64
    }

    public fun new(): NativeIdRegistry {
        NativeIdRegistry {
            index: 1
        }
    }

    public fun next_id(self: &mut NativeIdRegistry): ExternalAddress {
        use wormhole::bytes::serialize_u64_be;

        let bytes = vector::empty<u8>();
        serialize_u64_be(&mut bytes, self.index);

        self.index = self.index + 1;
        external_address::from_bytes(bytes)
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
        // generate a large number of IDs using native_id_registry::next_id
        // and check that they are indeed generated in monotonic increasing
        // order in increments of one
        while (i < 2000){
            let addr = native_id_registry::next_id(&mut registry);
            let cursor = cursor::new<u8>(external_address::get_bytes(&addr));

            // deserialize the 32-byte representation of the ID into an integer
            let w = bytes::deserialize_u256_be(&mut cursor);
            cursor::destroy_empty<u8>(cursor);
            assert!(w==i, 0);
            i = i + 1;
        };
        destroy(registry);
    }
}
