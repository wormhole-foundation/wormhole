module token_bridge::transfer_tokens {
    use sui::sui::SUI;
    use sui::coin::{Self, Coin, CoinMetadata};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::myu16::{Self as u16, U16};
    use wormhole::emitter::{Self, EmitterCapability};

    use token_bridge::bridge_state::{Self, BridgeState};
    use token_bridge::transfer_result::{Self, TransferResult};
    use token_bridge::transfer::{Self};
    use token_bridge::normalized_amount::{Self};
    use token_bridge::transfer_with_payload::{Self};

    const E_TOO_MUCH_RELAYER_FEE: u64 = 0;

    public entry fun transfer_tokens<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        coin_metadata: &CoinMetadata<CoinType>,
        wormhole_fee_coins: Coin<SUI>,
        recipient_chain: u64,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u64,
    ) {
        let result = transfer_tokens_internal<CoinType>(
            bridge_state,
            coins,
            coin_metadata,
            relayer_fee,
        );
        let (token_chain, token_address, normalized_amount, normalized_relayer_fee)
            = transfer_result::destroy(result);
        let transfer = transfer::create(
            normalized_amount,
            token_address,
            token_chain,
            external_address::from_bytes(recipient),
            u16::from_u64(recipient_chain),
            normalized_relayer_fee,
        );
        bridge_state::publish_message(
            wormhole_state,
            bridge_state,
            nonce,
            transfer::encode(transfer),
            wormhole_fee_coins,
        );
    }

    public fun transfer_tokens_with_payload<CoinType>(
        emitter_cap: &EmitterCapability,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        coin_metadata: &CoinMetadata<CoinType>,
        wormhole_fee_coins: Coin<SUI>,
        recipient_chain: U16,
        recipient: ExternalAddress,
        relayer_fee: u64,
        nonce: u64,
        payload: vector<u8>,
    ): u64 {
        let result = transfer_tokens_internal<CoinType>(
            bridge_state,
            coins,
            coin_metadata,
            relayer_fee,
        );
        let (token_chain, token_address, normalized_amount, _)
            = transfer_result::destroy(result);

        let transfer = transfer_with_payload::create(
            normalized_amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            emitter::get_external_address(emitter_cap),
            payload
        );
        let payload = transfer_with_payload::encode(transfer);
        bridge_state::publish_message(
            wormhole_state,
            bridge_state,
            nonce,
            payload,
            wormhole_fee_coins
        )
    }

    fun transfer_tokens_internal<CoinType>(
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        coin_metadata: &CoinMetadata<CoinType>,
        relayer_fee: u64,
    ): TransferResult {
        let amount = coin::value<CoinType>(&coins);
        assert!(relayer_fee <= amount, E_TOO_MUCH_RELAYER_FEE);

        if (bridge_state::is_wrapped_asset<CoinType>(bridge_state)) {
            // now we burn the wrapped coins to remove them from circulation
            bridge_state::burn<CoinType>(bridge_state, coins);
        } else {
            // deposit native assets. this call to deposit requires the native
            // asset to have been attested
            bridge_state::deposit<CoinType>(bridge_state, coins);
        };

        let origin_info = bridge_state::origin_info<CoinType>(bridge_state);
        let token_chain = bridge_state::get_token_chain_from_origin_info(&origin_info);
        let token_address = bridge_state::get_token_address_from_origin_info(&origin_info);

        let decimals = coin::get_decimals(coin_metadata);
        let normalized_amount = normalized_amount::normalize(amount, decimals);
        let normalized_relayer_fee = normalized_amount::normalize(relayer_fee, decimals);

        let transfer_result: TransferResult = transfer_result::create(
            token_chain,
            token_address,
            normalized_amount,
            normalized_relayer_fee,
        );
        transfer_result
    }

    #[test_only]
    public fun transfer_tokens_test<CoinType>(
        bridge_state: &mut BridgeState,
        coins: Coin<CoinType>,
        coin_metadata: &CoinMetadata<CoinType>,
        relayer_fee: u64,
    ): TransferResult {
        transfer_tokens_internal(
            bridge_state,
            coins,
            coin_metadata,
            relayer_fee
        )
    }
}

#[test_only]
module token_bridge::transfer_token_test {
    use sui::sui::{SUI};
    use sui::test_scenario::{Self, Scenario, next_tx, return_shared, take_shared, take_from_address, ctx};
    use sui::coin::{Self, CoinMetadata, TreasuryCap};

    use token_bridge::transfer_result::Self;
    use token_bridge::transfer_tokens::{E_TOO_MUCH_RELAYER_FEE, transfer_tokens, transfer_tokens_test};
    use token_bridge::bridge_state::{Self, BridgeState};
    use token_bridge::coin_witness::{Self, COIN_WITNESS};
    use token_bridge::wrapped::{Self, NewWrappedCoin};
    use token_bridge::normalized_amount::{Self};
    use token_bridge::native_coin_witness::{Self, NATIVE_COIN_WITNESS};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};

    use wormhole::state::{State};
    use wormhole::myu16::{Self as u16};
    use wormhole::external_address::{Self};

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
            let bridge_state = take_shared<BridgeState>(&test);
            let worm_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            let treasury_cap = take_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            bridge_state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut treasury_cap, 10000, ctx(&mut test));

            transfer_tokens<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                coins,
                &coin_meta,
                coin::zero<SUI>(ctx(&mut test)), // zero fee paid to wormhole
                3, // recipient chain id
                x"deadbeef0000beef", // recipient address
                100000000, // relayer fee (too much)
                0 // nonce is unused field for now
            );
            return_shared<BridgeState>(bridge_state);
            return_shared<State>(worm_state);
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
            let bridge_state = take_shared<BridgeState>(&test);
            let worm_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            let treasury_cap = take_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            bridge_state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut treasury_cap, 10000, ctx(&mut test));

            transfer_tokens<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                coins,
                &coin_meta,
                coin::zero<SUI>(ctx(&mut test)), // zero fee paid to wormhole
                3, // recipient chain id
                x"deadbeef0000beef", // recipient address
                0, // relayer fee
                0 // unused field for now
            );
            return_shared<BridgeState>(bridge_state);
            return_shared<State>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(treasury_cap);
        };
        // check that custody of the coins is indeed transferred to token bridge
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<BridgeState>(&test);
            let cur_bal = bridge_state::balance<NATIVE_COIN_WITNESS>(&mut bridge_state);
            assert!(cur_bal==10000, 0);
            return_shared<BridgeState>(bridge_state);
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
            let bridge_state = take_shared<BridgeState>(&test);
            let worm_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            let treasury_cap = take_shared<TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            bridge_state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut treasury_cap, 10000, ctx(&mut test));

            let transfer_result = transfer_tokens_test<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                coins,
                &coin_meta,
                0 // relayer fee is zero
            );
            let (token_chain, token_address, normalized_amount, normalized_relayer_fee) = transfer_result::destroy(transfer_result);
            assert!(token_chain==u16::from_u64(21), 0);
            assert!(token_address==external_address::from_bytes(x"01"), 0); // wormhole addresses of coins are selected from a monotonic sequence starting from 1
            assert!(normalized_amount::get_amount(normalized_amount)==100, 0); // 10 - 8 = 2 decimals are removed from 10000, resulting in 100
            assert!(normalized_amount::get_amount(normalized_relayer_fee)==0, 0);

            return_shared<BridgeState>(bridge_state);
            return_shared<State>(worm_state);
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
            let bridge_state = take_shared<BridgeState>(&test);
            bridge_state::set_registered_emitter(&mut bridge_state, u16::from_u64(2), external_address::from_bytes(x"00000000000000000000000000000000000000000000000000000000deadbeef"));
            return_shared<BridgeState>(bridge_state);
        };
        // register wrapped asset type with the token bridge, mint some coins, initiate transfer
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<BridgeState>(&test);
            let worm_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<COIN_WITNESS>>(&test);
            let new_wrapped_coin = take_from_address<NewWrappedCoin<COIN_WITNESS>>(&test, admin);

            // register wrapped asset with the token bridge
            wrapped::register_wrapped_coin<COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                new_wrapped_coin,
                ctx(&mut test)
            );

            let verified_coin_witness = bridge_state::verify_coin_type<COIN_WITNESS>(
                &mut bridge_state,
                u16::from_u64(2),
                external_address::from_bytes(x"00000000000000000000000000000000000000000000000000000000beefface")
            );

            let coins = bridge_state::mint<COIN_WITNESS>(verified_coin_witness, &mut bridge_state, 1000, ctx(&mut test));

            transfer_tokens<COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                coins,
                &coin_meta,
                coin::zero<SUI>(ctx(&mut test)), // zero fee paid to wormhole
                3, // recipient chain id
                x"deadbeef0000beef", // recipient address
                0, // relayer fee
                0 // unused field for now
            );
            return_shared<BridgeState>(bridge_state);
            return_shared<State>(worm_state);
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
            let bridge_state = take_shared<BridgeState>(&test);
            bridge_state::set_registered_emitter(&mut bridge_state, u16::from_u64(2), external_address::from_bytes(x"00000000000000000000000000000000000000000000000000000000deadbeef"));
            return_shared<BridgeState>(bridge_state);
        };
        // register wrapped asset type with the token bridge, mint some coins, initiate transfer
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<BridgeState>(&test);
            let worm_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<COIN_WITNESS>>(&test);
            let new_wrapped_coin = take_from_address<NewWrappedCoin<COIN_WITNESS>>(&test, admin);

            // register wrapped asset with the token bridge
            wrapped::register_wrapped_coin<COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                new_wrapped_coin,
                ctx(&mut test)
            );

            let verified_coin_witness = bridge_state::verify_coin_type<COIN_WITNESS>(
                &mut bridge_state,
                u16::from_u64(2),
                external_address::from_bytes(x"00000000000000000000000000000000000000000000000000000000beefface")
            );

            let coins = bridge_state::mint<COIN_WITNESS>(verified_coin_witness, &mut bridge_state, 10000000000, ctx(&mut test));

            let transfer_result = transfer_tokens_test<COIN_WITNESS>(
                &mut bridge_state,
                coins,
                &coin_meta,
                0 // relayer fee is zero
            );

            let (token_chain, token_address, normalized_amount, normalized_relayer_fee) = transfer_result::destroy(transfer_result);
            assert!(token_chain==u16::from_u64(2), 0); // token chain id
            assert!(token_address==external_address::from_bytes(x"00000000000000000000000000000000000000000000000000000000beefface"), 0); // wrapped token native address
            assert!(normalized_amount::get_amount(normalized_amount)==10000000000, 0); // wrapped coin is created with maximum of 8 decimals (see wrapped.move)
            assert!(normalized_amount::get_amount(normalized_relayer_fee)==0, 0);

            return_shared<BridgeState>(bridge_state);
            return_shared<State>(worm_state);
            return_shared<CoinMetadata<COIN_WITNESS>>(coin_meta);
        };
        test_scenario::end(test);
    }
}
