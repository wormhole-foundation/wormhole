module token_bridge::complete_transfer_with_payload {
    use sui::tx_context::{TxContext};
    use sui::coin::{Self, Coin, CoinMetadata};

    use wormhole::state::{State as WormholeState};
    use wormhole::external_address::{Self};

    use token_bridge::bridge_state::{Self, BridgeState, VerifiedCoinType};
    use token_bridge::vaa::{Self};
    //use token_bridge::transfer::{Self, Transfer};
    use token_bridge::transfer_with_payload::{Self, TransferWithPayload};
    use token_bridge::normalized_amount::{denormalize};

    const E_INVALID_TARGET: u64 = 0;

    public fun submit_vaa<CoinType>(
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        vaa: vector<u8>,
        ctx: &mut TxContext
    ): (Coin<CoinType>, TransferWithPayload) {
        let vaa = vaa::parse_verify_and_replay_protect(
            wormhole_state,
            bridge_state,
            vaa,
            ctx
        );

        let transfer = transfer_with_payload::parse(wormhole::myvaa::destroy(vaa));

        let token_chain = transfer_with_payload::get_token_chain(&transfer);
        let token_address = transfer_with_payload::get_token_address(&transfer);
        let verified_coin_witness = bridge_state::verify_coin_type<CoinType>(
            bridge_state,
            token_chain,
            token_address
        );

        complete_transfer_with_payload<CoinType>(
            verified_coin_witness,
            transfer,
            wormhole_state,
            bridge_state,
            coin_meta,
            ctx
        )
    }

    // complete transfer with arbitrary TransferWithPayload request and without the VAA
    // for native tokens
    #[test_only]
    public fun test_complete_transfer_with_payload<CoinType>(
        transfer: TransferWithPayload,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        ctx: &mut TxContext
    ): (Coin<CoinType>, TransferWithPayload) {
        let token_chain = transfer_with_payload::get_token_chain(&transfer);
        let token_address = transfer_with_payload::get_token_address(&transfer);
        let verified_coin_witness = bridge_state::verify_coin_type<CoinType>(
            bridge_state,
            token_chain,
            token_address
        );
        complete_transfer_with_payload<CoinType>(
            verified_coin_witness,
            transfer,
            wormhole_state,
            bridge_state,
            coin_meta,
            ctx
        )
    }

    fun complete_transfer_with_payload<CoinType>(
        verified_coin_witness: VerifiedCoinType<CoinType>,
        transfer: TransferWithPayload,
        wormhole_state: &mut WormholeState,
        bridge_state: &mut BridgeState,
        coin_meta: &CoinMetadata<CoinType>,
        ctx: &mut TxContext
    ): (Coin<CoinType>, TransferWithPayload) {
        let to_chain = transfer_with_payload::get_to_chain(&transfer);
        let this_chain = wormhole::state::get_chain_id(wormhole_state);
        assert!(to_chain == this_chain, E_INVALID_TARGET);

        let _recipient = external_address::to_address(&transfer_with_payload::get_to(&transfer));
        // TODO - should we pass an emitter cap into this function and assert that recipient==address(emitter_cap?)
        // Aptos reference: https://github.com/wormhole-foundation/wormhole/blob/main/aptos/token_bridge/sources/complete_transfer_with_payload.move#L37
        let decimals = coin::get_decimals(coin_meta);
        let amount = denormalize(transfer_with_payload::get_amount(&transfer), decimals);

        let recipient_coins;
        if (bridge_state::is_wrapped_asset<CoinType>(bridge_state)) {
            recipient_coins = bridge_state::mint<CoinType>(
                verified_coin_witness,
                bridge_state,
                amount,
                ctx
            );
        } else {
            recipient_coins = bridge_state::withdraw<CoinType>(
                verified_coin_witness,
                bridge_state,
                amount,
                ctx
            );
        };
        (recipient_coins, transfer)
    }
}

#[test_only]
module token_bridge::complete_transfer_with_payload_test {
    use std::bcs::{Self};

    use sui::test_scenario::{Self, Scenario, next_tx, return_shared, take_shared, ctx};
    use sui::coin::{Self, CoinMetadata};
    use sui::transfer::{Self};

    //use wormhole::myu16::{Self as u16};
    use wormhole::external_address::{Self};

    use token_bridge::normalized_amount::{Self};
    use token_bridge::transfer_with_payload::{Self, TransferWithPayload};
    use token_bridge::bridge_state::{Self, BridgeState};
    //use token_bridge::coin_witness::{Self, COIN_WITNESS};
    //use token_bridge::coin_witness_test::{test_register_wrapped_};
    use token_bridge::complete_transfer_with_payload::{Self};
    use token_bridge::native_coin_witness::{Self, NATIVE_COIN_WITNESS};
    use token_bridge::bridge_state_test::{set_up_wormhole_core_and_token_bridges};

    use wormhole::state::{Self as wormhole_state, State};

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
            let bridge_state = take_shared<BridgeState>(&test);
            let worm_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);
            bridge_state::register_native_asset<NATIVE_COIN_WITNESS>(
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            return_shared<BridgeState>(bridge_state);
            return_shared<State>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
        };
        // create a treasury cap for the native asset type, mint some tokens,
        // and deposit the native tokens into the token bridge
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<BridgeState>(&test);
            let worm_state = take_shared<State>(&test);
            let t_cap = take_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(&test);
            let coins = coin::mint<NATIVE_COIN_WITNESS>(&mut t_cap, 10000000000, ctx(&mut test));
            bridge_state::deposit<NATIVE_COIN_WITNESS>(&mut bridge_state, coins);
            return_shared<coin::TreasuryCap<NATIVE_COIN_WITNESS>>(t_cap);
            return_shared<BridgeState>(bridge_state);
            return_shared<State>(worm_state);
        };
        // complete transfer, sending native tokens to a recipient address
        next_tx(&mut test, admin); {
            let bridge_state = take_shared<BridgeState>(&test);
            let worm_state = take_shared<State>(&test);
            let coin_meta = take_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(&test);

            let to = admin;
            let amount = 1000000000;
            let decimals = 10;
            let token_address = external_address::from_bytes(x"01");
            let token_chain = wormhole_state::get_chain_id(&worm_state);
            let to_chain = wormhole_state::get_chain_id(&worm_state);
            let from_address = x"111122";
            let payload = x"beefbeef22";

            let transfer: TransferWithPayload = transfer_with_payload::create(
                normalized_amount::normalize(amount, decimals),
                token_address,
                token_chain,
                external_address::from_bytes(bcs::to_bytes(&to)),
                to_chain,
                external_address::from_bytes(from_address),
                payload
            );

            let (coins, _transfer_res) = complete_transfer_with_payload::test_complete_transfer_with_payload<NATIVE_COIN_WITNESS>(
                transfer,
                &mut worm_state,
                &mut bridge_state,
                &coin_meta,
                ctx(&mut test)
            );
            assert!(coin::value<NATIVE_COIN_WITNESS>(&coins) == 1000000000, 0);
            transfer::transfer(coins, admin);

            //TODO - make assertions on _transfer_res

            return_shared<BridgeState>(bridge_state);
            return_shared<State>(worm_state);
            return_shared<CoinMetadata<NATIVE_COIN_WITNESS>>(coin_meta);
        };
        test_scenario::end(test);
    }
}
