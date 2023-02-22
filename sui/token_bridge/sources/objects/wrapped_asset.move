module token_bridge::wrapped_asset {
    use sui::coin::{Self, Coin, TreasuryCap};
    use sui::tx_context::{TxContext};
    use wormhole::external_address::{ExternalAddress};

    use token_bridge::token_info::{Self, TokenInfo};

    // For `burn` and `mint`
    friend token_bridge::registered_tokens;

    /// WrappedAsset<C> stores all the metadata about a wrapped asset
    struct WrappedAsset<phantom C> has store {
        token_chain: u16,
        token_address: ExternalAddress,
        treasury_cap: TreasuryCap<C>,
        decimals: u8,
    }

    public fun new<C>(
        token_chain: u16,
        token_address: ExternalAddress,
        treasury_cap: TreasuryCap<C>,
        decimals: u8,
    ): WrappedAsset<C> {
        return WrappedAsset {
            token_chain,
            token_address,
            treasury_cap,
            decimals
        }
    }

    #[test_only]
    public fun destroy<C>(wrapped_asset: WrappedAsset<C>): TreasuryCap<C>{
        let WrappedAsset {
            token_chain: _,
            token_address: _,
            treasury_cap: tcap,
            decimals: _
        } = wrapped_asset;
        tcap
    }

    public fun token_chain<C>(self: &WrappedAsset<C>): u16 {
        self.token_chain
    }

    public fun token_address<C>(self: &WrappedAsset<C>): ExternalAddress {
        self.token_address
    }

    public fun treasury_cap<C>(self: &WrappedAsset<C>): &TreasuryCap<C> {
        &self.treasury_cap
    }

    public fun decimals<C>(self: &WrappedAsset<C>): u8 {
        self.decimals
    }

    public fun to_token_info<C>(self: &WrappedAsset<C>): TokenInfo<C> {
        token_info::new(
            true, // is_wrapped
            self.token_chain,
            self.token_address
        )
    }

    public(friend) fun burn<C>(
        self: &mut WrappedAsset<C>,
        burnable: Coin<C>
    ): u64 {
        coin::burn(&mut self.treasury_cap, burnable)
    }

    public(friend) fun mint<C>(
        self: &mut WrappedAsset<C>,
        amount: u64,
        ctx: &mut TxContext
    ): Coin<C> {
        coin::mint(&mut self.treasury_cap, amount, ctx)
    }
}

#[test_only]
module token_bridge::wrapped_asset_test {
    use sui::transfer::{Self};
    use sui::coin::{TreasuryCap};
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_from_address};

    use wormhole::external_address::{Self};

    use token_bridge::native_coin_witness_v3::{Self, NATIVE_COIN_WITNESS_V3};
    use token_bridge::wrapped_asset::{Self, token_chain, token_address,
        decimals};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    public fun test_wrapped_asset(){
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
            let addr =  external_address::from_bytes(x"112233");
            let wrapped_asset = wrapped_asset::new(
                2, // token chain
                addr, //token address
                tcap, // treasury cap
                6, // decimals
            );
            assert!(token_chain(&wrapped_asset) == 2, 0);
            assert!(decimals(&wrapped_asset) == 6, 0);
            assert!(token_address(&wrapped_asset)==addr, 0);
            let tcap = wrapped_asset::destroy<NATIVE_COIN_WITNESS_V3>(
                wrapped_asset
            );
            transfer::transfer(tcap, admin);
        };
        test_scenario::end(test);
    }
}
