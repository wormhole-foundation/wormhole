module wormhole::fee_collector {
    use sui::balance::{Self, Balance};
    use sui::coin::{Self, Coin};
    use sui::sui::{SUI};
    use sui::tx_context::{TxContext};

    const E_INCORRECT_FEE: u64 = 0;

    struct FeeCollector has store {
        fee_amount: u64,
        balance: Balance<SUI>
    }

    public fun new(fee_amount: u64): FeeCollector {
        FeeCollector { fee_amount, balance: balance::zero() }
    }

    public fun fee_amount(self: &FeeCollector): u64 {
        self.fee_amount
    }

    public fun balance_value(self: &FeeCollector): u64 {
        balance::value(&self.balance)
    }

    public fun deposit(self: &mut FeeCollector, fee: Coin<SUI>) {
        assert!(coin::value(&fee) == self.fee_amount, E_INCORRECT_FEE);
        coin::put(&mut self.balance, fee);
    }

    public fun withdraw(
        self: &mut FeeCollector,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<SUI> {
        coin::take(&mut self.balance, amount, ctx)
    }
}
