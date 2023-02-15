module token_bridge::registered_emitters {
    use sui::dynamic_object_field::{Self};
    use sui::object::{UID};
    use sui::table::{Self};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};

    const KEY: vector<u8> = b"registered_emitters";

    public fun new(parent_id: &mut UID, ctx: &mut TxContext) {
        dynamic_object_field::add(
            parent_id,
            KEY,
            table::new<u16, ExternalAddress>(ctx)
        )
    }

    public fun add(parent_id: &mut UID, chain: u16, addr: ExternalAddress) {
        table::add(
            dynamic_object_field::borrow_mut(parent_id, KEY),
            chain,
            addr
        )
    }

    public fun has(parent_id: &UID, chain: u16): bool {
        let ref = dynamic_object_field::borrow(parent_id, KEY);
        table::contains<u16, ExternalAddress>(ref, chain)
    }

    public fun external_address(parent_id: &UID, chain: u16): ExternalAddress {
        *table::borrow(dynamic_object_field::borrow(parent_id, KEY), chain)
    }
}
