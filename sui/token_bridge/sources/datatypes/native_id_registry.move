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
}
