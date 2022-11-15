// This module defines stores for TreasuryCaps and Coins
// owned by the Token Bridge, as well as related functions
// like minting and burning wrapped tokens.

// TODO: full support dynamic child object access pattern when available:
//       https://github.com/MystenLabs/sui/issues/4203

module token_bridge::treasury {
    use sui::tx_context::{TxContext};
    use sui::object::{Self, UID};
    use sui::coin::{Self, TreasuryCap, Coin};
    //use sui::dynamic_object_field::{Self};
    //use sui::balance::{Self};
    //use sui::transfer::{Self};

    friend token_bridge::wrapped;
    friend token_bridge::bridge_state;

    struct TreasuryCapStore<phantom CoinType> has key, store {
        id: UID,
        cap: TreasuryCap<CoinType>,
    }

    struct UnparametrizedObject has key, store {id: UID}

    struct CoinStore<phantom CoinType> has key, store {
        id: UID,
        coins: Coin<CoinType>,
    }

    public(friend) fun create_treasury_cap_store<CoinType>(cap: TreasuryCap<CoinType>, ctx: &mut TxContext): TreasuryCapStore<CoinType> { //
         TreasuryCapStore<CoinType> { id: object::new(ctx), cap: cap }
    }

    public fun deposit<CoinType>(store: &mut CoinStore<CoinType>, coin: Coin<CoinType>){
        coin::join<CoinType>(&mut store.coins, coin);
    }

    public(friend) fun withdraw<CoinType>(store: &mut CoinStore<CoinType>, value: u64, ctx: &mut TxContext): Coin<CoinType> {
        let balance = coin::balance_mut<CoinType>(&mut store.coins);
        let b = coin::take<CoinType>(balance, value, ctx);
        return b
    }

    public(friend) fun create_coin_store<CoinType>(ctx: &mut TxContext): CoinStore<CoinType> {
        CoinStore<CoinType> { id: object::new(ctx), coins: coin::zero<CoinType>(ctx) }
    }

    // This low-level mint funtion is used to implement a higher-level mint function in
    // bridge_state.move
    public(friend) fun mint<T>(
        cap_container: &mut TreasuryCapStore<T>,
        value: u64,
        ctx: &mut TxContext,
    ): Coin<T> {
        coin::mint<T>(&mut cap_container.cap, value, ctx)
    }

    public(friend) fun burn<T>(
        cap_container: &mut TreasuryCapStore<T>,
        coin: Coin<T>,
    ) {
        coin::burn<T>(&mut cap_container.cap, coin);
    }

}