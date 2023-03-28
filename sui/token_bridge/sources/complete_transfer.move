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
            &transfer::deserialize(wormhole::vaa::take_payload(transfer_vaa)),
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
                transfer::recipient(parsed_transfer)
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
            sui::transfer::public_transfer(
                coin::split(&mut my_coins, fee, ctx),
                relayer
            );
        };

        // Finally transfer tokens to the recipient.
        sui::transfer::public_transfer(my_coins, recipient);
    }
}

#[test_only]
module token_bridge::complete_transfer_test {
    use sui::test_scenario::{Self, Scenario, next_tx, return_shared,
        take_shared, ctx, take_from_address, return_to_address};
    use sui::coin::{Self, Coin, CoinMetadata};


    use token_bridge::state::{Self, State};
    use token_bridge::wrapped_coin_12_decimals::{Self, WRAPPED_COIN_12_DECIMALS};
    use token_bridge::wrapped_coin_12_decimals_test::{test_register_wrapped_};
    use token_bridge::complete_transfer::{Self};
    use token_bridge::native_coin_10_decimals::{Self, NATIVE_COIN_10_DECIMALS};
    use token_bridge::native_coin_4_decimals::{Self, NATIVE_COIN_4_DECIMALS};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};
    use token_bridge::register_chain::{Self};

    use wormhole::state::{State as WormholeState};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    struct OTHER_COIN_WITNESS has drop {}

    /// Registration VAA for the etheruem token bridge 0xdeadbeef.
    const ETHEREUM_TOKEN_REG: vector<u8> =
        x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    /// Used for test_complete_native_transfer and test_complete_native_transfer_wrong_coin
    const VAA_NATIVE_TRANSFER: vector<u8> = x"010000000001002ff4303d38fe2eade48868ab51d31e21c32303d512501b26bfb6a3da9e9a41635ee41360458471d7b2af59b9b2cd48a11bea714e147e0ae76d0182e1af56bf41000000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f010000000000000000000000000000000000000000000000000000000000000bb8000000000000000000000000000000000000000000000000000000000000000100150000000000000000000000000000000000000000000000000000000000124323001500000000000000000000000000000000000000000000000000000000000003e8";
    // ============================ VAA details ============================
    // emitterChain: 2,
    // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
    // module: 'TokenBridge',
    // type: 'Transfer',
    // amount: 3000n,
    // tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000001',
    // tokenChain: 21,
    // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
    // chain: 21,
    // fee: 1000n
    // ============================ VAA details ============================

    #[test]
    /// An end-to-end test for complete transer native with VAA.
    fun test_complete_native_transfer_10_decimals(){
        {
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // Create coin.
        next_tx(&mut test, admin);{
            native_coin_10_decimals::test_init(ctx(&mut test));
        };
        // Register eth token bridge (where transfer VAA will originate from)
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            register_chain::submit_vaa(&mut bridge_state, &mut wormhole_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
        };
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(&test);
            state::register_native_asset<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &coin_meta,
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(coin_meta);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(&test);
            let coins = coin::mint<NATIVE_COIN_10_DECIMALS>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_10_DECIMALS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Complete transfer, sending native tokens to a recipient address.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            complete_transfer::complete_transfer<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                VAA_NATIVE_TRANSFER,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<NATIVE_COIN_10_DECIMALS>>(&test, admin);
            assert!(coin::value<NATIVE_COIN_10_DECIMALS>(&coins) == 200000, 0);
            return_to_address<Coin<NATIVE_COIN_10_DECIMALS>>(admin, coins);

            let fee_coins = take_from_address<Coin<NATIVE_COIN_10_DECIMALS>>(&test, fee_recipient_person);
            assert!(coin::value<NATIVE_COIN_10_DECIMALS>(&fee_coins) == 100000, 0);
            return_to_address<Coin<NATIVE_COIN_10_DECIMALS>>(fee_recipient_person, fee_coins);
        };
        test_scenario::end(test);
    }
    }

    #[test]
    /// An end-to-end test for complete transer native with VAA.
    fun test_complete_native_transfer_4_decimals(){
        {
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // Create coin.
        next_tx(&mut test, admin);{
            native_coin_4_decimals::test_init(ctx(&mut test));
        };
        // Register eth token bridge (where transfer VAA will originate from)
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            register_chain::submit_vaa(&mut bridge_state, &mut wormhole_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
        };
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_4_DECIMALS>>(&test);
            state::register_native_asset<NATIVE_COIN_4_DECIMALS>(
                &mut bridge_state,
                &coin_meta,
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_4_DECIMALS>>(coin_meta);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_4_DECIMALS>>(&test);
            let coins = coin::mint<NATIVE_COIN_4_DECIMALS>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_4_DECIMALS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_4_DECIMALS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Complete transfer, sending native tokens to a recipient address.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            complete_transfer::complete_transfer<NATIVE_COIN_4_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                VAA_NATIVE_TRANSFER,
                fee_recipient_person,
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<NATIVE_COIN_4_DECIMALS>>(&test, admin);
            assert!(coin::value<NATIVE_COIN_4_DECIMALS>(&coins) == 2000, 0);
            return_to_address<Coin<NATIVE_COIN_4_DECIMALS>>(admin, coins);

            let fee_coins = take_from_address<Coin<NATIVE_COIN_4_DECIMALS>>(&test, fee_recipient_person);
            assert!(coin::value<NATIVE_COIN_4_DECIMALS>(&fee_coins) == 1000, 0);
            return_to_address<Coin<NATIVE_COIN_4_DECIMALS>>(fee_recipient_person, fee_coins);
        };
        test_scenario::end(test);
    }
    }

    #[test]
    #[expected_failure(
        abort_code = token_bridge::complete_transfer::E_UNREGISTERED_TOKEN,
        location=token_bridge::complete_transfer
    )]
    /// A negative test for the internal function handle_complete_transfer.
    /// Test that transfer fails if the origin chain is not specified correctly,
    /// causing the bridge to think that the token is unregistered.
    fun test_complete_native_transfer_wrong_origin_chain(){
        let wrong_origin_chain_vaa = x"01000000000100b0d67f0102856458dc68a29e88e5574f4b0b129841522066adbf275f0749129a319bc6a94a7b7c8822e91e1ceb2b1e22adee0d283fba5b97dab5f7165216785e010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f01000000000000000000000000000000000000000000000000000000003b9aca4f00000000000000000000000000000000000000000000000000000000000000010017000000000000000000000000000000000000000000000000000000000012432300150000000000000000000000000000000000000000000000000000000005f5e100";
        // ============================ VAA details ============================
        // emitterChain: 2,
        // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
        // module: 'TokenBridge',
        // type: 'Transfer',
        // amount: 1000000079n,
        // tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000001',
        // tokenChain: 23, // Wrong chain! Should be 21.
        // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
        // chain: 21,
        // fee: 100000000n
        // ============================ VAA details ============================
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_10_decimals::test_init(ctx(&mut test));
        };
        // Register eth token bridge (where transfer VAA will originate from)
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            register_chain::submit_vaa(&mut bridge_state, &mut wormhole_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
        };
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(&test);
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            state::register_native_asset<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &coin_meta
            );
            native_coin_10_decimals::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(&test);
            let coins =
                coin::mint<NATIVE_COIN_10_DECIMALS>(
                    &mut t_cap,
                    10000000000, // amount
                    ctx(&mut test)
                );
            state::deposit<NATIVE_COIN_10_DECIMALS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Attempt complete transfer. Fails because the origin chain of the token
        // in the VAA is not specified correctly (though the address is the native-Sui
        // of the previously registered native token).
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            complete_transfer::complete_transfer<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                wrong_origin_chain_vaa,
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
        let vaa_transfer_wrong_address = x"010000000001008490d3e139f3b705282df4686907dfff358dd365e7471d8d6793ade61a27d33d48fc665198bc8022bebd8f8a91f29b3df75455180bf3b2d39eb97f93be3a8caf000000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f01000000000000000000000000000000000000000000000000000000003b9aca4f00000000000000000000000000000000000000000000000000000000000004440015000000000000000000000000000000000000000000000000000000000012432300150000000000000000000000000000000000000000000000000000000005f5e100";
        // ============================ VAA details ============================
        //   emitterChain: 2,
        //   emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
        //   sequence: 1n,
        //   consistencyLevel: 15,
        //   module: 'TokenBridge',
        //   type: 'Transfer',
        //   amount: 1000000079n,
        //   tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000444', // Wrong!
        //   tokenChain: 21,
        //   toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
        //   chain: 21,
        //   fee: 100000000n
        // ============================ VAA details ============================
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_10_decimals::test_init(ctx(&mut test));
        };
        // Register eth token bridge (where transfer VAA will originate from)
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            register_chain::submit_vaa(&mut bridge_state, &mut wormhole_state,  ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
        };
        // register native asset type with the token bridge
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(&test);
            state::register_native_asset<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &coin_meta,
            );
            native_coin_10_decimals::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(&test);
            let coins = coin::mint<NATIVE_COIN_10_DECIMALS>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_10_DECIMALS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Attempt complete transfer.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            complete_transfer::complete_transfer<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                vaa_transfer_wrong_address,
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
        let vaa_transfer_too_much_fee = x"01000000000100032a439b4cf8f793e2a0b4281344bc81af9ff9f118ff9e320cbde49b072f23a5500963f2e66632143a97f2f0a5bc5370f006d2382f2e06c09b542476b02d099e000000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f01000000000000000000000000000000000000000000000000000000003b9aca000000000000000000000000000000000000000000000000000000000000000001001500000000000000000000000000000000000000000000000000000000001243230015000000000000000000000000000000000000000000000000000009184e72a000";
        // ============================ VAA details ============================
        //   emitterChain: 2,
        //   emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
        //   module: 'TokenBridge',
        //   type: 'Transfer',
        //   amount: 1000000000n,
        //   tokenAddress: '0x0000000000000000000000000000000000000000000000000000000000000001',
        //   tokenChain: 21,
        //   toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
        //   chain: 21,
        //   fee: 10000000000000n // Wrong! Way too much fee!
        // ============================ VAA details ============================
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_10_decimals::test_init(ctx(&mut test));
        };
        // Register eth token bridge (where transfer VAA will originate from)
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            register_chain::submit_vaa(&mut bridge_state, &mut wormhole_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
        };
        // Register native asset type with the token bridge.
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(&test);
            state::register_native_asset<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &coin_meta,
            );
            native_coin_10_decimals::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(&test);
            let coins = coin::mint<NATIVE_COIN_10_DECIMALS>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_10_DECIMALS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // attempt complete transfer
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            complete_transfer::complete_transfer<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                vaa_transfer_too_much_fee,
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
    /// In this test, the generic CoinType arg to complete_transfer
    /// is not specified correctly, causing the token bridge
    /// to not recignize that coin as being registered and for
    /// the complete transfer to fail.
    fun test_complete_native_transfer_wrong_coin(){
        let (admin, fee_recipient_person, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        // Register eth token bridge (where transfer VAA will originate from)
        next_tx(&mut test, admin); {
            let wormhole_state = take_shared<WormholeState>(&test);
            let bridge_state = take_shared<State>(&test);
            register_chain::submit_vaa(&mut bridge_state, &mut wormhole_state, ETHEREUM_TOKEN_REG, ctx(&mut test));
            return_shared<WormholeState>(wormhole_state);
            return_shared<State>(bridge_state);
        };
        next_tx(&mut test, admin);{
            native_coin_10_decimals::test_init(ctx(&mut test));
        };
        next_tx(&mut test, admin);{
            native_coin_4_decimals::test_init(ctx(&mut test));
        };
        // Register native asset type NATIVE_COIN_10_DECIMALS with the token bridge.
        // Note that NATIVE_COIN_4_DECIMALS is not registered!
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(&test);
            state::register_native_asset<NATIVE_COIN_10_DECIMALS>(
                &mut bridge_state,
                &coin_meta,
            );
            native_coin_10_decimals::test_init(ctx(&mut test));
            return_shared<CoinMetadata<NATIVE_COIN_10_DECIMALS>>(coin_meta);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(&test);
            let coins = coin::mint<NATIVE_COIN_10_DECIMALS>(&mut t_cap, 10000000000, ctx(&mut test));
            state::deposit<NATIVE_COIN_10_DECIMALS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_10_DECIMALS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // Attempt complete transfer with wrong coin type (NATIVE_COIN_4_DECIMALS).
        // Fails because NATIVE_COIN_4_DECIMALS is unregistered.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            complete_transfer::complete_transfer<NATIVE_COIN_4_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                VAA_NATIVE_TRANSFER,
                fee_recipient_person,
                ctx(&mut test)
            );

            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }

    #[test]
    /// Test the external-facing function complete_transfer using a real VAA.
    /// Check balance of recipient (admin = @0x124323) after complete_transfer
    /// is called.
    fun complete_wrapped_transfer_external_no_fee_test(){
        // transfer token VAA (details below)
        let vaa = x"010000000001001d7c73fe9d1fd168fd8b4e767557724f82c56469560ccffd5f5dc6c49afd15007e27d1ed19a83ae11c68b28e848d67e8674ad29b19ee8d097c3ccc78ab292813010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f010000000000000000000000000000000000000000000000000000000000000bb800000000000000000000000000000000000000000000000000000000beefface0002000000000000000000000000000000000000000000000000000000000012432300150000000000000000000000000000000000000000000000000000000000000000";
        // ============================ VAA details ============================
        // emitterChain: 2,
        // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
        // module: 'TokenBridge',
        // type: 'Transfer',
        // amount: 3000n,
        // tokenAddress: '0x00000000000000000000000000000000000000000000000000000000beefface',
        // tokenChain: 2,
        // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
        // chain: 21,
        // fee: 0n
        // ============================ VAA details ============================
        let (admin, _fee_recipient_person, _) = people();
        let scenario = scenario();
        // First register foreign chain, create wrapped asset, register wrapped asset.
        let test = test_register_wrapped_(admin, scenario);
        next_tx(&mut test, admin);{
            wrapped_coin_12_decimals::test_init(ctx(&mut test));
        };
        // Complete transfer of wrapped asset from foreign chain to this chain.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            complete_transfer::complete_transfer<WRAPPED_COIN_12_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                vaa,
                @0x0, // relayer
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<WRAPPED_COIN_12_DECIMALS>>(&test, admin);
            assert!(coin::value<WRAPPED_COIN_12_DECIMALS>(&coins) == 3000, 0);
            return_to_address<Coin<WRAPPED_COIN_12_DECIMALS>>(admin, coins);
        };
        test_scenario::end(test);
    }

    #[test]
    /// Test the external-facing function complete_transfer using a real VAA.
    /// This time include a relayer fee when calling complete_transfer.
    fun complete_wrapped_transfer_external_relayer_fee_test(){
        // transfer token VAA (details below)
        let vaa = x"01000000000100f90d171a2c4ffde9cf214ce2ed94b384e5ee8384ef3129e1342bf6b10db8201122fb9ff501e9f28367a48f191cf5fb8ff51ff58e8091745a392ec8053a05b55e010000000000000000000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f010000000000000000000000000000000000000000000000000000000000000bb800000000000000000000000000000000000000000000000000000000beefface00020000000000000000000000000000000000000000000000000000000000124323001500000000000000000000000000000000000000000000000000000000000003e8";
        // ============================ VAA details ============================
        // emitterChain: 2,
        // emitterAddress: '0x00000000000000000000000000000000000000000000000000000000deadbeef',
        // module: 'TokenBridge',
        // type: 'Transfer',
        // amount: 3000n,
        // tokenAddress: '0x00000000000000000000000000000000000000000000000000000000beefface',
        // tokenChain: 2,
        // toAddress: '0x0000000000000000000000000000000000000000000000000000000000124323',
        // chain: 21,
        // fee: 1000n
        // ============================ VAA details ============================
        let (admin, fee_recipient_person, _) = people();
        let scenario = scenario();
        // First register foreign chain, create wrapped asset, register wrapped asset.
        let test = test_register_wrapped_(admin, scenario);
        next_tx(&mut test, admin);{
            wrapped_coin_12_decimals::test_init(ctx(&mut test));
        };
        // Complete transfer of wrapped asset from foreign chain to this chain.
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            complete_transfer::complete_transfer<WRAPPED_COIN_12_DECIMALS>(
                &mut bridge_state,
                &mut worm_state,
                vaa,
                fee_recipient_person, // relayer
                ctx(&mut test)
            );
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };

        // Check balances after.
        next_tx(&mut test, admin);{
            let coins = take_from_address<Coin<WRAPPED_COIN_12_DECIMALS>>(&test, admin);
            assert!(coin::value<WRAPPED_COIN_12_DECIMALS>(&coins) == 2000, 0);
            return_to_address<Coin<WRAPPED_COIN_12_DECIMALS>>(admin, coins);

            let fee_coins = take_from_address<Coin<WRAPPED_COIN_12_DECIMALS>>(&test, fee_recipient_person);
            assert!(coin::value<WRAPPED_COIN_12_DECIMALS>(&fee_coins) == 1000, 0);
            return_to_address<Coin<WRAPPED_COIN_12_DECIMALS>>(fee_recipient_person, fee_coins);
        };
        test_scenario::end(test);
    }
}
