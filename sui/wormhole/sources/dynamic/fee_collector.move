module wormhole::fee_collector {
    use sui::coin::{Self, Coin};
    use sui::dynamic_object_field::{Self};
    use sui::object::{UID};
    use sui::sui::{SUI};
    use sui::tx_context::{TxContext};

    const KEY: vector<u8> = b"fee_collector";

    public fun new(parent_id: &mut UID, ctx: &mut TxContext) {
        dynamic_object_field::add(parent_id, KEY, coin::zero<SUI>(ctx));
    }

    public fun deposit(parent_id: &mut UID, coin: Coin<SUI>) {
        coin::join(borrow_mut(parent_id), coin);
    }

    public fun withdraw(
        parent_id: &mut UID,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<SUI> {
        coin::split(borrow_mut(parent_id), amount, ctx)
    }

    fun borrow_mut(parent_id: &mut UID): &mut Coin<SUI> {
        dynamic_object_field::borrow_mut(parent_id, KEY)
    }
}
