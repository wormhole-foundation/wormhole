module token_bridge::native_id_registry {
    use std::vector::{Self};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{Self, ExternalAddress};

    // Needs `next_id`
    friend token_bridge::state;

    struct NativeIdRegistry has key, store {
        id: UID,
        
        /// Integer label for coin types registered with Wormhole
        index: u64
    }

    public fun new(ctx: &mut TxContext): NativeIdRegistry {
        NativeIdRegistry {
            id: object::new(ctx),
            index: 1
        }
    }

    public(friend) fun next_id(self: &mut NativeIdRegistry): ExternalAddress {
        use wormhole::bytes::serialize_u64_be;

        let cur_index = self.index;
        self.index = cur_index + 1;
        let bytes = vector::empty<u8>();
        serialize_u64_be(&mut bytes, cur_index);
        external_address::from_bytes(bytes)
    }
}
