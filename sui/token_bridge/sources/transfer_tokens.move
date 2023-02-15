module token_bridge::transfer_tokens {
    use sui::sui::SUI;
    use sui::coin::{Self, Coin};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self};

    use token_bridge::normalized_amount::{Self};
    use token_bridge::state::{Self, State};
    use token_bridge::token_info::{Self};
    use token_bridge::transfer_result::{Self, TransferResult};
    use token_bridge::transfer::{Self};

    // `transfer_tokens_with_payload` requires `handle_transfer_tokens`
    friend token_bridge::transfer_tokens_with_payload;

    const E_TOO_MUCH_RELAYER_FEE: u64 = 0;

    public entry fun transfer_tokens<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<SUI>,
        recipient_chain: u16,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u32,
    ) {
        let result = handle_transfer_tokens<CoinType>(
            token_bridge_state,
            coins,
            relayer_fee,
        );
        let (
            token_chain,
            token_address,
            normalized_amount,
            normalized_relayer_fee
        ) = transfer_result::destroy(result);
        let transfer = transfer::new(
            normalized_amount,
            token_address,
            token_chain,
            external_address::from_bytes(recipient),
            recipient_chain,
            normalized_relayer_fee,
        );

        state::publish_wormhole_message(
            token_bridge_state,
            worm_state,
            nonce,
            transfer::serialize(transfer),
            wormhole_fee_coins,
        );
    }

    public(friend) fun handle_transfer_tokens<CoinType>(
        token_bridge_state: &mut State,
        coins: Coin<CoinType>,
        relayer_fee: u64,
    ): TransferResult {
        let amount = coin::value<CoinType>(&coins);

        // It doesn't make sense to specify a `relayer_fee` larger than the
        // total amount bridged over.
        assert!(relayer_fee <= amount, E_TOO_MUCH_RELAYER_FEE);

        // Get info about the token
        let info = state::token_info<CoinType>(token_bridge_state);

        if (token_info::is_wrapped(&info)) {
            // now we burn the wrapped coins to remove them from circulation
            state::burn<CoinType>(token_bridge_state, coins);
        } else {
            // deposit native assets. this call to deposit requires the native
            // asset to have been attested
            state::deposit<CoinType>(token_bridge_state, coins);
        };

        let decimals = state::coin_decimals<CoinType>(token_bridge_state);

        transfer_result::new(
            token_info::chain(&info),
            token_info::addr(&info),
            normalized_amount::from_raw(amount, decimals),
            normalized_amount::from_raw(relayer_fee, decimals),
        )
    }

    #[test_only]
    public fun transfer_tokens_test<CoinType>(
        bridge_state: &mut State,
        coins: Coin<CoinType>,
        relayer_fee: u64,
    ): TransferResult {
        handle_transfer_tokens(
            bridge_state,
            coins,
            relayer_fee
        )
    }
}

#[test_only]
module token_bridge::transfer_token_test {
    use sui::coin::{Self, CoinMetadata, TreasuryCap};
    use sui::sui::{SUI};
    use sui::test_scenario::{
        Self,
        Scenario,
        next_tx,
        return_shared,
        take_shared,
        take_from_address,
        ctx
    };
    use wormhole::external_address::{Self};
    use wormhole::state::{State as WormholeState};

    use token_bridge::bridge_state_test::{
        set_up_wormhole_core_and_token_bridges
    };
    use token_bridge::create_wrapped::{Self};
    use token_bridge::coin_witness::{Self, COIN_WITNESS};
    use token_bridge::native_coin_witness::{Self, NATIVE_COIN_WITNESS};
    use token_bridge::normalized_amount::{Self};
    use token_bridge::state::{Self, State};
    use token_bridge::transfer_result::{Self};
    use token_bridge::transfer_tokens::{
        E_TOO_MUCH_RELAYER_FEE,
        transfer_tokens,
        transfer_tokens_test
    };
    use token_bridge::wrapped_coin::{WrappedCoin};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    #[expected_failure(abort_code = E_TOO_MUCH_RELAYER_FEE)] // E_TOO_MUCH_RELAYER_FEE
    fun test_transfer_native_token_too_much_relayer_fee(){
        let (admin, _, _) = people();
        let test = scenario();
        // set up core and token bridges
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // initialize the coin
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge, mint some coins, initiate transfer
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            let treasury_cap = take_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut treasury_cap, 10000, ctx(&mut test));

            transfer_tokens<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                coins,
                coin::zero<SUI>(ctx(&mut test)), // zero fee paid to wormhole
                3, // recipient chain id
                x"deadbeef0000beef", // recipient address
                100000000, // relayer fee (too much)
                0 // nonce is unused field for now
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(treasury_cap);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_transfer_native_token(){
        let (admin, _, _) = people();
        let test = scenario();
        // set up core and token bridges
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // initialize the coin
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge, mint some coins, initiate transfer
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            let treasury_cap = take_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut treasury_cap, 10000, ctx(&mut test));

            transfer_tokens<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                coins,
                coin::zero<SUI>(ctx(&mut test)), // zero fee paid to wormhole
                3, // recipient chain id
                x"deadbeef0000beef", // recipient address
                0, // relayer fee
                0 // unused field for now
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(treasury_cap);
        };
        // check that custody of the coins is indeed transferred to token bridge
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let cur_bal = state::balance<NATIVE_COIN_WITNESS>(&mut bridge_state);
            assert!(cur_bal==10000, 0);
            return_shared<State>(bridge_state);
        };
        test_scenario::end(test);
    }

    // check transfer result for native token transfer is constructed properly
    #[test]
    fun test_transfer_native_token_internal(){
        let (admin, _, _) = people();
        let test = scenario();
        // set up core and token bridges
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // initialize the coin
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge, mint some coins, initiate transfer
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            let treasury_cap = take_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut treasury_cap, 10000, ctx(&mut test));

            let transfer_result = transfer_tokens_test<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                coins,
                0 // relayer fee is zero
            );
            let (token_chain, token_address, normalized_amount, normalized_relayer_fee) = transfer_result::destroy(transfer_result);
            assert!(token_chain == 21, 0);
            assert!(token_address==external_address::from_bytes(x"01"), 0); // wormhole addresses of coins are selected from a monotonic sequence starting from 1
            assert!(normalized_amount::value(&normalized_amount)==100, 0); // 10 - 8 = 2 decimals are removed from 10000, resulting in 100
            assert!(normalized_amount::value(&normalized_relayer_fee)==0, 0);

            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(treasury_cap);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_transfer_wrapped_token(){
        let (admin, _, _) = people();
        let test = scenario();
        // set up core and token bridges
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // initialize the wrapped coin and register the eth chain
        next_tx(&mut test, admin);{
            coin_witness::test_init(ctx(&mut test));
        };
        // register chain emitter (chain id x emitter address) that attested the wrapped token
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            state::register_emitter(
                &mut bridge_state,
                2, // chain ID
                external_address::from_bytes(
                    x"00000000000000000000000000000000000000000000000000000000deadbeef"
                )
            );
            return_shared<State>(bridge_state);
        };
        // register wrapped asset type with the token bridge, mint some coins, initiate transfer
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<COIN_WITNESS>>(&test);
            let new_wrapped_coin =
                take_from_address<WrappedCoin<COIN_WITNESS>>(&test, admin);

            // register wrapped asset with the token bridge
            create_wrapped::register_wrapped_coin<COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                new_wrapped_coin,
                ctx(&mut test)
            );

            let coins =
                state::mint<COIN_WITNESS>(
                    &mut bridge_state,
                    1000, // amount
                    ctx(&mut test)
                );

            transfer_tokens<COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                coins,
                coin::zero<SUI>(ctx(&mut test)), // zero fee paid to wormhole
                3, // recipient chain id
                x"deadbeef0000beef", // recipient address
                0, // relayer fee
                0 // unused field for now
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<COIN_WITNESS>>(coin_meta);
        };
        // How to check if token was actually burned?
        test_scenario::end(test);
    }

     #[test]
    fun test_transfer_wrapped_token_internal(){
        let (admin, _, _) = people();
        let test = scenario();
        // set up core and token bridges
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // initialize the wrapped coin and register the eth chain
        next_tx(&mut test, admin);{
            coin_witness::test_init(ctx(&mut test));
        };
        // register chain emitter (chain id x emitter address) that attested the wrapped token
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            state::register_emitter(
                &mut bridge_state,
                2, // chain ID
                external_address::from_bytes(
                    x"00000000000000000000000000000000000000000000000000000000deadbeef"
                )
            );
            return_shared<State>(bridge_state);
        };
        // register wrapped asset type with the token bridge, mint some coins, initiate transfer
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<COIN_WITNESS>>(&test);
            let new_wrapped_coin = take_from_address<WrappedCoin<COIN_WITNESS>>(&test, admin);

            // register wrapped asset with the token bridge
            create_wrapped::register_wrapped_coin<COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                new_wrapped_coin,
                ctx(&mut test)
            );

            let coins =
                state::mint<COIN_WITNESS>(
                    &mut bridge_state,
                    10000000000, // amount
                    ctx(&mut test)
                );

            let transfer_result = transfer_tokens_test<COIN_WITNESS>(
                &mut bridge_state,
                coins,
                0 // relayer fee is zero
            );

            let (
                token_chain,
                token_address,
                normalized_amount,
                normalized_relayer_fee
            ) = transfer_result::destroy(transfer_result);
            assert!(token_chain == 2, 0); // token chain id
            assert!(
                token_address == external_address::from_bytes(
                    x"00000000000000000000000000000000000000000000000000000000beefface"
                ),
                0
            ); // wrapped token native address
            assert!(
                normalized_amount::value(&normalized_amount) == 10000000000,
                0
            ); // wrapped coin is created with maximum of 8 decimals (see wrapped.move)
            assert!(
                normalized_amount::value(&normalized_relayer_fee) == 0,
                0
            );

            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<COIN_WITNESS>>(coin_meta);
        };
        test_scenario::end(test);
    }
}
