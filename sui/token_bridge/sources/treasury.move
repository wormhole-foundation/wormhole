// This module defines stores for TreasuryCaps and Coins
// owned by the Token Bridge, as well as related functions
// like minting and burning wrapped tokens.

// TODO: full support dynamic child object access pattern when available:
//       https://github.com/MystenLabs/sui/issues/4203

module token_bridge::treasury {
    use sui::tx_context::{TxContext};
    use sui::object::{Self, UID};
    use sui::coin::{Self, TreasuryCap, Coin};
    //use sui::balance::{Self};
    use sui::transfer::{Self};

    use token_bridge::bridge_state::{BridgeState};

    friend token_bridge::wrapped;

    struct TreasuryCapStore<phantom CoinType> has key, store {
        id: UID,
        cap: TreasuryCap<CoinType>,
    }

    struct UnparametrizedObject has key, store {id: UID}

    struct CoinStore<phantom CoinType> has key, store {
        id: UID,
        coins: Coin<CoinType>,
    }

    #[test_only]
    public entry fun init_unparametrized_object(bridge_state: &mut BridgeState, ctx: &mut TxContext){
        transfer::transfer_to_object<UnparametrizedObject, BridgeState>(UnparametrizedObject{id: object::new(ctx)}, bridge_state);
    }

    public(friend) fun create_treasury_cap_store<CoinType>(bridge_state: &mut BridgeState, cap: TreasuryCap<CoinType>, ctx: &mut TxContext) { //
        let store = TreasuryCapStore<CoinType> { id: object::new(ctx), cap: cap };
        transfer::transfer_to_object<TreasuryCapStore<CoinType>,BridgeState>(store, bridge_state);
    }

    public(friend) fun create_coin_store<CoinType>(bridge_state: &mut BridgeState, ctx: &mut TxContext) {
        let store = CoinStore<CoinType> { id: object::new(ctx), coins: coin::zero<CoinType>(ctx) };
        transfer::transfer_to_object<CoinStore<CoinType>,BridgeState>(store, bridge_state);
    }

    // One can only call mint in complete_transfer when minting wrapped assets is necessary
    // TODO: get instead of passing in TreasuryCapStore, we should pass in BridgeState,
    //       and get the child TreasuryCapStore object dynamically
    public(friend) fun mint<T: drop>(
        cap_container: &mut TreasuryCapStore<T>,
        value: u64,
        ctx: &mut TxContext,
    ): Coin<T> {
        coin::mint<T>(&mut cap_container.cap, value, ctx)
    }

    public(friend) fun burn<T: drop>(
        cap_container: &mut TreasuryCapStore<T>,
        coin: Coin<T>,
    ) {
        coin::burn<T>(&mut cap_container.cap, coin);
    }

    //public(friend) fun deposit<CoinType>(_bridge_state: &mut BridgeState, _coin: Coin<CoinType>) {
        // TODO: detect if CoinStore<CoinType> exists as a child object of bridge_state
        //        if it is not a child object, initialize a CoinStore and transfer it to bridge
        //        if it is, obtain a reference to it
        // TODO: coin::join<CoinType>(&mut coin_store, coin);
    //}

    //public(friend) fun withdraw<phantom CoinType>(_bridge_state: &mut BridgeState, value: u64, ctx: &mut TxContext) { //: Coin<CoinType> {
        // TODO: detect if CoinStore<CoinType> exists as a child object of bridge_state
        //        if it is not a child object, initialize a CoinStore and transfer it to bridge
        //        if it is, obtain a reference to it

        //let balance_mut = coins::balance_mut<CoinType>(&mut store.coins, ctx);
        //coin::take<CoinType>(balance_mut, value, ctx)
    //}

}