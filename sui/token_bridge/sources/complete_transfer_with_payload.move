module token_bridge::complete_transfer_with_payload {
    use sui::tx_context::{TxContext};
    use sui::coin::{Coin};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self};
    use wormhole::emitter::{Self, EmitterCapability};
    use wormhole::myvaa::{get_emitter_chain};

    use token_bridge::complete_transfer::{handle_complete_transfer};
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

        // Store the emitter chain ID to return to the caller.
        let emitter_chain_id = get_emitter_chain(&transfer_vaa);

        // Deserialize for processing.
        let my_transfer =
            transfer_with_payload::deserialize(
                wormhole::myvaa::destroy(transfer_vaa)
            );

        let recipient =
            external_address::to_address(
                &transfer_with_payload::recipient(&my_transfer)
            );

        // Transfer must be redeemed by the designated wormhole emitter.
        assert!(
            external_address::to_address(
                &emitter::get_external_address(emitter_cap)
            ) == recipient,
            E_INVALID_RECIPIENT
        );

        let (my_coins, _) =
            handle_complete_transfer<CoinType>(
                token_bridge_state,
                worm_state,
                transfer_with_payload::token_chain(&my_transfer),
                transfer_with_payload::token_address(&my_transfer),
                transfer_with_payload::recipient_chain(&my_transfer),
                transfer_with_payload::amount(&my_transfer),
                ctx
            );

        (my_coins, my_transfer, emitter_chain_id)
    }

    // TODO: remove this and make test with public method above
    #[test_only]
    public fun test_complete_transfer_with_payload<CoinType>(
        my_transfer: TransferWithPayload,
        worm_state: &mut WormholeState,
        token_bridge_state: &mut State,
        emitter_cap: &EmitterCapability,
        ctx: &mut TxContext
    ): (Coin<CoinType>, TransferWithPayload) {
        let recipient =
            external_address::to_address(
                &transfer_with_payload::recipient(&my_transfer)
            );

        // Transfer must be redeemed by the designated wormhole emitter.
        assert!(
            external_address::to_address(
                &emitter::get_external_address(emitter_cap)
            ) == recipient,
            E_INVALID_RECIPIENT
        );

        let (my_coins, _) =
            handle_complete_transfer<CoinType>(
                token_bridge_state,
                worm_state,
                transfer_with_payload::token_chain(&my_transfer),
                transfer_with_payload::token_address(&my_transfer),
                transfer_with_payload::recipient_chain(&my_transfer),
                transfer_with_payload::amount(&my_transfer),
                ctx
            );

        (my_coins, my_transfer)
    }
}

#[test_only]
module token_bridge::complete_transfer_with_payload_test {
    use sui::test_scenario::{
        Self, Scenario, next_tx, return_shared, take_shared, ctx
    };
    use sui::coin::{Self, CoinMetadata};
    use sui::transfer::{Self};

    use wormhole::external_address::{Self};

    use token_bridge::normalized_amount::{Self};
    use token_bridge::transfer_with_payload::{Self};
    use token_bridge::state::{Self, State};
    use token_bridge::complete_transfer_with_payload::{
        test_complete_transfer_with_payload
    };
    use token_bridge::native_coin_witness::{Self, NATIVE_COIN_WITNESS};
    use token_bridge::bridge_state_test::{
        set_up_wormhole_core_and_token_bridges
    };

    use wormhole::state::{Self as wormhole_state, State as WormholeState};
    use wormhole::wormhole::Self;

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

     #[test]
    fun test_complete_native_transfer(){
        let (admin, _, _) = people();
        let test = scenario();
        test = set_up_wormhole_core_and_token_bridges(admin, test);
        next_tx(&mut test, admin);{
            native_coin_witness::test_init(ctx(&mut test));
        };
        // register native asset type with the token bridge
        next_tx(&mut test, admin);{
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);
            let coin_meta =
                take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                &mut worm_state,
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
            let t_cap =
                take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            let coins =
                coin::mint<NATIVE_COIN_WITNESS>(
                    &mut t_cap,
                    10000000000, // amount
                    ctx(&mut test)
                );
            state::deposit<NATIVE_COIN_WITNESS>(
                &mut bridge_state,
                coins
            );
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        // complete transfer with payload (send native tokens + payload)
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<State>(&test);
            let worm_state = take_shared<WormholeState>(&test);

            let amount = 1000000000;
            let decimals = 10;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::get_chain_id(&worm_state);
            let to_chain = wormhole_state::get_chain_id(&worm_state);
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
                wormhole::register_emitter(&mut worm_state, ctx(&mut test));

            let (coins, transfer_res) =
                test_complete_transfer_with_payload<NATIVE_COIN_WITNESS>(
                    transfer,
                    &mut worm_state,
                    &mut bridge_state,
                    //&coin_meta,
                    &emitter_cap,
                    ctx(&mut test)
                );

            // assert coin value is as expected
            assert!(coin::value<NATIVE_COIN_WITNESS>(&coins) == 1000000000, 0);
            transfer::transfer(coins, admin);

            // assert payload and other fields are as expected
            assert!(
                normalized_amount::value(
                    &transfer_with_payload::amount(&transfer_res)
                ) == 10000000,
                0
            );
            assert!(
                transfer_with_payload::token_address(
                    &transfer_res
                ) == token_address,
                0
            );
            assert!(transfer_with_payload::token_chain(&transfer_res) == 21, 0);
            assert!(
                transfer_with_payload::recipient_chain(&transfer_res) == 21,
            0);
            assert!(
                transfer_with_payload::sender(&transfer_res)==from_address,
                0
            );
            assert!(transfer_with_payload::payload(&transfer_res)==payload, 0);

            transfer::transfer(emitter_cap, admin);
            return_shared<State>(bridge_state);
            return_shared<WormholeState>(worm_state);
        };
        test_scenario::end(test);
    }
}
