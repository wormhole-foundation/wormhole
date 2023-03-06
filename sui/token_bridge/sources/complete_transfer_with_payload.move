module token_bridge::complete_transfer_with_payload {
    use sui::tx_context::{TxContext};
    use sui::coin::{Coin};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self};
    use wormhole::emitter::{Self, EmitterCapability};
    use wormhole::myvaa::{get_emitter_chain};

    use token_bridge::complete_transfer::{verify_transfer_details};
    use token_bridge::state::{State};
    use token_bridge::transfer_with_payload::{Self, TransferWithPayload};
    use token_bridge::vaa::{Self};

    const E_INVALID_TARGET: u64 = 0;
    const E_INVALID_RECIPIENT: u64 = 1;

    public fun complete_transfer_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCapability,
        worm_state: &mut WormholeState,
        vaa: vector<u8>,
        ctx: &mut TxContext
    ): (Coin<CoinType>, TransferWithPayload, u16) {
        // Parse and verify Token Bridge transfer message. This method
        // guarantees that a verified transfer message cannot be redeemed again.
        let transfer_vaa =
            vaa::parse_verify_and_replay_protect(
                token_bridge_state,
                worm_state,
                vaa,
                ctx
            );

        // Before destroying VAA, store the emitter chain ID for the caller.
        let source_chain = get_emitter_chain(&transfer_vaa);

        // Deserialize for processing.
        let parsed_transfer =
            transfer_with_payload::deserialize(
                wormhole::myvaa::destroy(transfer_vaa)
            );

        let token_coin =
            handle_complete_transfer_with_payload(
                token_bridge_state,
                emitter_cap,
                &parsed_transfer,
                ctx
            );

        (token_coin, parsed_transfer, source_chain)
    }

    fun handle_complete_transfer_with_payload<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCapability,
        parsed_transfer: &TransferWithPayload,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        let recipient =
            external_address::to_address(
                &transfer_with_payload::recipient(parsed_transfer)
            );

        // Transfer must be redeemed by the contract's registered Wormhole
        // emitter.
        assert!(
            external_address::to_address(
                &emitter::get_external_address(emitter_cap)
            ) == recipient,
            E_INVALID_RECIPIENT
        );

        let (token_coin, _) =
            verify_transfer_details<CoinType>(
                token_bridge_state,
                transfer_with_payload::token_chain(parsed_transfer),
                transfer_with_payload::token_address(parsed_transfer),
                transfer_with_payload::recipient_chain(parsed_transfer),
                transfer_with_payload::amount(parsed_transfer),
                ctx
            );

        token_coin
    }

    #[test_only]
    /// This method is exists to expose `handle_complete_transfer_with_payload`
    /// and validate its job. `handle_complete_transfer_with_payload` is used by
    /// `complete_transfer_with_payload`.
    public fun complete_transfer_with_payload_test_only<CoinType>(
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCapability,
        _worm_state: &mut WormholeState,
        parsed_transfer: TransferWithPayload,
        ctx: &mut TxContext
    ): Coin<CoinType> {
        handle_complete_transfer_with_payload<CoinType>(
                token_bridge_state,
                emitter_cap,
                &parsed_transfer,
                ctx
            )
    }
}

#[test_only]
module token_bridge::complete_transfer_with_payload_test {
    use sui::coin::{Self, CoinMetadata};
    use sui::test_scenario::{Self, Scenario};
    use wormhole::external_address::{Self};
    use wormhole::state::{Self as wormhole_state, State as WormholeState};
    use wormhole::wormhole::{Self};

    use token_bridge::bridge_state_test::{
        set_up_wormhole_core_and_token_bridges
    };
    use token_bridge::complete_transfer_with_payload::{
        complete_transfer_with_payload_test_only
    };
    use token_bridge::native_coin_witness::{Self, NATIVE_COIN_WITNESS};
    use token_bridge::normalized_amount::{Self};
    use token_bridge::state::{Self, State};
    use token_bridge::transfer_with_payload::{Self};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    fun test_complete_native_transfer(){
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        test_scenario::next_tx(&mut test, admin);{
            native_coin_witness::test_init(test_scenario::ctx(&mut test));
        };
        // register native asset type with the token bridge
        test_scenario::next_tx(&mut test, admin);{
            let bridge_state = test_scenario::take_shared<State>(&test);
            let worm_state = test_scenario::take_shared<WormholeState>(&test);
            let coin_meta =
                test_scenario::take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &coin_meta,
            );
            test_scenario::return_shared<State>(bridge_state);
            test_scenario::return_shared<WormholeState>(worm_state);
            test_scenario::return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
        };
        // create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge
        test_scenario::next_tx(&mut test, admin); {
            let bridge_state = test_scenario::take_shared<State>(&test);
            let worm_state = test_scenario::take_shared<WormholeState>(&test);
            let t_cap =
                test_scenario::take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            let coins =
                coin::mint<NATIVE_COIN_WITNESS>(
                    &mut t_cap,
                    10000000000, // amount
                    test_scenario::ctx(&mut test)
                );
            state::deposit<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                coins
            );
            test_scenario::return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            test_scenario::return_shared<State>(bridge_state);
            test_scenario::return_shared<WormholeState>(worm_state);
        };
        // complete transfer with payload (send native tokens + payload)
        test_scenario::next_tx(&mut test, admin); {
            let bridge_state = test_scenario::take_shared<State>(&test);
            let worm_state = test_scenario::take_shared<WormholeState>(&test);

            let amount = 1000000000;
            let decimals = 10;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::chain_id();
            let to_chain = wormhole_state::chain_id();
            // The emitter_cap defined below corresponds to the second wormhole-
            // registered emitter. As per naming conventions, we know that the
            // address of the emitter is precisely "0x2".
            let to = external_address::from_bytes(x"02");
            let from_address = external_address::from_bytes(x"111122");
            let payload = x"beefbeef22";

            let transfer = transfer_with_payload::new(
                normalized_amount::from_raw(amount, decimals),
                token_address,
                token_chain,
                to,
                to_chain,
                from_address,
                payload
            );

            let emitter_cap =
                wormhole::register_emitter(
                    &mut worm_state, test_scenario::ctx(&mut test)
                );

            let token_coins =
                complete_transfer_with_payload_test_only<NATIVE_COIN_WITNESS>(
                    &mut bridge_state,
                    &emitter_cap,
                    &mut worm_state,
                    transfer,
                    test_scenario::ctx(&mut test)
                );

            // assert coin value is as expected
            assert!(coin::value(&token_coins) == amount, 0);
            test_scenario::return_shared<State>(bridge_state);
            test_scenario::return_shared<WormholeState>(worm_state);

            // Trash remaining objects.
            sui::transfer::transfer(token_coins, @0x0);
            sui::transfer::transfer(emitter_cap, @0x0);
        };
        test_scenario::end(test);
    }
}
