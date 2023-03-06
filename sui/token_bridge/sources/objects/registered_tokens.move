module token_bridge::registered_tokens {
    use sui::coin::{Coin, TreasuryCap};
    use sui::dynamic_field::{Self};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::id_registry::{Self, IdRegistry};
    use wormhole::state::{chain_id};

    use token_bridge::native_asset::{Self, NativeAsset};
    use token_bridge::token_info::{TokenInfo};
    use token_bridge::wrapped_asset::{Self, WrappedAsset};

    friend token_bridge::state;
    #[test_only]
    friend token_bridge::registered_tokens_test;

    const E_UNREGISTERED: u64 = 0;
    const E_ALREADY_REGISTERED: u64 = 1;
    const E_CANNOT_DEPOSIT_WRAPPED_COIN: u64 = 2;
    const E_CANNOT_GET_TREASURY_CAP_FOR_NON_WRAPPED_COIN: u64 = 3;
    const E_CANNOT_REGISTER_NATIVE_COIN: u64 = 4;

    struct RegisteredTokens has key, store {
        id: UID,
        native_id_registry: IdRegistry,
        num_wrapped: u64,
        num_native: u64
    }

    struct Key<phantom C> has copy, drop, store {}

    public fun new(ctx: &mut TxContext): RegisteredTokens {
        RegisteredTokens {
            id: object::new(ctx),
            native_id_registry: id_registry::new(),
            num_wrapped: 0,
            num_native: 0
        }
    }

    #[test_only]
    public fun destroy(r: RegisteredTokens) {
        let RegisteredTokens {
            id: id,
            native_id_registry,
            num_wrapped: _,
            num_native: _
        } = r;
        object::delete(id);
        id_registry::destroy(native_id_registry);
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

    public(friend) fun treasury_cap<C>(
        self: &RegisteredTokens
    ): &TreasuryCap<C> {
        assert!(is_wrapped<C>(self),
            E_CANNOT_GET_TREASURY_CAP_FOR_NON_WRAPPED_COIN);
        wrapped_asset::treasury_cap<C>(
            dynamic_field::borrow(&self.id, Key<C>{})
        )
    }

    public(friend) fun add_new_wrapped<C>(
        self: &mut RegisteredTokens,
        chain: u16,
        addr: ExternalAddress,
        treasury_cap: TreasuryCap<C>,
        decimals: u8,
    ) {
        // Note: we do not assert that the coin type has not already been
        // registered using !has<C>(self), because add_new_wrapped
        // consumes TreasuryCap<C> and stores it within a WrappedAsset
        // within the token bridge forever. Since the treasury cap
        // is globally unique and can only be created once, there is no
        // risk that add_new_wrapped can be called again on the same
        // coin type.
        assert!(chain != chain_id(), E_CANNOT_REGISTER_NATIVE_COIN);
        add_wrapped<C>(
            self,
            wrapped_asset::new(chain, addr, treasury_cap, decimals)
        )
    }

    public(friend) fun add_new_native<C>(
        self: &mut RegisteredTokens,
        decimals: u8,
    ) {
        assert!(!has<C>(self), E_ALREADY_REGISTERED);
        let addr = id_registry::next_address(&mut self.native_id_registry);
        add_native<C>(
            self,
            native_asset::new(addr, decimals)
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

    fun add_wrapped<C>(
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

// In this test, we exercise the various functionalities of RegisteredTokens,
// including registering native and wrapped coins via add_new_native, and
// add_new_wrapped, minting/burning/depositing/withdrawing said tokens, and also
// storing metadata about the tokens.
#[test_only]
module token_bridge::registered_tokens_test {
    use sui::test_scenario::{Self, Scenario, ctx, take_shared, return_shared,
    next_tx, take_from_address};
    use sui::coin::{Self, TreasuryCap};

    use wormhole::external_address::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::registered_tokens::{Self, new, num_wrapped, num_native,
        is_native, add_new_native, is_wrapped, token_chain, withdraw,
        deposit, balance, add_new_wrapped, decimals, token_address, destroy};
    use token_bridge::native_coin_10_decimals::{Self, NATIVE_COIN_10_DECIMALS};
    use token_bridge::wrapped_coin_7_decimals::{Self, WRAPPED_COIN_7_DECIMALS};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    fun test_registered_tokens(){
        let test = scenario();
        let (admin, _, _) = people();

        // 1) initialize RegisteredTokens object, native and wrapped coins
        next_tx(&mut test, admin);{
            //coin_witness::test_init(ctx(&mut test));
            native_coin_10_decimals::test_init(ctx(&mut test));
            wrapped_coin_7_decimals::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            let registered_tokens = new(ctx(&mut test));

            // 2) check initial state
            assert!(num_wrapped(&registered_tokens)==0, 0);
            assert!(num_native(&registered_tokens)==0, 0);

            // 3) register wrapped and native tokens, then mint/burn/deposit
            let tcap = take_from_address<TreasuryCap<WRAPPED_COIN_7_DECIMALS>>(
                &mut test,
                admin
            );
            add_new_wrapped<WRAPPED_COIN_7_DECIMALS>(
                &mut registered_tokens,
                2, // chain
                external_address::from_any_bytes(x"001234"), // external address
                tcap, // treasury cap
                7 // decimals
            );

            add_new_native<NATIVE_COIN_10_DECIMALS>(
                &mut registered_tokens,
                10,
            );

            // mint some native coins, then deposit them into the token registry
            let native_tcap = take_shared<TreasuryCap<NATIVE_COIN_10_DECIMALS>>(
                &mut test
            );
            let coins = coin::mint<NATIVE_COIN_10_DECIMALS>(
                &mut native_tcap,
                999,
                ctx(&mut test)
            );
            assert!(coin::value(&coins)==999, 0);
            deposit<NATIVE_COIN_10_DECIMALS>(&mut registered_tokens, coins);

            // withdraw, check value, and re-deposit native coins into registry
            coins = withdraw<NATIVE_COIN_10_DECIMALS>(
                &mut registered_tokens,
                499,
                ctx(&mut test)
            );
            assert!(coin::value(&coins)==499, 0);
            deposit<NATIVE_COIN_10_DECIMALS>(&mut registered_tokens, coins);

            // mint some wrapped coins, then burn them
            let wcoins = registered_tokens::mint<WRAPPED_COIN_7_DECIMALS>(
                &mut registered_tokens,
                420420420,
                ctx(&mut test)
            );
            assert!(coin::value(&wcoins)==420420420, 0);
            registered_tokens::burn<WRAPPED_COIN_7_DECIMALS>(
                &mut registered_tokens,
                wcoins
            );

            // 4) more checks and assertions on registered_tokens

            // check amount in native coin custody is equal to amount deposited
            assert!(balance<NATIVE_COIN_10_DECIMALS>(&registered_tokens)==999, 0);

            // check that native/wrapped classification is correct
            assert!(is_native<NATIVE_COIN_10_DECIMALS>(&registered_tokens), 0);
            assert!(is_wrapped<WRAPPED_COIN_7_DECIMALS>(&registered_tokens), 0);

            // check decimals are correct
            assert!(decimals<NATIVE_COIN_10_DECIMALS>(&registered_tokens)==10, 0);
            assert!(decimals<WRAPPED_COIN_7_DECIMALS>(&registered_tokens)==7, 0);

            // check token addresses are correct
            assert!(token_address<NATIVE_COIN_10_DECIMALS>(&registered_tokens)==
                external_address::from_any_bytes(x"01"), 0);
            assert!(token_address<WRAPPED_COIN_7_DECIMALS>(&registered_tokens)==
                external_address::from_any_bytes(x"001234"), 0);

            // check token chains are correct
            assert!(token_chain<NATIVE_COIN_10_DECIMALS>(&registered_tokens)==
                chain_id(), 0);
            assert!(token_chain<WRAPPED_COIN_7_DECIMALS>(&registered_tokens)==
                2, 0);

            // 5) cleanup

            return_shared(native_tcap);
            destroy(registered_tokens);
        };
         next_tx(&mut test, admin);{
            test_scenario::end(test);
        };
    }

    // In this negative test case, we try to register a native token twice.
    #[test]
    #[expected_failure(
        abort_code = token_bridge::registered_tokens::E_ALREADY_REGISTERED,
        location=token_bridge::registered_tokens
    )]
    fun test_registered_tokens_already_registered(){
        let test = scenario();
        let (admin, _, _) = people();

        // 1) Initialize RegisteredTokens object, native and wrapped coins.
        next_tx(&mut test, admin);{
            //coin_witness::test_init(ctx(&mut test));
            native_coin_10_decimals::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            let registered_tokens = new(ctx(&mut test));

            // 2) Check initial state.
            assert!(num_wrapped(&registered_tokens)==0, 0);
            assert!(num_native(&registered_tokens)==0, 0);

            // 3)  Attempt to register native coin twice.
            add_new_native<NATIVE_COIN_10_DECIMALS>(
                &mut registered_tokens,
                10, // Decimals.
            );
            add_new_native<NATIVE_COIN_10_DECIMALS>(
                &mut registered_tokens,
                10, // Decimals.
            );

            //4) Cleanup.
            destroy(registered_tokens);
        };
         next_tx(&mut test, admin);{
            test_scenario::end(test);
        };
    }

    // In this negative test case, we attempt to register a native coin as a
    // wrapped coin.
    #[test]
    #[expected_failure(
        abort_code = token_bridge::registered_tokens::E_CANNOT_REGISTER_NATIVE_COIN,
        location=token_bridge::registered_tokens
    )]
    fun test_registered_tokens_cannot_register_native(){
        let test = scenario();
        let (admin, _, _) = people();

        // 1) Initialize RegisteredTokens object, native and wrapped coins.
        next_tx(&mut test, admin);{
            native_coin_10_decimals::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            let registered_tokens = new(ctx(&mut test));

            // 2) Check initial state.
            assert!(num_wrapped(&registered_tokens)==0, 0);
            assert!(num_native(&registered_tokens)==0, 0);

            // 3) Attempt to register a native coin as wrapped.
            let tcap = take_shared<TreasuryCap<NATIVE_COIN_10_DECIMALS>>(
                &mut test
            );
            add_new_wrapped<NATIVE_COIN_10_DECIMALS>(
                &mut registered_tokens,
                21, // Chain.
                external_address::from_any_bytes(x"001234"), // External address.
                tcap, // Treasury cap.
                7 // Decimals.
            );

            //4) Cleanup.
            destroy(registered_tokens);
        };
         next_tx(&mut test, admin);{
            test_scenario::end(test);
        };
    }

    // In this negative test case, we attempt to deposit a wrapped token into
    // a RegisteredTokens object, resulting in failure. A wrapped coin can
    // only be minted and burned, not deposited.
    #[test]
    #[expected_failure(
        abort_code = token_bridge::registered_tokens::E_CANNOT_DEPOSIT_WRAPPED_COIN,
        location=token_bridge::registered_tokens
    )]
    fun test_registered_tokens_deposit_wrapped_fail(){
        let test = scenario();
        let (admin, _, _) = people();

        // 1) initialize RegisteredTokens object, native and wrapped coins
        next_tx(&mut test, admin);{
            //coin_witness::test_init(ctx(&mut test));
            wrapped_coin_7_decimals::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            let registered_tokens = new(ctx(&mut test));

            // 2) check initial state
            assert!(num_wrapped(&registered_tokens)==0, 0);
            assert!(num_native(&registered_tokens)==0, 0);

            // 3) register wrapped tokens, then mint/burn/deposit
            let tcap = take_from_address<TreasuryCap<WRAPPED_COIN_7_DECIMALS>>(
                &mut test,
                admin
            );
            add_new_wrapped<WRAPPED_COIN_7_DECIMALS>(
                &mut registered_tokens,
                2, // chain
                external_address::from_any_bytes(x"001234"), // external address
                tcap, // treasury cap
                7 // decimals
            );

            // mint some wrapped coins, then attempt to deposit them
            let wcoins = registered_tokens::mint<WRAPPED_COIN_7_DECIMALS>(
                &mut registered_tokens,
                420420420,
                ctx(&mut test)
            );
            assert!(coin::value(&wcoins)==420420420, 0);
            // the line below will fail
            registered_tokens::deposit<WRAPPED_COIN_7_DECIMALS>(
                &mut registered_tokens,
                wcoins
            );

            //4) cleanup
            destroy(registered_tokens);
        };
         next_tx(&mut test, admin);{
            test_scenario::end(test);
        };
    }
}
