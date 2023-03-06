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

    const E_UNREGISTERED: u64 = 0;
    const E_ALREADY_REGISTERED: u64 = 1;

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
    ) {
        assert!(!has<C>(self), E_ALREADY_REGISTERED);
        let addr = native_id_registry::next_id(&mut self.native_id_registry);
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
