module token_bridge::complete_transfer {
    use sui::tx_context::{TxContext};
    use sui::coin::{Self, Coin};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::normalized_amount::{Self, NormalizedAmount};
    use token_bridge::state::{Self, State};
    use token_bridge::token_info::{Self};
    use token_bridge::transfer::{Self, Transfer};
    use token_bridge::vaa::{Self};

    // Requires `handle_complete_transfer`.
    friend token_bridge::complete_transfer_with_payload;

    const E_INVALID_TARGET: u64 = 0;
    const E_UNREGISTERED_TOKEN: u64 = 1;

    /// `complete_transfer` takes a verified Wormhole message and validates
    /// that this message was sent by a registered foreign Token Bridge contract
    /// and has a Token Bridge transfer payload.
    ///
    /// After processing the token transfer payload, coins are sent to the
    /// encoded recipient. If the specified `relayer` differs from this
    /// recipient, a relayer fee is split from this coin and sent to `relayer`.
    public entry fun complete_transfer<CoinType>(
        token_bridge_state: &mut State,
        worm_state: &mut WormholeState,
        vaa: vector<u8>,
        relayer: address,
        ctx: &mut TxContext
    ) {
        // Parse and verify Token Bridge transfer message. This method
        // guarantees that a verified transfer message cannot be redeemed again.
        let transfer_vaa =
            vaa::parse_verify_and_replay_protect(
                token_bridge_state,
                worm_state,
                vaa,
                ctx
            );

        // Deserialize transfer message and process.
        handle_complete_transfer<CoinType>(
            token_bridge_state,
            &transfer::deserialize(wormhole::myvaa::destroy(transfer_vaa)),
            relayer,
            ctx
        )
    }

    /// `verify_transfer_details` is only friendly with this module and the
    /// `complete_transfer` module. For inbound transfers, the deserialized
    /// transfer message needs to be validated.
    ///
    /// This method also de-normalizes the amount encoded in the transfer based
    /// on the coin's decimals.
    ///
    /// Depending on whether this coin is a Token Bridge wrapped asset or a
    /// natively existing asset on Sui, the coin is either minted or withdrawn
    /// from Token Bridge's custody.
    public(friend) fun verify_transfer_details<CoinType>(
        token_bridge_state: &mut State,
        token_chain: u16,
        token_address: ExternalAddress,
        recipient_chain: u16,
        amount: NormalizedAmount,
        ctx: &mut TxContext
    ): (Coin<CoinType>, u8) {
        // Verify that the intended chain ID for this transfer is for Sui.
        assert!(
            recipient_chain == wormhole::state::chain_id(),
            E_INVALID_TARGET
        );

        // Get info about the transferred token.
        let info = state::token_info<CoinType>(token_bridge_state);

        // Verify that the token info agrees with the info encoded in this
        // transfer.
        assert!(
            token_info::equals(&info, token_chain, token_address),
            E_UNREGISTERED_TOKEN
        );

        let decimals = state::coin_decimals<CoinType>(token_bridge_state);

        // If the token is wrapped by Token Bridge, we will mint these tokens.
        // Otherwise, we will withdraw from custody.
        let token_coin = {
            if (token_info::is_wrapped(&info)) {
                state::mint<CoinType>(
                    token_bridge_state,
                    normalized_amount::to_raw(amount, decimals),
                    ctx
                )
            } else {
                state::withdraw<CoinType>(
                    token_bridge_state,
                    normalized_amount::to_raw(amount, decimals),
                    ctx
                )
            }
        };

        (token_coin, decimals)
    }

    fun handle_complete_transfer<CoinType>(
        token_bridge_state: &mut State,
        parsed_transfer: &Transfer,
        relayer: address,
        ctx: &mut TxContext
    ) {
        let (my_coins, decimals) =
            verify_transfer_details<CoinType>(
                token_bridge_state,
                transfer::token_chain(parsed_transfer),
                transfer::token_address(parsed_transfer),
                transfer::recipient_chain(parsed_transfer),
                transfer::amount(parsed_transfer),
                ctx
            );

        let recipient =
            external_address::to_address(
                &transfer::recipient(parsed_transfer)
            );

        // If the recipient did not redeem his own transfer, Token Bridge will
        // split the withdrawn coins and send a portion to the transaction
        // relayer.
        if (recipient != relayer) {
            let fee =
                normalized_amount::to_raw(
                    transfer::relayer_fee(parsed_transfer),
                    decimals
                );
            sui::transfer::transfer(
                coin::split(&mut my_coins, fee, ctx),
                relayer
            );
        };

        // Finally transfer tokens to the recipient.
        sui::transfer::transfer(my_coins, recipient);
    }


    #[test_only]
    /// This method is exists to expose `handle_complete_transfer` and validate
    /// its job. `handle_complete_transfer` is used by `complete_transfer`.
    public fun complete_transfer_test_only<CoinType>(
        token_bridge_state: &mut State,
        _worm_state: &mut WormholeState,
        parsed_transfer: Transfer,
        relayer: address,
        ctx: &mut TxContext
    ) {
        handle_complete_transfer<CoinType>(
            token_bridge_state,
            &parsed_transfer,
            relayer,
            ctx
        )
    }
}

#[test_only]
module token_bridge::complete_transfer_test {
    use std::bcs::{Self};

    use sui::test_scenario::{Self, Scenario, next_tx, return_shared,
        take_shared, ctx, take_from_address, return_to_address};
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
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &coin_meta,
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            let coins = coin::mint<NATIVE_COIN_WITNESS>(
                &mut t_cap,
                10000000000,
                ctx(&mut test)
            );
            state::deposit<NATIVE_COIN_WITNESS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Complete transfer, sending native tokens to a recipient address.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 1000000000;
            let fee_amount = 100000000;
            let decimals = 10;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::chain_id();
            let to_chain = wormhole_state::chain_id();

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::complete_transfer_test_only<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                my_transfer,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<NATIVE_COIN_WITNESS>>(
                &test,
                admin
            );
            assert!(coin::value<NATIVE_COIN_WITNESS>(&coins) == 900000000, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS>>(admin, coins);

            let fee_coins = take_from_address<Coin<NATIVE_COIN_WITNESS>>(
                &test,
                fee_recipient_person
            );
            assert!(coin::value<NATIVE_COIN_WITNESS>(&fee_coins)
                == 100000000, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS>>(
                fee_recipient_person,
                fee_coins
            );
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
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(
                &test
            );
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &coin_meta,
            );
            native_coin_witness::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(
                &test
            );
            let coins = coin::mint<NATIVE_COIN_WITNESS>(
                &mut t_cap,
                10000000000,
                ctx(&mut test)
            );
            state::deposit<NATIVE_COIN_WITNESS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Complete transfer, sending native tokens to a recipient address.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            // Dust at the end gets rounded to nothing, since 10-8=2 digits
            // are lopped off.
            let amount = 1000000079;
            let fee_amount = 100000000;
            let decimals = 10;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::chain_id();
            let to_chain = wormhole_state::chain_id();

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::complete_transfer_test_only<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                my_transfer,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<NATIVE_COIN_WITNESS>>(
                &test,
                admin
            );
            assert!(coin::value<NATIVE_COIN_WITNESS>(&coins) == 900000000, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS>>(admin, coins);

            let fee_coins = take_from_address<Coin<NATIVE_COIN_WITNESS>>(
                &test,
                fee_recipient_person
            );
            assert!(coin::value<NATIVE_COIN_WITNESS>(&fee_coins)
                == 100000000, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS>>(
                fee_recipient_person,
                fee_coins
            );
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
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS_V2>>(
                &test
            );
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS_V2>(
                &mut bridge_state,
                &coin_meta,
            );
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS_V2>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS_V2>>(
                &test
            );
            let coins = coin::mint<NATIVE_COIN_WITNESS_V2>(
                &mut t_cap,
                10000000000,
                ctx(&mut test)
            );
            state::deposit<NATIVE_COIN_WITNESS_V2>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS_V2>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Complete transfer, sending native tokens to a recipient address.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 100;
            let fee_amount = 40;
            let decimals = 4;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::chain_id();
            let to_chain = wormhole_state::chain_id();

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::complete_transfer_test_only<NATIVE_COIN_WITNESS_V2>(
                &mut bridge_state,
                &mut worm_state,
                my_transfer,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // check balances after
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<NATIVE_COIN_WITNESS_V2>>(
                &test,
                admin
            );
            assert!(coin::value<NATIVE_COIN_WITNESS_V2>(&coins) == 60, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS_V2>>(admin, coins);

            let fee_coins = take_from_address<Coin<NATIVE_COIN_WITNESS_V2>>(
                &test,
                fee_recipient_person
            );
            assert!(coin::value<NATIVE_COIN_WITNESS_V2>(&fee_coins) == 40, 0);
            return_to_address<Coin<NATIVE_COIN_WITNESS_V2>>(
                fee_recipient_person,
                fee_coins
            );
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::complete_transfer::E_UNREGISTERED_TOKEN,
        location=token_bridge::complete_transfer
    )]
    fun test_complete_native_transfer_wrong_origin_chain(){
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(
                &test
            );
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &coin_meta
            );
            native_coin_witness::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(
                &test
            );
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
            let to_chain = wormhole_state::chain_id();

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::complete_transfer_test_only<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                my_transfer,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::complete_transfer::E_UNREGISTERED_TOKEN,
        location=token_bridge::complete_transfer
    )]
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
                &mut bridge_state,
                &coin_meta,
            );
            native_coin_witness::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(
                &test
            );
            let coins = coin::mint<NATIVE_COIN_WITNESS>(
                &mut t_cap,
                10000000000,
                ctx(&mut test)
            );
            state::deposit<NATIVE_COIN_WITNESS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Attempt complete transfer.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 1000000000;
            let fee_amount = 100000000;
            let decimals = 8;
            let token_address = external_address::from_bytes(x"1111"); // wrong address!
            let token_chain = wormhole_state::chain_id();
            let to_chain = wormhole_state::chain_id();

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::complete_transfer_test_only<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                my_transfer,
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
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &coin_meta,
            );
            native_coin_witness::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
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
            let token_chain = wormhole_state::chain_id();
            let to_chain = wormhole_state::chain_id();

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::complete_transfer_test_only<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                my_transfer,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::registered_tokens::E_UNREGISTERED,
        location = token_bridge::registered_tokens
    )]
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
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &coin_meta,
            );
            native_coin_witness::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
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
        // Attempt complete transfer with wrong coin type (NATIVE_COIN_WITNESS_V2).
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let to = admin;
            let amount = 1000000000;
            let fee_amount = 10000000;
            let decimals = 8;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::chain_id();
            let to_chain = wormhole_state::chain_id();

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );

            complete_transfer::complete_transfer_test_only<NATIVE_COIN_WITNESS_V2>(
                &mut bridge_state,
                &mut worm_state,
                my_transfer,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }

    // The following test is for the "beefface" token from ethereum (chain id = 2),
    // which has 8 decimals.
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
            let to_chain = wormhole_state::chain_id();

            let my_transfer = transfer::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                normalized_amount::from_raw(fee_amount, decimals),
            );
            complete_transfer::complete_transfer_test_only<COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
                my_transfer,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
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
