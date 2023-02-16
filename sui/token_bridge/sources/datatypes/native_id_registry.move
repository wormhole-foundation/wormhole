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
    use sui::test_scenario::{Self, Scenario};

    use wormhole::bytes::{Self};
    use wormhole::external_address::{Self};
    use wormhole::cursor::{Self};

    use token_bridge::native_id_registry::{Self, destroy};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }


    #[test]
    fun test_native_id_registry(){
        let registry = native_id_registry::new();
        let i = 1;
        while (i < 1000){
            let addr = native_id_registry::next_id(&mut registry);
            let cursor = cursor::new<u8>(external_address::get_bytes(&addr));
            let w = bytes::deserialize_u256_be(&mut cursor);
            cursor::destroy_empty<u8>(cursor);
            assert!(w==i, 0);
            i = i + 1;
        };
        destroy(registry);
    }
}

#[test_only]
module token_bridge::native_registry_test{
    use sui::test_scenario::{Self, Scenario, next_tx, ctx};
    use sui::transfer::transfer;
    use sui::object::{Self, UID};

    use wormhole::bytes::{Self};
    use wormhole::external_address::{Self};
    use wormhole::cursor::{Self};

    use token_bridge::native_id_registry::{Self, NativeIdRegistry};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    struct MyCoinType {}

    struct RegistryContainer has key, store {id: UID, registry: NativeIdRegistry}

    #[test]
    fun test_create_token_info_1(){
        let test = scenario();
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            let registry = native_id_registry::new();
            let i = 1;
            while (i < 1000){
                let addr = native_id_registry::next_id(&mut registry);
                let cursor = cursor::new<u8>(external_address::get_bytes(&addr));
                let w = bytes::deserialize_u256_be(&mut cursor);
                cursor::destroy_empty<u8>(cursor);
                assert!(w==i, 0);
                i = i + 1;
            };
            transfer(RegistryContainer{id: object::new(ctx(&mut test)), registry: registry}, admin);
        };
        test_scenario::end(test);
    }
}
