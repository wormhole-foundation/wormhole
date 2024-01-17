// SPDX-License-Identifier: Apache 2

/// This module implements three methods: `prepare_transfer` and
/// `transfer_tokens_with_payload`, which are meant to work together.
///
/// `prepare_transfer` allows a contract to pack token transfer parameters with
/// an arbitrary payload in preparation to bridge these assets to another
/// network. Only an `EmitterCap` has the capability to create
/// `TransferTicket`. The `EmitterCap` object ID is encoded as the
/// sender.
///
/// `transfer_tokens_with_payload` unpacks the `TransferTicket` and
/// constructs a `MessageTicket`, which will be used by Wormhole's
/// `publish_message` module.
///
/// The purpose of splitting this token transferring into two steps is in case
/// Token Bridge needs to be upgraded and there is a breaking change for this
/// module, an integrator would not be left broken. It is discouraged to put
/// `transfer_tokens_with_payload` in an integrator's package logic. Otherwise,
/// this integrator needs to be prepared to upgrade his contract to handle the
/// latest version of `transfer_tokens_with_payload`.
///
/// Instead, an integrator is encouraged to execute a transaction block, which
/// executes `transfer_tokens_with_payload` using the latest Token Bridge
/// package ID and to implement `prepare_transfer` in his contract to produce
/// `PrepareTransferWithPayload`.
///
/// NOTE: Only assets that exist in the `TokenRegistry` can be bridged out,
/// which are native Sui assets that have been attested for via `attest_token`
/// and wrapped foreign assets that have been created using foreign asset
/// metadata via the `create_wrapped` module.
///
/// See `transfer_with_payload` module for serialization and deserialization of
/// Wormhole message payload.
module token_bridge::transfer_tokens_with_payload {
    use sui::balance::{Balance};
    use sui::coin::{Coin};
    use sui::object::{Self, ID};
    use wormhole::bytes32::{Self};
    use wormhole::emitter::{EmitterCap};
    use wormhole::external_address::{Self};
    use wormhole::publish_message::{MessageTicket};

    use token_bridge::normalized_amount::{NormalizedAmount};
    use token_bridge::state::{Self, State, LatestOnly};
    use token_bridge::token_registry::{VerifiedAsset};
    use token_bridge::transfer_with_payload::{Self};

    /// This type represents transfer data for a specific redeemer contract on a
    /// foreign chain. The only way to destroy this type is calling
    /// `transfer_tokens_with_payload`. Only the owner of an `EmitterCap` has
    /// the capability of generating `TransferTicket`. This emitter
    /// cap will usually live in an integrator's contract storage object.
    struct TransferTicket<phantom CoinType> {
        asset_info: VerifiedAsset<CoinType>,
        bridged_in: Balance<CoinType>,
        norm_amount: NormalizedAmount,
        sender: ID,
        redeemer_chain: u16,
        redeemer: vector<u8>,
        payload: vector<u8>,
        nonce: u32
    }

    /// `prepare_transfer` constructs token transfer parameters. Any remaining
    /// amount (A.K.A. dust) from the funds provided will be returned along with
    /// the `TransferTicket` type. The returned coin object is the
    /// same object moved into this method.
    ///
    /// NOTE: Integrators of Token Bridge should be calling only this method
    /// from their contracts. This method is not guarded by version control
    /// (thus not requiring a reference to the Token Bridge `State` object), so
    /// it is intended to work for any package version.
    public fun prepare_transfer<CoinType>(
        emitter_cap: &EmitterCap,
        asset_info: VerifiedAsset<CoinType>,
        funded: Coin<CoinType>,
        redeemer_chain: u16,
        redeemer: vector<u8>,
        payload: vector<u8>,
        nonce: u32
    ): (
        TransferTicket<CoinType>,
        Coin<CoinType>
    ) {
        use token_bridge::transfer_tokens::{take_truncated_amount};

        let (
            bridged_in,
            norm_amount
        ) = take_truncated_amount(&asset_info, &mut funded);

        let prepared_transfer =
            TransferTicket {
                asset_info,
                bridged_in,
                norm_amount,
                sender: object::id(emitter_cap),
                redeemer_chain,
                redeemer,
                payload,
                nonce
            };

        // The remaining amount of funded may have dust depending on the
        // decimals of this asset.
        (prepared_transfer, funded)
    }

    /// `transfer_tokens_with_payload` is the only method that can unpack the
    /// members of `TransferTicket`. This method takes the balance
    /// from this type and bridges this asset out of Sui by either joining its
    /// balance in the Token Bridge's custody for native assets or burning its
    /// balance for wrapped assets.
    ///
    /// The unpacked sender ID comes from an `EmitterCap`. It is encoded as the
    /// sender of these assets. And associated with this transfer is an
    /// arbitrary payload, which can be consumed by the specified redeemer and
    /// used as instructions for a contract composing with Token Bridge.
    ///
    /// This method returns the prepared Wormhole message (which should be
    /// consumed by calling `publish_message` in a transaction block).
    ///
    /// NOTE: This method is guarded by a minimum build version check. This
    /// method could break backward compatibility on an upgrade.
    ///
    /// It is important for integrators to refrain from calling this method
    /// within their contracts. This method is meant to be called in a
    /// transaction block after receiving a `TransferTicket` from calling
    /// `prepare_transfer` within a contract. If in a circumstance where this
    /// module has a breaking change in an upgrade, `prepare_transfer` will not
    /// be affected by this change.
    public fun transfer_tokens_with_payload<CoinType>(
        token_bridge_state: &mut State,
        prepared_transfer: TransferTicket<CoinType>
    ): MessageTicket {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        // Encode Wormhole message payload.
        let (
            nonce,
            encoded_transfer_with_payload
         ) =
            bridge_in_and_serialize_transfer(
                &latest_only,
                token_bridge_state,
                prepared_transfer
            );

        // Prepare Wormhole message with encoded `TransferWithPayload`.
        state::prepare_wormhole_message(
            &latest_only,
            token_bridge_state,
            nonce,
            encoded_transfer_with_payload
        )
    }

    fun bridge_in_and_serialize_transfer<CoinType>(
        latest_only: &LatestOnly,
        token_bridge_state: &mut State,
        prepared_transfer: TransferTicket<CoinType>
    ): (
        u32,
        vector<u8>
    ) {
        use token_bridge::transfer_tokens::{burn_or_deposit_funds};

        let TransferTicket {
            asset_info,
            bridged_in,
            norm_amount,
            sender,
            redeemer_chain,
            redeemer,
            payload,
            nonce
        } = prepared_transfer;

        let (
            token_chain,
            token_address
        ) =
            burn_or_deposit_funds(
                latest_only,
                token_bridge_state,
                &asset_info,
                bridged_in
            );

        let redeemer = external_address::new(bytes32::from_bytes(redeemer));

        let encoded =
            transfer_with_payload::serialize(
                transfer_with_payload::new(
                    sender,
                    norm_amount,
                    token_address,
                    token_chain,
                    redeemer,
                    redeemer_chain,
                    payload
                )
            );

        (nonce, encoded)
    }

    #[test_only]
    public fun bridge_in_and_serialize_transfer_test_only<CoinType>(
        token_bridge_state: &mut State,
        prepared_transfer: TransferTicket<CoinType>
    ): (
        u32,
        vector<u8>
    ) {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        bridge_in_and_serialize_transfer(
            &latest_only,
            token_bridge_state,
            prepared_transfer
        )
    }
}

#[test_only]
module token_bridge::transfer_tokens_with_payload_tests {
    use sui::coin::{Self};
    use sui::object::{Self};
    use sui::test_scenario::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::emitter::{Self};
    use wormhole::external_address::{Self};
    use wormhole::publish_message::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::coin_wrapped_7::{Self, COIN_WRAPPED_7};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::native_asset::{Self};
    use token_bridge::normalized_amount::{Self};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        set_up_wormhole_and_token_bridge,
        register_dummy_emitter,
        return_state,
        take_state,
        person
    };
    use token_bridge::token_registry::{Self};
    use token_bridge::transfer_with_payload::{Self};
    use token_bridge::wrapped_asset::{Self};

    /// Test consts.
    const TEST_TARGET_RECIPIENT: vector<u8> = x"beef4269";
    const TEST_TARGET_CHAIN: u16 = 2;
    const TEST_NONCE: u32 = 0;
    const TEST_COIN_NATIVE_10_DECIMALS: u8 = 10;
    const TEST_COIN_WRAPPED_7_DECIMALS: u8 = 7;
    const TEST_MESSAGE_PAYLOAD: vector<u8> = x"deadbeefdeadbeef";

    #[test]
    fun test_transfer_tokens_with_payload_native_10() {
        use token_bridge::transfer_tokens_with_payload::{
            prepare_transfer,
            transfer_tokens_with_payload
        };

        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 6942000;
        let coin_10_balance =
            coin_native_10::init_register_and_mint(
                scenario,
                sender,
                transfer_amount
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let token_bridge_state = take_state(scenario);

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be zero for COIN_NATIVE_10.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == 0, 0);
        };

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        let asset_info = state::verified_asset(&token_bridge_state);
        let (
            prepared_transfer,
            dust
        ) =
            prepare_transfer(
                &emitter_cap,
                asset_info,
                coin::from_balance(
                    coin_10_balance,
                    test_scenario::ctx(scenario)
                ),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // Call `transfer_tokens_with_payload`.
        let prepared_msg =
            transfer_tokens_with_payload(
                &mut token_bridge_state,
                prepared_transfer
            );

        // Balance check the Token Bridge after executing the transfer. The
        // balance should now reflect the `transfer_amount` defined in this
        // test.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == transfer_amount, 0);
        };

        // Clean up.
        publish_message::destroy(prepared_msg);
        return_state(token_bridge_state);
        emitter::destroy_test_only(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_transfer_tokens_native_10_with_dust_refund() {
        use token_bridge::transfer_tokens_with_payload::{
            prepare_transfer,
            transfer_tokens_with_payload
        };

        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 1000069;
        let coin_10_balance = coin_native_10::init_register_and_mint(
            scenario,
            sender,
            transfer_amount
        );

        // This value will be used later. The contract should return dust
        // to the caller since COIN_NATIVE_10 has 10 decimals.
        let expected_dust = 69;

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let token_bridge_state = take_state(scenario);

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be zero for COIN_NATIVE_10.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == 0, 0);
        };

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        let asset_info = state::verified_asset(&token_bridge_state);
        let (
            prepared_transfer,
            dust
        ) =
            prepare_transfer(
                &emitter_cap,
                asset_info,
                coin::from_balance(
                    coin_10_balance,
                    test_scenario::ctx(scenario)
                ),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD,
                TEST_NONCE,
            );
        assert!(coin::value(&dust) == expected_dust, 0);

        // Call `transfer_tokens`.
        let prepared_msg =
            transfer_tokens_with_payload(
                &mut token_bridge_state,
                prepared_transfer
            );

        // Balance check the Token Bridge after executing the transfer. The
        // balance should now reflect the `transfer_amount` less `expected_dust`
        // defined in this test.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(
                native_asset::custody(asset) == transfer_amount - expected_dust,
                0
            );
        };

        // Clean up.
        publish_message::destroy(prepared_msg);
        coin::burn_for_testing(dust);
        emitter::destroy_test_only(emitter_cap);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_serialize_transfer_tokens_native_10() {
        use token_bridge::transfer_tokens_with_payload::{
            bridge_in_and_serialize_transfer_test_only,
            prepare_transfer
        };

        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 6942000;
        let bridge_coin_10 =
            coin::from_balance(
                coin_native_10::init_register_and_mint(
                    scenario,
                    sender,
                    transfer_amount
                ),
                test_scenario::ctx(scenario)
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let token_bridge_state = take_state(scenario);

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        let asset_info = state::verified_asset(&token_bridge_state);
        let expected_token_address = token_registry::token_address(&asset_info);

        let (
            prepared_transfer,
            dust
        ) =
            prepare_transfer(
                &emitter_cap,
                asset_info,
                bridge_coin_10,
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // Serialize the payload.
        let (
            nonce,
            payload
         ) =
            bridge_in_and_serialize_transfer_test_only(
                &mut token_bridge_state,
                prepared_transfer
            );
        assert!(nonce == TEST_NONCE, 0);

        // Construct expected payload from scratch and confirm that the
        // `transfer_tokens` call produces the same payload.
        let expected_amount = normalized_amount::from_raw(
            transfer_amount,
            TEST_COIN_NATIVE_10_DECIMALS
        );

        let expected_payload =
            transfer_with_payload::new_test_only(
                object::id(&emitter_cap),
                expected_amount,
                expected_token_address,
                chain_id(),
                external_address::new(bytes32::from_bytes(TEST_TARGET_RECIPIENT)),
                TEST_TARGET_CHAIN,
                TEST_MESSAGE_PAYLOAD
            );
        assert!(
            transfer_with_payload::serialize(expected_payload) == payload,
            0
        );

        // Clean up.
        return_state(token_bridge_state);
        emitter::destroy_test_only(emitter_cap);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_transfer_tokens_with_payload_wrapped_7() {
        use token_bridge::transfer_tokens_with_payload::{
            prepare_transfer,
            transfer_tokens_with_payload
        };

        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 6942000;
        let coin_7_balance =
            coin_wrapped_7::init_register_and_mint(
                scenario,
                sender,
                transfer_amount
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let token_bridge_state = take_state(scenario);

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be the `transfer_amount` for COIN_WRAPPED_7.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_wrapped<COIN_WRAPPED_7>(registry);
            assert!(wrapped_asset::total_supply(asset) == transfer_amount, 0);
        };

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        let asset_info = state::verified_asset(&token_bridge_state);
        let (
            prepared_transfer,
            dust
        ) =
            prepare_transfer(
                &emitter_cap,
                asset_info,
                coin::from_balance(
                    coin_7_balance,
                    test_scenario::ctx(scenario)
                ),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // Call `transfer_tokens_with_payload`.
        let prepared_msg =
            transfer_tokens_with_payload(
                &mut token_bridge_state,
                prepared_transfer
            );

        // Balance check the Token Bridge after executing the transfer. The
        // balance should be zero, since tokens are burned when an outbound
        // wrapped token transfer occurs.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_wrapped<COIN_WRAPPED_7>(registry);
            assert!(wrapped_asset::total_supply(asset) == 0, 0);
        };

        // Clean up.
        publish_message::destroy(prepared_msg);
        emitter::destroy_test_only(emitter_cap);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_serialize_transfer_tokens_wrapped_7() {
        use token_bridge::transfer_tokens_with_payload::{
            bridge_in_and_serialize_transfer_test_only,
            prepare_transfer
        };

        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 6942000;
        let bridged_coin_7 =
            coin::from_balance(
                coin_wrapped_7::init_register_and_mint(
                    scenario,
                    sender,
                    transfer_amount
                ),
                test_scenario::ctx(scenario)
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let token_bridge_state = take_state(scenario);

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        let asset_info = state::verified_asset(&token_bridge_state);
        let expected_token_address = token_registry::token_address(&asset_info);
        let expected_token_chain = token_registry::token_chain(&asset_info);

        let (
            prepared_transfer,
            dust
        ) =
            prepare_transfer(
                &emitter_cap,
                asset_info,
                bridged_coin_7,
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // Serialize the payload.
        let (
            nonce,
            payload
         ) =
            bridge_in_and_serialize_transfer_test_only(
                &mut token_bridge_state,
                prepared_transfer
            );
        assert!(nonce == TEST_NONCE, 0);

        // Construct expected payload from scratch and confirm that the
        // `transfer_tokens` call produces the same payload.
        let expected_amount = normalized_amount::from_raw(
            transfer_amount,
            TEST_COIN_WRAPPED_7_DECIMALS
        );

        let expected_payload =
            transfer_with_payload::new_test_only(
                object::id(&emitter_cap),
                expected_amount,
                expected_token_address,
                expected_token_chain,
                 external_address::new(bytes32::from_bytes(TEST_TARGET_RECIPIENT)),
                TEST_TARGET_CHAIN,
                TEST_MESSAGE_PAYLOAD
            );
        assert!(
            transfer_with_payload::serialize(expected_payload) == payload,
            0
        );

        // Clean up.
        emitter::destroy_test_only(emitter_cap);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_transfer_tokens_with_payload_outdated_version() {
        use token_bridge::transfer_tokens_with_payload::{
            prepare_transfer,
            transfer_tokens_with_payload
        };

        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 6942000;
        let coin_10_balance =
            coin_native_10::init_register_and_mint(
                scenario,
                sender,
                transfer_amount
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let token_bridge_state = take_state(scenario);

        // Register and obtain a new wormhole emitter cap.
        let emitter_cap = emitter::dummy();

        let asset_info = state::verified_asset(&token_bridge_state);
        let (
            prepared_transfer,
            dust
        ) =
            prepare_transfer(
                &emitter_cap,
                asset_info,
                coin::from_balance(
                    coin_10_balance,
                    test_scenario::ctx(scenario)
                ),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                TEST_MESSAGE_PAYLOAD,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut token_bridge_state);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        state::migrate_version_test_only(
            &mut token_bridge_state,
            token_bridge::version_control::previous_version_test_only(),
            token_bridge::version_control::next_version()
        );

        // You shall not pass!
        let prepared_msg =
            transfer_tokens_with_payload(
                &mut token_bridge_state,
                prepared_transfer
            );

        // Clean up.
        publish_message::destroy(prepared_msg);

        abort 42
    }
}
