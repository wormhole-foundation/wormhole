module token_bridge::complete_transfer {
    use sui::tx_context::{TxContext};
    use sui::transfer::{Self as transfer_object};
    use sui::coin::{Self};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self};

    use token_bridge::state::{Self, State, VerifiedCoinType};
    use token_bridge::vaa::{Self};
    use token_bridge::transfer::{Self, Transfer};
    use token_bridge::normalized_amount::{Self};

    const E_INVALID_TARGET: u64 = 0;

    public entry fun submit_vaa_entry<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut State,
        vaa: vector<u8>,
        fee_recipient: address,
        ctx: &mut TxContext
    ) {
        submit_vaa<CoinType>(
            wormhole_state,
            bridge_state,
            vaa,
            fee_recipient,
            ctx
        );
    }

    public fun submit_vaa<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut State,
        vaa: vector<u8>,
        fee_recipient: address,
        ctx: &mut TxContext
    ): Transfer {

        let vaa = vaa::parse_verify_and_replay_protect(
            wormhole_state,
            bridge_state,
            vaa,
            ctx
        );

        let transfer = transfer::deserialize(wormhole::myvaa::destroy(vaa));

        let token_chain = transfer::token_chain(&transfer);
        let token_address = transfer::token_address(&transfer);
        let verified_coin_witness = state::verify_coin_type<CoinType>(
            bridge_state,
            token_chain,
            token_address
        );

        complete_transfer<CoinType>(
            verified_coin_witness,
            transfer,
            wormhole_state,
            bridge_state,
            fee_recipient,
            ctx
        )
    }

    // complete transfer with arbitrary Transfer request and without the VAA
    // for native tokens
    #[test_only]
    public fun test_complete_transfer<CoinType>(
        my_transfer: Transfer,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut State,
        fee_recipient: address,
        ctx: &mut TxContext
    ): Transfer {
        let token_chain = transfer::token_chain(&my_transfer);
        let token_address = transfer::token_address(&my_transfer);
        let verified_coin_witness = state::verify_coin_type<CoinType>(
            bridge_state,
            token_chain,
            token_address
        );
        complete_transfer<CoinType>(
            verified_coin_witness,
            my_transfer,
            wormhole_state,
            bridge_state,
            fee_recipient,
            ctx
        )
    }

    fun complete_transfer<CoinType>(
        verified_coin_witness: VerifiedCoinType<CoinType>,
        my_transfer: Transfer,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut State,
        fee_recipient: address,
        ctx: &mut TxContext
    ): Transfer { // TODO: why return transfer?
        let to_chain = transfer::recipient_chain(&my_transfer);
        let this_chain = wormhole::state::get_chain_id(wormhole_state);
        assert!(to_chain == this_chain, E_INVALID_TARGET);

        let recipient = external_address::to_address(
            &transfer::recipient(&my_transfer)
        );
        let normed_amount = transfer::amount(&my_transfer);
        let normed_fee = transfer::relayer_fee(&my_transfer);

        let recipient_coins;
        let amount;
        let fee_amount;
        if (state::is_wrapped_asset<CoinType>(bridge_state)) {
            let decimals =
                state::get_wrapped_decimals<CoinType>(bridge_state);
            amount =
                normalized_amount::to_raw(normed_amount, decimals);
            fee_amount =
                normalized_amount::to_raw(normed_fee, decimals);
            recipient_coins =
                state::mint<CoinType>(
                    verified_coin_witness,
                    bridge_state,
                    amount,
                    ctx
                );
        } else {
            let decimals =
                state::get_native_decimals<CoinType>(bridge_state);
            amount =
                normalized_amount::to_raw(normed_amount, decimals);
            fee_amount =
                normalized_amount::to_raw(normed_fee, decimals);
            recipient_coins =
                state::withdraw<CoinType>(
                    verified_coin_witness,
                    bridge_state,
                    amount,
                    ctx
                );
        };
        // take out fee from the recipient's coins. `extract` will revert
        // if fee > amount
        let fee_coins = coin::split(&mut recipient_coins, fee_amount, ctx);
        transfer_object::transfer(recipient_coins, recipient);
        transfer_object::transfer(fee_coins, fee_recipient);
        my_transfer
    }
}

#[test_only]
module token_bridge::complete_transfer_test {
    use std::bcs::{Self};

    use sui::test_scenario::{Self, Scenario, next_tx, return_shared, take_shared, ctx, take_from_address, return_to_address};
    use sui::coin::{Self, Coin, CoinMetadata};

    use wormhole::external_address::{Self};

    use token_bridge::normalized_amount::{Self};
    use token_bridge::transfer::{Self};
    use token_bridge::state::{Self, State};
    use token_bridge::coin_witness::{Self, COIN_WITNESS};
    use token_bridge::coin_witness_test::{test_register_wrapped_};
    use token_bridge::complete_transfer::{Self};
    use token_bridge::native_coin_witness::{Self, NATIVE_COIN_WITNESS};
    use token_bridge::native_coin_witness_v2::{Self, NATIVE_COIN_WITNESS_V2};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};

    use wormhole::state::{Self as wormhole_state, State as WormholeState};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    struct OTHER_COIN_WITNESS has drop {}

    #[test]
    fun test_complete_native_transfer(){
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
        };
        // create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_WITNESS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // complete transfer, sending native tokens to a recipient address
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 1000000000;
            let fee_amount = 100000000;
            let decimals = 10;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::get_chain_id(&worm_state);
            let to_chain = wormhole_state::get_chain_id(&worm_state);

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::test_complete_transfer<NATIVE_COIN_WITNESS>(
                my_transfer,
                &mut worm_state,
                &mut bridge_state,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // check balances after
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<NATIVE_COIN_WITNESS>>(&test, admin);
            assert!(coin::value<NATIVE_COIN_WITNESS>(&coins) == 900000000, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS>>(admin, coins);

            let fee_coins = take_from_address<Coin<NATIVE_COIN_WITNESS>>(&test, fee_recipient_person);
            assert!(coin::value<NATIVE_COIN_WITNESS>(&fee_coins) == 100000000, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS>>(fee_recipient_person, fee_coins);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_complete_native_transfer_10_decimals(){
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge
        next_tx(&mut test, admin);{
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            native_coin_witness::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_WITNESS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // complete transfer, sending native tokens to a recipient address
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            // dust at the end gets rounded to nothing, since 10-8=2 digits are lopped off
            let amount = 1000000079;
            let fee_amount = 100000000;
            let decimals = 10;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::get_chain_id(&worm_state);
            let to_chain = wormhole_state::get_chain_id(&worm_state);

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::test_complete_transfer<NATIVE_COIN_WITNESS>(
                my_transfer,
                &mut worm_state,
                &mut bridge_state,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // check balances after
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<NATIVE_COIN_WITNESS>>(&test, admin);
            assert!(coin::value<NATIVE_COIN_WITNESS>(&coins) == 900000000, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS>>(admin, coins);

            let fee_coins = take_from_address<Coin<NATIVE_COIN_WITNESS>>(&test, fee_recipient_person);
            assert!(coin::value<NATIVE_COIN_WITNESS>(&fee_coins) == 100000000, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS>>(fee_recipient_person, fee_coins);
        };
        test_scenario::end(test);
    }

    #[test]
    fun test_complete_native_transfer_4_decimals(){
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_witness_v2::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge
        next_tx(&mut test, admin);{
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS_V2>>(&test);
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS_V2>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS_V2>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS_V2>>(&test);
            let coins = coin::mint<NATIVE_COIN_WITNESS_V2>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_WITNESS_V2>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS_V2>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // complete transfer, sending native tokens to a recipient address
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 100;
            let fee_amount = 40;
            let decimals = 4;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::get_chain_id(&worm_state);
            let to_chain = wormhole_state::get_chain_id(&worm_state);

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::test_complete_transfer<NATIVE_COIN_WITNESS_V2>(
                my_transfer,
                &mut worm_state,
                &mut bridge_state,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // check balances after
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<NATIVE_COIN_WITNESS_V2>>(&test, admin);
            assert!(coin::value<NATIVE_COIN_WITNESS_V2>(&coins) == 60, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS_V2>>(admin, coins);

            let fee_coins = take_from_address<Coin<NATIVE_COIN_WITNESS_V2>>(&test, fee_recipient_person);
            assert!(coin::value<NATIVE_COIN_WITNESS_V2>(&fee_coins) == 40, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS_V2>>(fee_recipient_person, fee_coins);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = 4, location=token_bridge::state)] // E_ORIGIN_CHAIN_MISMATCH
    fun test_complete_native_transfer_wrong_origin_chain(){
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge
        next_tx(&mut test, admin);{
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            native_coin_witness::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            let coins =
                coin::mint<NATIVE_COIN_WITNESS>(
                    &mut t_cap,
                    10000000000, // amount
                    ctx(&mut test)
                );
            state::deposit<NATIVE_COIN_WITNESS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // attempt complete transfer
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 1000000000;
            let fee_amount = 100000000;
            let decimals = 8;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = 34; // wrong chain!
            let to_chain = wormhole_state::get_chain_id(&worm_state);

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::test_complete_transfer<NATIVE_COIN_WITNESS>(
                my_transfer,
                &mut worm_state,
                &mut bridge_state,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = 5, location=token_bridge::state)] // E_ORIGIN_ADDRESS_MISMATCH
    fun test_complete_native_transfer_wrong_coin_address(){
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            native_coin_witness::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_WITNESS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // attempt complete transfer
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 1000000000;
            let fee_amount = 100000000;
            let decimals = 8;
            let token_address = external_address::from_bytes(x"1111"); // wrong address!
            let token_chain = wormhole_state::get_chain_id(&worm_state);
            let to_chain = wormhole_state::get_chain_id(&worm_state);

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::test_complete_transfer<NATIVE_COIN_WITNESS>(
                my_transfer,
                &mut worm_state,
                &mut bridge_state,
                fee_recipient_person,
                ctx(&mut test)
            );

            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = 2, location=sui::balance)] // E_TOO_MUCH_FEE
    fun test_complete_native_transfer_too_much_fee(){
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            native_coin_witness::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_WITNESS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // attempt complete transfer
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 1000000000;
            let fee_amount = 1000000001; // Too much fee! Can't be greater than amount
            let decimals = 8;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::get_chain_id(&worm_state);
            let to_chain = wormhole_state::get_chain_id(&worm_state);

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::test_complete_transfer<NATIVE_COIN_WITNESS>(
                my_transfer,
                &mut worm_state,
                &mut bridge_state,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = 1, location=sui::dynamic_field)] // E_WRONG_COIN_TYPE
    fun test_complete_native_transfer_wrong_coin(){
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            native_coin_witness_v2::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            native_coin_witness::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_WITNESS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // attempt complete transfer with wrong coin type (NATIVE_COIN_WITNESS_V2)
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 1000000000;
            let fee_amount = 10000000;
            let decimals = 8;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::get_chain_id(&worm_state);
            let to_chain = wormhole_state::get_chain_id(&worm_state);

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::test_complete_transfer<NATIVE_COIN_WITNESS_V2>(
                my_transfer,
                &mut worm_state,
                &mut bridge_state,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }

    // the following test is for the "beefface" token from ethereum (chain id = 2),
    // which has 8 decimals
    #[test]
    fun complete_wrapped_transfer_test(){
        let (admin, fee_recipient_person, _) = people();
        let scenario = scenario();
        // First register foreign chain, create wrapped asset, register wrapped asset.
        let test = test_register_wrapped_(admin, scenario);
        next_tx(&mut test, admin);{
            coin_witness::test_init(ctx(&mut test));
        };
        // Complete transfer of wrapped asset from foreign chain to this chain.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 1000000000;
            let fee_amount = 100000000;
            let decimals = 8;
            let token_address = external_address::from_bytes(x"beefface");
            let token_chain = 2;
            let to_chain = wormhole_state::get_chain_id(&worm_state);

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );
            complete_transfer::test_complete_transfer<COIN_WITNESS>(
                my_transfer,
                &mut worm_state,
                &mut bridge_state,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // check balances after
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<COIN_WITNESS>>(&test, admin);
            assert!(coin::value<COIN_WITNESS>(&coins) == 900000000, 0);
            return_to_address<Coin<COIN_WITNESS>>(admin, coins);

            let fee_coins = take_from_address<Coin<COIN_WITNESS>>(&test, fee_recipient_person);
            assert!(coin::value<COIN_WITNESS>(&fee_coins) == 100000000, 0);
            return_to_address<Coin<COIN_WITNESS>>(fee_recipient_person, fee_coins);
        };
        test_scenario::end(test);
    }
}
