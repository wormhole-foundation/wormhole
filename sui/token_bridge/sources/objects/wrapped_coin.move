module token_bridge::wrapped_coin {
    use sui::coin::{TreasuryCap};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};

    // Exclusive access to this object.
    friend token_bridge::create_wrapped;

    /// Wrapped assets are created in two steps.
    /// 1) The coin is initialised by calling `create_unregistered_currency` in the
    /// `init` function of a OTW module.
    /// 2) The coin is registered in the token bridge in
    /// `register_new_coin`.
    ///
    /// Since Step 1. takes places in an untrusted context, we want to remove
    /// all degrees of freedom. To this end, `create_unregistered_currency` just takes a
    /// VAA, and returns a `WrappedCoin` object. That's the only way to
    /// create a `WrappedCoin` object. Then this object can be passed to
    /// `register_new_coin` in Step 2.
    ///
    /// This setup ensures that we don't have to trust (or verify) that the OTW
    /// initialiser did the right thing.
    ///
    /// TODO: it would be nice if we could also enforce that the OTW struct's
    /// name matches the token symbol being registered. Currently there's no way
    /// to do this in the sui framework.
    struct WrappedCoin<phantom C> has key, store {
        id: UID,
        vaa_buf: vector<u8>,
        treasury_cap: TreasuryCap<C>,
        decimals: u8
    }

    public(friend) fun new<C>(
        vaa_buf: vector<u8>,
        treasury_cap: TreasuryCap<C>,
        decimals: u8,
        ctx: &mut TxContext
    ): WrappedCoin<C> {
        WrappedCoin {
            id: object::new(ctx),
            vaa_buf,
            treasury_cap,
            decimals
        }
    }

    #[test_only]
    public fun new_test_only<C>(
        vaa_bytes: vector<u8>,
        treasury_cap: TreasuryCap<C>,
        decimals: u8,
        ctx: &mut TxContext
    ): WrappedCoin<C> {
        new(vaa_bytes, treasury_cap, decimals, ctx)
    }

    public(friend) fun destroy<C>(
        coin: WrappedCoin<C>
    ): (vector<u8>, TreasuryCap<C>, u8) {
        let WrappedCoin {
            id,
            vaa_buf,
            treasury_cap,
            decimals
        } = coin;
        object::delete(id);

        (vaa_buf, treasury_cap, decimals)
    }

    #[test_only]
    public fun destroy_test_only<C>(
        coin: WrappedCoin<C>
    ): (vector<u8>, TreasuryCap<C>, u8) {
        destroy(coin)
    }
}


#[test_only]
module token_bridge::wrapped_coin_test {
    use sui::transfer::{Self};
    use sui::coin::{Self, TreasuryCap};
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_from_address};

    use token_bridge::wrapped_coin_7_decimals::{Self, WRAPPED_COIN_7_DECIMALS};
    use token_bridge::wrapped_coin::{Self};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    public fun test_wrapped_coin_creation(){
        let test = scenario();
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            wrapped_coin_7_decimals::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            let tcap = take_from_address<TreasuryCap<WRAPPED_COIN_7_DECIMALS>>(
                &mut test,
                admin
            );
            let wrapped_coin = wrapped_coin::new_test_only(
                x"112233", //vaa bytes
                tcap, // treasury cap
                6, // decimals
                ctx(&mut test)
            );
            let (vaa_bytes, tcap, decimals) = wrapped_coin::destroy_test_only(
                wrapped_coin
            );
            assert!(vaa_bytes == x"112233", 0);
            assert!(decimals == 6, 0);
            assert!(coin::total_supply<WRAPPED_COIN_7_DECIMALS>(&tcap)==0, 0);
            transfer::public_transfer(tcap, admin);
        };
        test_scenario::end(test);
    }
}
