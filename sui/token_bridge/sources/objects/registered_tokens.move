module token_bridge::registered_tokens {
    use sui::coin::{Coin, TreasuryCap};
    use sui::dynamic_field::{Self};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{chain_id};

    use token_bridge::native_asset::{Self, NativeAsset};
    use token_bridge::native_id_registry::{Self, NativeIdRegistry};
    use token_bridge::token_info::{TokenInfo};
    use token_bridge::wrapped_asset::{Self, WrappedAsset};

    friend token_bridge::state;
    #[test_only]
    friend token_bridge::registered_tokens_test;

    const E_UNREGISTERED: u64 = 0;
    const E_ALREADY_REGISTERED: u64 = 1;
    const E_CANNOT_DEPOSIT_WRAPPED_COIN: u64 = 2;

    struct RegisteredTokens has key, store {
        id: UID,
        native_id_registry: NativeIdRegistry,
        num_wrapped: u64,
        num_native: u64
    }

    struct Key<phantom C> has copy, drop, store {}

    public fun new(ctx: &mut TxContext): RegisteredTokens {
        RegisteredTokens {
            id: object::new(ctx),
            native_id_registry: native_id_registry::new(),
            num_wrapped: 0,
            num_native: 0
        }
    }

    #[test_only]
    public fun destroy(r: RegisteredTokens){
        let RegisteredTokens {
            id: id,
            native_id_registry: native_id_registry,
            num_wrapped: _,
            num_native: _
        } = r;
        object::delete(id);
        native_id_registry::destroy(native_id_registry);
    }

    public fun num_native(self: &RegisteredTokens): u64 {
        self.num_native
    }

    public fun num_wrapped(self: &RegisteredTokens): u64 {
        self.num_wrapped
    }

    public fun has<C>(self: &RegisteredTokens): bool {
        dynamic_field::exists_(&self.id, Key<C>{})
    }

    public fun is_wrapped<C>(self: &RegisteredTokens): bool {
        assert!(has<C>(self), E_UNREGISTERED);
        dynamic_field::exists_with_type<Key<C>, WrappedAsset<C>>(
            &self.id,
            Key<C>{}
        )
    }

    public fun is_native<C>(self: &RegisteredTokens): bool {
        // `is_wrapped` asserts that `C` is registered. So if `C` is not
        // wrapped, then it is native.
        !is_wrapped<C>(self)
    }

    public(friend) fun add_new_wrapped<C>(
        self: &mut RegisteredTokens,
        chain: u16,
        addr: ExternalAddress,
        treasury_cap: TreasuryCap<C>,
        decimals: u8,
    ) {
        assert!(!has<C>(self), E_ALREADY_REGISTERED);
        add_wrapped<C>(
            self,
            wrapped_asset::new(chain, addr, treasury_cap, decimals)
        )
    }

    public(friend) fun add_new_native<C>(
        self: &mut RegisteredTokens,
        decimals: u8,
        ctx: &mut TxContext
    ) {
        assert!(!has<C>(self), E_ALREADY_REGISTERED);
        let addr = native_id_registry::next_id(&mut self.native_id_registry);
        add_native<C>(
            self,
            native_asset::new(addr, decimals, ctx)
        )
    }

    public(friend) fun burn<C>(
        self: &mut RegisteredTokens,
        coin: Coin<C>
    ): u64 {
        wrapped_asset::burn(
            dynamic_field::borrow_mut(&mut self.id, Key<C>{}),
            coin
        )
    }

    public(friend) fun mint<C>(
        self: &mut RegisteredTokens,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<C> {
        wrapped_asset::mint(
            dynamic_field::borrow_mut(&mut self.id, Key<C>{}),
            amount,
            ctx
        )
    }

    public(friend) fun deposit<C>(
        self: &mut RegisteredTokens,
        some_coin: Coin<C>
    ) {
        assert!(is_native<C>(self), E_CANNOT_DEPOSIT_WRAPPED_COIN);
        native_asset::deposit(
            dynamic_field::borrow_mut(&mut self.id, Key<C>{}),
            some_coin
        )
    }

    public(friend) fun withdraw<C>(
        self: &mut RegisteredTokens,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<C> {
        native_asset::withdraw(
            dynamic_field::borrow_mut(&mut self.id, Key<C>{}),
            amount,
            ctx
        )
    }

    public fun balance<C>(self: &RegisteredTokens): u64 {
        native_asset::balance<C>(dynamic_field::borrow(&self.id, Key<C>{}))
    }

    public fun decimals<C>(self: &RegisteredTokens): u8 {
        if (is_wrapped<C>(self)) {
            wrapped_asset::decimals(borrow_wrapped<C>(self))
        } else {
            native_asset::decimals(borrow_native<C>(self))
        }
    }

    public fun to_token_info<C>(self: &RegisteredTokens): TokenInfo<C> {
        if (is_wrapped<C>(self)) {
            wrapped_asset::to_token_info(borrow_wrapped<C>(self))
        } else {
            native_asset::to_token_info(borrow_native<C>(self))
        }
    }

    public fun token_chain<C>(self: &RegisteredTokens): u16 {
        if (is_wrapped<C>(self)) {
            wrapped_asset::token_chain(borrow_wrapped<C>(self))
        } else {
            chain_id()
        }
    }

    public fun token_address<C>(self: &RegisteredTokens): ExternalAddress {
        if (is_wrapped<C>(self)) {
            wrapped_asset::token_address(borrow_wrapped<C>(self))
        } else {
            native_asset::token_address(borrow_native<C>(self))
        }
    }

    fun add_native<C>(
        self: &mut RegisteredTokens,
        asset: NativeAsset<C>
    ) {
        dynamic_field::add(&mut self.id, Key<C>{}, asset);
        self.num_native = self.num_native + 1;
    }

    public(friend) fun add_wrapped<C>(
        self: &mut RegisteredTokens,
        asset: WrappedAsset<C>
    ) {
        dynamic_field::add(&mut self.id, Key<C>{}, asset);
        self.num_wrapped = self.num_wrapped + 1;
    }

    fun borrow_wrapped<C>(self: &RegisteredTokens): &WrappedAsset<C> {
        dynamic_field::borrow(&self.id, Key<C>{})
    }

    fun borrow_native<C>(self: &RegisteredTokens): &NativeAsset<C> {
        dynamic_field::borrow(&self.id, Key<C>{})
    }
}

#[test_only]
module token_bridge::registered_tokens_test{
    use sui::test_scenario::{Self, Scenario, ctx, take_shared, return_shared,
    next_tx, take_from_address};
    use sui::coin::{Self, TreasuryCap};

    //use std::debug::print;

    use wormhole::external_address::{Self};
    //use wormhole::state::{chain_id};

    //use token_bridge::token_info::{Self};
    use token_bridge::registered_tokens::{new, num_wrapped, num_native,
        is_native, add_new_native, is_wrapped,
        deposit, balance, add_new_wrapped,
        destroy};
    use token_bridge::native_coin_witness::{Self, NATIVE_COIN_WITNESS};
    use token_bridge::native_coin_witness_v3::{Self, NATIVE_COIN_WITNESS_V3};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    fun test_registered_tokens(){
        let test = scenario();
        let (admin, _, _) = people();
        // 1) initialize RegisteredTokens object, native and wrapped coins
        next_tx(&mut test, admin);{
            //coin_witness::test_init(ctx(&mut test));
            native_coin_witness::test_init(ctx(&mut test));
            native_coin_witness_v3::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            let registered_tokens = new(ctx(&mut test));
            // 2) check initial state
            assert!(num_wrapped(&registered_tokens)==0, 0);
            assert!(num_native(&registered_tokens)==0, 0);

            // 3) register wrapped and native tokens
            //    (for testing purposes, we tread native_coin_witness_v2 as a
            //    wrapped coin type)
            let tcap = take_from_address<TreasuryCap<NATIVE_COIN_WITNESS_V3>>(&mut test, admin);
            add_new_wrapped<NATIVE_COIN_WITNESS_V3>(
                &mut registered_tokens,
                2, // chain
                external_address::from_bytes(x"001234"), // external address
                tcap, // treasury cap
                6 // decimals
            );

            add_new_native<NATIVE_COIN_WITNESS>(
                &mut registered_tokens,
                10,
                ctx(&mut test)
            );

            // mint some native coins, then deposit them into the token registry
            let native_tcap = take_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(&mut test);
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut native_tcap, 999, ctx(&mut test));
            assert!(coin::value(&coins)==999, 0);
            deposit<NATIVE_COIN_WITNESS>(&mut registered_tokens, coins);
            //print(&x"06");

            // 4) more assertions
            assert!(balance<NATIVE_COIN_WITNESS>(&registered_tokens)==999, 0);
            assert!(is_native<NATIVE_COIN_WITNESS>(&registered_tokens), 0);
            assert!(is_wrapped<NATIVE_COIN_WITNESS_V3>(&registered_tokens), 0);

            //?. cleanup
            return_shared(native_tcap);
            destroy(registered_tokens);
        };
         next_tx(&mut test, admin);{
            test_scenario::end(test);
        };
    }

    // #[test]
    // #[expected_failure(
    //     abort_code = token_bridge::registered_tokens::E_CANNOT_DEPOSIT_WRAPPED_COIN,
    //     location=token_bridge::registered_tokens
    // )]
    // fun test_registered_tokens_deposit_wrapped_fail(){
    //     let test = scenario();
    //     let (admin, _, _) = people();

    //     // 1) initialize RegisteredTokens object, native and wrapped coins
    //     let registered_tokens = new(ctx(&mut test));
    //     //coin_witness::test_init(ctx(&mut test));
    //     native_coin_witness::test_init(ctx(&mut test));
    //     native_coin_witness_v2::test_init(ctx(&mut test));
    //     next_tx(&mut test, admin);{
    //         // 2) check initial state
    //         assert!(num_wrapped(&registered_tokens)==0, 0);
    //         assert!(num_native(&registered_tokens)==0, 0);

    //         // 3) register wrapped and native tokens
    //         //    (for testing purposes, we tread native_coin_witness_v2 as a
    //         //    wrapped coin type)
    //         // let tcap = take_shared<TreasuryCap<NATIVE_COIN_WITNESS_V2>>(&mut test);
    //         // add_new_wrapped<NATIVE_COIN_WITNESS_V2>(
    //         //     &mut registered_tokens,
    //         //     2, // chain
    //         //     external_address::from_bytes(x"33"), // external address
    //         //     tcap, // treasury cap
    //         //     4 // decimals
    //         // );

    //         // mint some wrapped coins, then attempt to deposit them, resulting
    //         // in failure
    //         let coins = mint<NATIVE_COIN_WITNESS_V2>(
    //             &mut registered_tokens,
    //             1000,
    //             ctx(&mut test)
    //         );
    //         deposit<NATIVE_COIN_WITNESS_V2>(&mut registered_tokens, coins);

    //         // clean up
    //         destroy(registered_tokens);
    //         test_scenario::end(test);
    //     }
    // }
}
