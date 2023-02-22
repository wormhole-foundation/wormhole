module token_bridge::wrapped_coin {
    use sui::coin::{TreasuryCap};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};

    // Exclusive access to this object.
    friend token_bridge::create_wrapped;
    #[test_only]
    friend token_bridge::wrapped_coin_test;
    /// Wrapped assets are created in two steps.
    /// 1) The coin is initialised by calling `create_wrapped_coin` in the
    /// `init` function of a OTW module.
    /// 2) The coin is registered in the token bridge in
    /// `register_wrapped_coin`.
    ///
    /// Since Step 1. takes places in an untrusted context, we want to remove
    /// all degrees of freedom. To this end, `create_wrapped_coin` just takes a
    /// VAA, and returns a `WrappedCoin` object. That's the only way to
    /// create a `WrappedCoin` object. Then this object can be passed to
    /// `register_wrapped_coin` in Step 2.
    ///
    /// This setup ensures that we don't have to trust (or verify) that the OTW
    /// initialiser did the right thing.
    ///
    /// TODO: it would be nice if we could also enforce that the OTW struct's
    /// name matches the token symbol being registered. Currently there's no way
    /// to do this in the sui framework.
    struct WrappedCoin<phantom C> has key, store {
        id: UID,
        vaa_bytes: vector<u8>,
        treasury_cap: TreasuryCap<C>,
        decimals: u8
    }

    public(friend) fun new<C>(
        vaa_bytes: vector<u8>,
        treasury_cap: TreasuryCap<C>,
        decimals: u8,
        ctx: &mut TxContext
    ): WrappedCoin<C> {
        WrappedCoin {
            id: object::new(ctx),
            vaa_bytes,
            treasury_cap,
            decimals
        }
    }

    public(friend) fun destroy<C>(
        coin: WrappedCoin<C>
    ): (vector<u8>, TreasuryCap<C>, u8) {
        let WrappedCoin {
            id,
            vaa_bytes,
            treasury_cap,
            decimals
        } = coin;
        object::delete(id);

        (vaa_bytes, treasury_cap, decimals)
    }
}


#[test_only]
module token_bridge::wrapped_coin_test {
    use sui::transfer::{Self};
    use sui::coin::{Self, TreasuryCap};
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_from_address};

    use token_bridge::native_coin_witness_v3::{Self, NATIVE_COIN_WITNESS_V3};
    use token_bridge::wrapped_coin::{Self};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    public fun test_wrapped_coin_creation(){
        let test = scenario();
        let (admin, _, _) = people();
        next_tx(&mut test, admin); {
            native_coin_witness_v3::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            let tcap = take_from_address<TreasuryCap<NATIVE_COIN_WITNESS_V3>>(
                &mut test,
                admin
            );
            let wrapped_coin = wrapped_coin::new(
                x"112233", //vaa bytes
                tcap, // treasury cap
                6, // decimals
                ctx(&mut test)
            );
            let (vaa_bytes, tcap, decimals) = wrapped_coin::destroy(
                wrapped_coin
            );
            assert!(vaa_bytes == x"112233", 0);
            assert!(decimals == 6, 0);
            assert!(coin::total_supply<NATIVE_COIN_WITNESS_V3>(&tcap)==0, 0);
            transfer::transfer(tcap, admin);
        };
        test_scenario::end(test);
    }
}
