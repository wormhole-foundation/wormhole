// Example wrapped coin for testing purposes

#[test_only]
module coins::coin {
    use sui::object::{Self};
    use sui::package::{Self};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use token_bridge::create_wrapped::{Self};

    struct COIN has drop {}

    fun init(witness: COIN, ctx: &mut TxContext) {
        use token_bridge::version_control::{V__0_2_0 as V__CURRENT};

        transfer::public_transfer(
            create_wrapped::prepare_registration<COIN, V__CURRENT>(
                witness,
                // TODO: create a version of this for each decimal to be used
                8,
                ctx
            ),
            tx_context::sender(ctx)
        );
    }

    #[test_only]
    /// NOTE: Even though this module is `#[test_only]`, this method is tagged
    /// with the same macro  as a trick to allow another method within this
    /// module to call `init` using OTW.
    public fun init_test_only(ctx: &mut TxContext) {
        init(COIN {}, ctx);

        // This will be created and sent to the transaction sender
        // automatically when the contract is published.
        transfer::public_transfer(
            package::test_publish(object::id_from_address(@coins), ctx),
            tx_context::sender(ctx)
        );
    }
}

#[test_only]
module coins::coin_tests {
    use sui::coin::{Self};
    use sui::package::{UpgradeCap};
    use sui::test_scenario::{Self};
    use token_bridge::create_wrapped::{Self, WrappedAssetSetup};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        register_dummy_emitter,
        return_state,
        set_up_wormhole_and_token_bridge,
        take_state,
        two_people
    };
    use token_bridge::token_registry::{Self};
    use token_bridge::vaa::{Self};
    use token_bridge::wrapped_asset::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::external_address::{Self};
    use wormhole::wormhole_scenario::{parse_and_verify_vaa};

    use token_bridge::version_control::{V__0_2_0 as V__CURRENT};

    use coins::coin::{COIN};

// +------------------------------------------------------------------------------+
// | Wormhole VAA v1         | nonce: 1                | time: 1                  |
// | guardian set #0         | #22080291               | consistency: 0           |
// |------------------------------------------------------------------------------|
// | Signature:                                                                   |
// |   #0: 80366065746148420220f25a6275097370e8db40984529a6676b7a5fc9fe...        |
// |------------------------------------------------------------------------------|
// | Emitter: 0x00000000000000000000000000000000deadbeef (Ethereum)               |
// |==============================================================================|
// | Token attestation                                                            |
// | decimals: 12                                                                 |
// | Token: 0x00000000000000000000000000000000beefface (Ethereum)                 |
// | Symbol: BEEF                                                                 |
// | Name: Beef face Token                                                        |
// +------------------------------------------------------------------------------+
    const VAA: vector<u8> =
        x"0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";

// +------------------------------------------------------------------------------+
// | Wormhole VAA v1         | nonce: 69               | time: 0                  |
// | guardian set #0         | #1                      | consistency: 15          |
// |------------------------------------------------------------------------------|
// | Signature:                                                                   |
// |   #0: b0571650590e147fce4eb60105e0463522c1244a97bd5dcb365d3e7bc7f3...        |
// |------------------------------------------------------------------------------|
// | Emitter: 0x00000000000000000000000000000000deadbeef (Ethereum)               |
// |==============================================================================|
// | Token attestation                                                            |
// | decimals: 12                                                                 |
// | Token: 0x00000000000000000000000000000000beefface (Ethereum)                 |
// | Symbol: BEEF??? and profit                                                   |
// | Name: Beef face Token??? and profit                                          |
// +------------------------------------------------------------------------------+
    const UPDATED_VAA: vector<u8> =
        x"0100000000010062f4dcd21bbbc4af8b8baaa2da3a0b168efc4c975de5b828c7a3c710b67a0a0d476d10a74aba7a7867866daf97d1372d8e6ee62ccc5ae522e3e603c67fa23787000000000000000045000200000000000000000000000000000000000000000000000000000000deadbeef00000000000000010f0200000000000000000000000000000000000000000000000000000000beefface00020c424545463f3f3f20616e642070726f666974000000000000000000000000000042656566206661636520546f6b656e3f3f3f20616e642070726f666974000000";


    #[test]
    public fun test_complete_and_update_attestation() {
        let (caller, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Ignore effects. Make sure `coin_deployer` receives
        // `WrappedAssetSetup`.
        test_scenario::next_tx(scenario, coin_deployer);

        // Publish coin.
        coins::coin::init_test_only(test_scenario::ctx(scenario));

        // Ignore effects.
        test_scenario::next_tx(scenario, coin_deployer);

        let wrapped_asset_setup =
            test_scenario::take_from_address<WrappedAssetSetup<COIN, V__CURRENT>>(
                scenario,
                coin_deployer
            );

        let token_bridge_state = take_state(scenario);

        let verified_vaa = parse_and_verify_vaa(scenario, VAA);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        let coin_meta = test_scenario::take_shared(scenario);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        create_wrapped::complete_registration(
            &mut token_bridge_state,
            &mut coin_meta,
            wrapped_asset_setup,
            test_scenario::take_from_address<UpgradeCap>(
                scenario,
                coin_deployer
            ),
            msg
        );

        // Check registry.
        {
            let verified = state::verified_asset<COIN>(&token_bridge_state);
            assert!(token_bridge::token_registry::is_wrapped<COIN>(&verified), 0);

            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset =
                token_registry::borrow_wrapped<COIN>(registry);
            assert!(wrapped_asset::total_supply(asset) == 0, 0);

            // Decimals are capped for this wrapped asset.
            assert!(coin::get_decimals(&coin_meta) == 8, 0);

            // Check metadata against asset metadata.
            let info = wrapped_asset::info(asset);
            assert!(wrapped_asset::token_chain(info) == 2, 0);
            assert!(wrapped_asset::token_address(info) == external_address::new(bytes32::from_bytes(x"00000000000000000000000000000000beefface")), 0);
            assert!(
                wrapped_asset::native_decimals(info) == 12,
                0
            );
            assert!(coin::get_symbol(&coin_meta) == std::ascii::string(b"BEEF"), 0);
            assert!(coin::get_name(&coin_meta) == std::string::utf8(b"Beef face Token"), 0);
        };

        let verified_vaa =
            parse_and_verify_vaa(scenario, UPDATED_VAA);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Now update metadata.
        create_wrapped::update_attestation<COIN>(&mut token_bridge_state, &mut coin_meta, msg);

        // Check updated name and symbol.
        assert!(
            coin::get_name(&coin_meta) == std::string::utf8(b"Beef face Token??? and profit"),
            0
        );
        assert!(
            coin::get_symbol(&coin_meta) == std::ascii::string(b"BEEF??? and profit"),
            0
        );

        // Clean up.
        return_state(token_bridge_state);
        test_scenario::return_shared(coin_meta);


        // Done.
        test_scenario::end(my_scenario);
    }
}
