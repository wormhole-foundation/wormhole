// SPDX-License-Identifier: Apache 2

/// This module implements three methods: `prepare_transfer` and
/// `transfer_tokens`, which are meant to work together.
///
/// `prepare_transfer` allows a contract to pack token transfer parameters in
/// preparation to bridge these assets to another network. Anyone can call this
/// method to create `TransferTicket`.
///
/// `transfer_tokens` unpacks the `TransferTicket` and constructs a
/// `MessageTicket`, which will be used by Wormhole's `publish_message`
/// module.
///
/// The purpose of splitting this token transferring into two steps is in case
/// Token Bridge needs to be upgraded and there is a breaking change for this
/// module, an integrator would not be left broken. It is discouraged to put
/// `transfer_tokens` in an integrator's package logic. Otherwise, this
/// integrator needs to be prepared to upgrade his contract to handle the latest
/// version of `transfer_tokens`.
///
/// Instead, an integrator is encouraged to execute a transaction block, which
/// executes `transfer_tokens` using the latest Token Bridge package ID and to
/// implement `prepare_transfer` in his contract to produce `PrepareTransfer`.
///
/// NOTE: Only assets that exist in the `TokenRegistry` can be bridged out,
/// which are native Sui assets that have been attested for via `attest_token`
/// and wrapped foreign assets that have been created using foreign asset
/// metadata via the `create_wrapped` module.
///
/// See `transfer` module for serialization and deserialization of Wormhole
/// message payload.
module token_bridge::transfer_tokens {
    use sui::balance::{Self, Balance};
    use sui::coin::{Self, Coin};
    use wormhole::bytes32::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::publish_message::{MessageTicket};

    use token_bridge::native_asset::{Self};
    use token_bridge::normalized_amount::{Self, NormalizedAmount};
    use token_bridge::state::{Self, State, LatestOnly};
    use token_bridge::token_registry::{Self, VerifiedAsset};
    use token_bridge::transfer::{Self};
    use token_bridge::wrapped_asset::{Self};

    friend token_bridge::transfer_tokens_with_payload;

    /// Relayer fee exceeds `Coin` object's value.
    const E_RELAYER_FEE_EXCEEDS_AMOUNT: u64 = 0;

    /// This type represents transfer data for a recipient on a foreign chain.
    /// The only way to destroy this type is calling `transfer_tokens`.
    ///
    /// NOTE: An integrator that expects to bridge assets between his contracts
    /// should probably use the `transfer_tokens_with_payload` module, which
    /// expects a specific redeemer to complete the transfer (transfers sent
    /// using `transfer_tokens` can be redeemed by anyone on behalf of the
    /// encoded recipient).
    struct TransferTicket<phantom CoinType> {
        asset_info: VerifiedAsset<CoinType>,
        bridged_in: Balance<CoinType>,
        norm_amount: NormalizedAmount,
        recipient_chain: u16,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u32
    }

    /// `prepare_transfer` constructs token transfer parameters. Any remaining
    /// amount (A.K.A. dust) from the funds provided will be returned along with
    /// the `TransferTicket` type. The returned coin object is the same object
    /// moved into this method.
    ///
    /// NOTE: Integrators of Token Bridge should be calling only this method
    /// from their contracts. This method is not guarded by version control
    /// (thus not requiring a reference to the Token Bridge `State` object), so
    /// it is intended to work for any package version.
    public fun prepare_transfer<CoinType>(
        asset_info: VerifiedAsset<CoinType>,
        funded: Coin<CoinType>,
        recipient_chain: u16,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u32
    ): (
        TransferTicket<CoinType>,
        Coin<CoinType>
    ) {
        let (
            bridged_in,
            norm_amount
        ) = take_truncated_amount(&asset_info, &mut funded);

        let ticket =
            TransferTicket {
                asset_info,
                bridged_in,
                norm_amount,
                relayer_fee,
                recipient_chain,
                recipient,
                nonce
            };

        // The remaining amount of funded may have dust depending on the
        // decimals of this asset.
        (ticket, funded)
    }

    /// `transfer_tokens` is the only method that can unpack the members of
    /// `TransferTicket`. This method takes the balance from this type and
    /// bridges this asset out of Sui by either joining its balance in the Token
    /// Bridge's custody for native assets or burning its balance for wrapped
    /// assets.
    ///
    /// A `relayer_fee` of some value less than or equal to the bridged balance
    /// can be specified to incentivize someone to redeem this transfer on
    /// behalf of the `recipient`.
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
    public fun transfer_tokens<CoinType>(
        token_bridge_state: &mut State,
        ticket: TransferTicket<CoinType>
    ): MessageTicket {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        let (
            nonce,
            encoded_transfer
        ) =
            bridge_in_and_serialize_transfer(
                &latest_only,
                token_bridge_state,
                ticket
            );

        // Prepare Wormhole message with encoded `Transfer`.
        state::prepare_wormhole_message(
            &latest_only,
            token_bridge_state,
            nonce,
            encoded_transfer
        )
    }

    /// Modify coin based on the decimals of a given coin type, which may
    /// leave some amount if the decimals lead to truncating the coin's balance.
    /// This method returns the extracted balance (which will be bridged out of
    /// Sui) and the normalized amount, which will be encoded in the token
    /// transfer payload.
    ///
    /// NOTE: This is a privileged method, which only this and the
    /// `transfer_tokens_with_payload` modules can use.
    public(friend) fun take_truncated_amount<CoinType>(
        asset_info: &VerifiedAsset<CoinType>,
        funded: &mut Coin<CoinType>
    ): (
        Balance<CoinType>,
        NormalizedAmount
    ) {
        // Calculate dust. If there is any, `bridged_in` will have remaining
        // value after split. `norm_amount` is copied since it is denormalized
        // at this step.
        let decimals = token_registry::coin_decimals(asset_info);
        let norm_amount =
            normalized_amount::from_raw(coin::value(funded), decimals);

        // Split the `bridged_in` coin object to return any dust remaining on
        // that object. Only bridge in the adjusted amount after de-normalizing
        // the normalized amount.
        let truncated =
            balance::split(
                coin::balance_mut(funded),
                normalized_amount::to_raw(norm_amount, decimals)
            );

        (truncated, norm_amount)
    }

    /// For a given coin type, either burn Token Bridge wrapped assets or
    /// deposit coin into Token Bridge's custody. This method returns the
    /// canonical token info (chain ID and address), which will be encoded in
    /// the token transfer.
    ///
    /// NOTE: This is a privileged method, which only this and the
    /// `transfer_tokens_with_payload` modules can use.
    public(friend) fun burn_or_deposit_funds<CoinType>(
        latest_only: &LatestOnly,
        token_bridge_state: &mut State,
        asset_info: &VerifiedAsset<CoinType>,
        bridged_in: Balance<CoinType>
    ): (
        u16,
        ExternalAddress
    ) {
        // Either burn or deposit depending on `CoinType`.
        let registry =
            state::borrow_mut_token_registry(latest_only, token_bridge_state);
        if (token_registry::is_wrapped(asset_info)) {
            wrapped_asset::burn(
                token_registry::borrow_mut_wrapped(registry),
                bridged_in
            );
        } else {
            native_asset::deposit(
                token_registry::borrow_mut_native(registry),
                bridged_in
            );
        };

        // Return canonical token info.
        (
            token_registry::token_chain(asset_info),
            token_registry::token_address(asset_info)
        )
    }

    fun bridge_in_and_serialize_transfer<CoinType>(
        latest_only: &LatestOnly,
        token_bridge_state: &mut State,
        ticket: TransferTicket<CoinType>
    ): (
        u32,
        vector<u8>
    ) {
        let TransferTicket {
            asset_info,
            bridged_in,
            norm_amount,
            recipient_chain,
            recipient,
            relayer_fee,
            nonce
        } = ticket;

        // Disallow `relayer_fee` to be greater than the `Coin` object's value.
        // Keep in mind that the relayer fee is evaluated against the truncated
        // amount.
        let amount = sui::balance::value(&bridged_in);
        assert!(relayer_fee <= amount, E_RELAYER_FEE_EXCEEDS_AMOUNT);

        // Handle funds and get canonical token info for encoded transfer.
        let (
            token_chain,
            token_address
        ) = burn_or_deposit_funds(
            latest_only,
            token_bridge_state,
            &asset_info, bridged_in
        );

        // Ensure that the recipient is a 32-byte address.
        let recipient = external_address::new(bytes32::from_bytes(recipient));

        // Finally encode `Transfer`.
        let encoded =
            transfer::serialize(
                transfer::new(
                    norm_amount,
                    token_address,
                    token_chain,
                    recipient,
                    recipient_chain,
                    normalized_amount::from_raw(
                        relayer_fee,
                        token_registry::coin_decimals(&asset_info)
                    )
                )
            );

        (nonce, encoded)
    }

    #[test_only]
    public fun bridge_in_and_serialize_transfer_test_only<CoinType>(
        token_bridge_state: &mut State,
        ticket: TransferTicket<CoinType>
    ): (
        u32,
        vector<u8>
    ) {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        bridge_in_and_serialize_transfer(
            &latest_only,
            token_bridge_state,
            ticket
        )
    }
}

#[test_only]
module token_bridge::transfer_token_tests {
    use sui::coin::{Self};
    use sui::test_scenario::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::external_address::{Self};
    use wormhole::publish_message::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::coin_wrapped_7::{Self, COIN_WRAPPED_7};
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
    use token_bridge::transfer::{Self};
    use token_bridge::transfer_tokens::{Self};
    use token_bridge::wrapped_asset::{Self};

    /// Test consts.
    const TEST_TARGET_RECIPIENT: vector<u8> = x"beef4269";
    const TEST_TARGET_CHAIN: u16 = 2;
    const TEST_NONCE: u32 = 0;
    const TEST_COIN_NATIVE_10_DECIMALS: u8 = 10;
    const TEST_COIN_WRAPPED_7_DECIMALS: u8 = 7;

    #[test]
    fun test_transfer_tokens_native_10() {
        use token_bridge::transfer_tokens::{prepare_transfer, transfer_tokens};

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

        // Define the relayer fee.
        let relayer_fee = 100000;

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be zero for COIN_NATIVE_10.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == 0, 0);
        };

        let asset_info = state::verified_asset(&token_bridge_state);
        let (
            ticket,
            dust
        ) =
            prepare_transfer(
                asset_info,
                coin::from_balance(
                    coin_10_balance,
                    test_scenario::ctx(scenario)
                ),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                relayer_fee,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // Call `transfer_tokens`.
        let prepared_msg =
            transfer_tokens(&mut token_bridge_state, ticket);

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

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_transfer_tokens_native_10_with_dust_refund() {
        use token_bridge::transfer_tokens::{prepare_transfer, transfer_tokens};

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
        let coin_10_balance =
            coin_native_10::init_register_and_mint(
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

        // Define the relayer fee.
        let relayer_fee = 100000;

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be zero for COIN_NATIVE_10.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == 0, 0);
        };

        let asset_info = state::verified_asset(&token_bridge_state);
        let (
            ticket,
            dust
        ) =
            prepare_transfer(
                asset_info,
                coin::from_balance(
                    coin_10_balance,
                    test_scenario::ctx(scenario)
                ),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                relayer_fee,
                TEST_NONCE,
            );
        assert!(coin::value(&dust) == expected_dust, 0);

        // Call `transfer_tokens`.
        let prepared_msg =
            transfer_tokens(&mut token_bridge_state, ticket);

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
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_serialize_transfer_tokens_native_10() {
        use token_bridge::transfer_tokens::{
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
        let bridged_coin_10 =
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

        // Define the relayer fee.
        let relayer_fee = 100000;

        let asset_info = state::verified_asset(&token_bridge_state);
        let expected_token_address = token_registry::token_address(&asset_info);

        let (
            ticket,
            dust
        ) =
            prepare_transfer(
                asset_info,
                bridged_coin_10,
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                relayer_fee,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // Call `transfer_tokens`.
        let (
            nonce,
            payload
        ) =
            bridge_in_and_serialize_transfer_test_only(
                &mut token_bridge_state,
                ticket
            );
        assert!(nonce == TEST_NONCE, 0);

        // Construct expected payload from scratch and confirm that the
        // `transfer_tokens` call produces the same payload.
        let expected_amount =
            normalized_amount::from_raw(
                transfer_amount,
                TEST_COIN_NATIVE_10_DECIMALS
            );
        let expected_relayer_fee =
            normalized_amount::from_raw(
                relayer_fee,
                TEST_COIN_NATIVE_10_DECIMALS
            );

        let expected_payload =
            transfer::new_test_only(
                expected_amount,
                expected_token_address,
                chain_id(),
                external_address::new(
                    bytes32::from_bytes(TEST_TARGET_RECIPIENT)
                ),
                TEST_TARGET_CHAIN,
                expected_relayer_fee
            );
        assert!(transfer::serialize_test_only(expected_payload) == payload, 0);

        // Clean up.
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_transfer_tokens_wrapped_7() {
        use token_bridge::transfer_tokens::{prepare_transfer, transfer_tokens};

        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Register and mint coins.
        let transfer_amount = 42069000;
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

        // Define the relayer fee.
        let relayer_fee = 100000;

        // Balance check the Token Bridge before executing the transfer. The
        // initial balance should be the `transfer_amount` for COIN_WRAPPED_7.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset =
                token_registry::borrow_wrapped<COIN_WRAPPED_7>(registry);
            assert!(wrapped_asset::total_supply(asset) == transfer_amount, 0);
        };

        let asset_info = state::verified_asset(&token_bridge_state);
        let (
            ticket,
            dust
        ) =
            prepare_transfer(
                asset_info,
                coin::from_balance(
                    coin_7_balance,
                    test_scenario::ctx(scenario)
                ),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                relayer_fee,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // Call `transfer_tokens`.
        let prepared_msg =
            transfer_tokens(&mut token_bridge_state, ticket);

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
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_serialize_transfer_tokens_wrapped_7() {
        use token_bridge::transfer_tokens::{
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

        // Define the relayer fee.
        let relayer_fee = 100000;

        let asset_info = state::verified_asset(&token_bridge_state);
        let expected_token_address = token_registry::token_address(&asset_info);
        let expected_token_chain = token_registry::token_chain(&asset_info);

        let (
            ticket,
            dust
        ) =
            prepare_transfer(
                asset_info,
                bridged_coin_7,
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                relayer_fee,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // Call `transfer_tokens`.
        let (
            nonce,
            payload
        ) =
            bridge_in_and_serialize_transfer_test_only(
                &mut token_bridge_state,
                ticket
            );
        assert!(nonce == TEST_NONCE, 0);

        // Construct expected payload from scratch and confirm that the
        // `transfer_tokens` call produces the same payload.
        let expected_amount =
            normalized_amount::from_raw(
                transfer_amount,
                TEST_COIN_WRAPPED_7_DECIMALS
            );
        let expected_relayer_fee =
            normalized_amount::from_raw(
                relayer_fee,
                TEST_COIN_WRAPPED_7_DECIMALS
            );

        let expected_payload =
            transfer::new_test_only(
                expected_amount,
                expected_token_address,
                expected_token_chain,
                external_address::new(
                    bytes32::from_bytes(TEST_TARGET_RECIPIENT)
                ),
                TEST_TARGET_CHAIN,
                expected_relayer_fee
            );
        assert!(transfer::serialize_test_only(expected_payload) == payload, 0);

        // Clean up.
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = token_registry::E_UNREGISTERED)]
    fun test_cannot_transfer_tokens_native_not_registered() {
        use token_bridge::transfer_tokens::{prepare_transfer, transfer_tokens};

        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Initialize COIN_NATIVE_10 (but don't register it).
        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        // NOTE: This test purposely doesn't `attest` COIN_NATIVE_10.
        let transfer_amount = 6942000;
        let test_coins =
            coin::mint_for_testing<COIN_NATIVE_10>(
                transfer_amount,
                test_scenario::ctx(scenario)
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let token_bridge_state = take_state(scenario);

        // Define the relayer fee.
        let relayer_fee = 100000;

        let asset_info = state::verified_asset(&token_bridge_state);
        let (
            ticket,
            dust
        ) =
            prepare_transfer(
                asset_info,
                test_coins,
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                relayer_fee,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // You shall not pass!
        let prepared_msg =
            transfer_tokens(&mut token_bridge_state, ticket);

        // Clean up.
        publish_message::destroy(prepared_msg);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = token_registry::E_UNREGISTERED)]
    fun test_cannot_transfer_tokens_wrapped_not_registered() {
        use token_bridge::transfer_tokens::{prepare_transfer, transfer_tokens};

        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // Initialize COIN_WRAPPED_7 (but don't register it).
        coin_native_10::init_test_only(test_scenario::ctx(scenario));

        let treasury_cap =
            coin_wrapped_7::init_and_take_treasury_cap(
                scenario,
                sender
            );
        sui::test_utils::destroy(treasury_cap);

        // NOTE: This test purposely doesn't `attest` COIN_WRAPPED_7.
        let transfer_amount = 42069;
        let test_coins =
            coin::mint_for_testing<COIN_WRAPPED_7>(
                transfer_amount,
                test_scenario::ctx(scenario)
            );

        // Ignore effects.
        test_scenario::next_tx(scenario, sender);

        // Fetch objects necessary for sending the transfer.
        let token_bridge_state = take_state(scenario);

        // Define the relayer fee.
        let relayer_fee = 1000;

        let asset_info = state::verified_asset(&token_bridge_state);
        let (
            ticket,
            dust
        ) =
            prepare_transfer(
                asset_info,
                test_coins,
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                relayer_fee,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // You shall not pass!
        let prepared_msg =
            transfer_tokens(&mut token_bridge_state, ticket);

        // Clean up.
        publish_message::destroy(prepared_msg);

        abort 42
    }

    #[test]
    #[expected_failure(
        abort_code = transfer_tokens::E_RELAYER_FEE_EXCEEDS_AMOUNT
    )]
    fun test_cannot_transfer_tokens_fee_exceeds_amount() {
        use token_bridge::transfer_tokens::{prepare_transfer, transfer_tokens};

        let sender = person();
        let my_scenario = test_scenario::begin(sender);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        register_dummy_emitter(scenario, TEST_TARGET_CHAIN);

        // NOTE: The `relayer_fee` is intentionally set to a higher number
        // than the `transfer_amount`.
        let relayer_fee = 100001;
        let transfer_amount = 100000;
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

        let asset_info = state::verified_asset(&token_bridge_state);
        let (
            ticket,
            dust
        ) =
            prepare_transfer(
                asset_info,
                coin::from_balance(
                    coin_10_balance,
                    test_scenario::ctx(scenario)
                ),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                relayer_fee,
                TEST_NONCE,
            );
        coin::destroy_zero(dust);

        // You shall not pass!
        let prepared_msg =
            transfer_tokens(&mut token_bridge_state, ticket);

        // Done.
        publish_message::destroy(prepared_msg);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_transfer_tokens_outdated_version() {
        use token_bridge::transfer_tokens::{prepare_transfer, transfer_tokens};

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

        let asset_info = state::verified_asset(&token_bridge_state);

        let relayer_fee = 0;

        let (
            ticket,
            dust
        ) =
            prepare_transfer(
                asset_info,
                coin::from_balance(
                    coin_10_balance,
                    test_scenario::ctx(scenario)
                ),
                TEST_TARGET_CHAIN,
                TEST_TARGET_RECIPIENT,
                relayer_fee,
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
            transfer_tokens(&mut token_bridge_state, ticket);

        // Clean up.
        publish_message::destroy(prepared_msg);

        abort 42
    }
}
