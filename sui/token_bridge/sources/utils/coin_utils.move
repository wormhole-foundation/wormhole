// SPDX-License-Identifier: Apache 2

/// This module implements utilities helpful for outbound token transfers. These
/// utility methods should also help avoid having to work around conversions
/// between `Coin` and `Balance` avoiding unnecessary object creation and
/// destruction.
module token_bridge::coin_utils {
    use sui::balance::{Self, Balance};
    use sui::coin::{Self, Coin};
    use sui::tx_context::{TxContext};

    /// Method similar to `coin::take` where an amount is split from a `Coin`
    /// object's inner balance.
    public fun take_balance<C>(
        coin_mut: &mut Coin<C>,
        amount: u64
    ): Balance<C> {
        balance::split(coin::balance_mut(coin_mut), amount)
    }

    /// Method out of convenience to take the full balance value out of a `Coin`
    /// object while preserving that object. This method is used to avoid
    /// calling `coin::into_balance` which destroys the object.
    public fun take_full_balance<C>(coin_mut: &mut Coin<C>): Balance<C> {
        let amount = coin::value(coin_mut);
        take_balance(coin_mut, amount)
    }

    /// Method similar to `coin::put` where an outside balance is joined with
    /// an existing `Coin` object.
    public fun put_balance<C>(
        coin_mut: &mut Coin<C>,
        the_balance: Balance<C>
    ): u64 {
        balance::join(coin::balance_mut(coin_mut), the_balance)
    }

    /// Method for those integrators that use `Coin` objects, where `the_coin`
    /// will be destroyed if the value is zero. Otherwise it will be returned
    /// back to the transaction sender.
    public fun return_nonzero<C>(the_coin: Coin<C>, ctx: &TxContext) {
        if (coin::value(&the_coin) == 0) {
            coin::destroy_zero(the_coin);
        } else {
            sui::pay::keep(the_coin, ctx)
        }
    }
}
