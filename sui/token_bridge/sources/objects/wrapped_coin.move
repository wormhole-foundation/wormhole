module token_bridge::wrapped_coin {
    use sui::coin::{TreasuryCap};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};

    // Exclusive access to this object.
    friend token_bridge::create_wrapped;

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
