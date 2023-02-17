module wormhole::fee_collector {
    use sui::coin::{Self, Coin};
    use sui::sui::{SUI};
    use sui::tx_context::{TxContext};

    const E_INCORRECT_FEE: u64 = 0;

    struct FeeCollector has store {
        amount: u64,
        coin: Coin<SUI>
    }

    public fun new(amount: u64, ctx: &mut TxContext): FeeCollector {
        FeeCollector { amount, coin: coin::zero(ctx) }
    }

    public fun amount(self: &FeeCollector): u64 {
        self.amount
    }

    public fun coin_value(self: &FeeCollector): u64 {
        coin::value(&self.coin)
    }

    public fun deposit(self: &mut FeeCollector, fee: Coin<SUI>) {
        assert!(coin::value(&fee) == self.amount, E_INCORRECT_FEE);
        coin::join(&mut self.coin, fee);
    }

    public fun withdraw(
        self: &mut FeeCollector,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<SUI> {
        coin::split(&mut self.coin, amount, ctx)
    }
}
