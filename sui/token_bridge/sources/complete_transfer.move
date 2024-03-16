// SPDX-License-Identifier: Apache 2

/// This module implements two methods: `authorize_transfer` and
/// `redeem_relayer_payout`, which are to be executed in a transaction block in
/// this order.
///
/// `authorize_transfer` allows a contract to complete a Token Bridge transfer,
/// sending assets to the encoded recipient. The coin payout incentive in
/// redeeming the transfer is packaged in a `RelayerReceipt`.
///
/// `redeem_relayer_payout` unpacks the `RelayerReceipt` to release the coin
/// containing the relayer fee amount.
///
/// The purpose of splitting this transfer redemption into two steps is in case
/// Token Bridge needs to be upgraded and there is a breaking change for this
/// module, an integrator would not be left broken. It is discouraged to put
/// `authorize_transfer` in an integrator's package logic. Otherwise, this
/// integrator needs to be prepared to upgrade his contract to handle the latest
/// version of `complete_transfer`.
///
/// Instead, an integrator is encouraged to execute a transaction block, which
/// executes `authorize_transfer` using the latest Token Bridge package ID and
/// to implement `redeem_relayer_payout` in his contract to consume this receipt.
/// This is similar to how an integrator with Wormhole is not meant to use
/// `vaa::parse_and_verify` in his contract in case the `vaa` module needs to
/// be upgraded due to a breaking change.
///
/// See `transfer` module for serialization and deserialization of Wormhole
/// message payload.
module token_bridge::complete_transfer {
    use sui::balance::{Self, Balance};
    use sui::coin::{Self, Coin};
    use sui::tx_context::{Self, TxContext};
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::native_asset::{Self};
    use token_bridge::normalized_amount::{Self, NormalizedAmount};
    use token_bridge::state::{Self, State, LatestOnly};
    use token_bridge::token_registry::{Self, VerifiedAsset};
    use token_bridge::transfer::{Self};
    use token_bridge::vaa::{Self, TokenBridgeMessage};
    use token_bridge::wrapped_asset::{Self};

    // Requires `handle_complete_transfer`.
    friend token_bridge::complete_transfer_with_payload;

    /// Transfer not intended to be received on Sui.
    const E_TARGET_NOT_SUI: u64 = 0;
    /// Input token info does not match registered info.
    const E_CANONICAL_TOKEN_INFO_MISMATCH: u64 = 1;

    /// Event reflecting when a transfer via `complete_transfer` or
    /// `complete_transfer_with_payload` is successfully executed.
    struct TransferRedeemed has drop, copy {
        emitter_chain: u16,
        emitter_address: ExternalAddress,
        sequence: u64
    }

    #[allow(lint(coin_field))]
    /// This type is only generated from `authorize_transfer` and can only be
    /// redeemed using `redeem_relayer_payout`. Integrators running relayer
    /// contracts are expected to implement `redeem_relayer_payout` within their
    /// contracts and call `authorize_transfer` in a transaction block preceding
    /// the method that consumes this receipt.
    struct RelayerReceipt<phantom CoinType> {
        /// Coin of relayer fee payout.
        payout: Coin<CoinType>
    }

    /// `authorize_transfer` deserializes a token transfer VAA payload. Once the
    /// transfer is authorized, an event (`TransferRedeemed`) is emitted to
    /// reflect which Token Bridge this transfer originated from. The
    /// `RelayerReceipt` returned wraps a `Coin` object containing a payout that
    /// incentivizes someone to execute a transaction on behalf of the encoded
    /// recipient.
    ///
    /// NOTE: This method is guarded by a minimum build version check. This
    /// method could break backward compatibility on an upgrade.
    ///
    /// It is important for integrators to refrain from calling this method
    /// within their contracts. This method is meant to be called in a
    /// transaction block, passing the `RelayerReceipt` to a method which calls
    /// `redeem_relayer_payout` within a contract. If in a circumstance where
    /// this module has a breaking change in an upgrade, `redeem_relayer_payout`
    /// will not be affected by this change.
    ///
    /// See `redeem_relayer_payout` for more details.
    public fun authorize_transfer<CoinType>(
        token_bridge_state: &mut State,
        msg: TokenBridgeMessage,
        ctx: &mut TxContext
    ): RelayerReceipt<CoinType> {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        // Emitting the transfer being redeemed (and disregard return value).
        emit_transfer_redeemed(&msg);

        // Deserialize transfer message and process.
        handle_complete_transfer<CoinType>(
            &latest_only,
            token_bridge_state,
            vaa::take_payload(msg),
            ctx
        )
    }

    /// After a transfer is authorized, a relayer contract may unpack the
    /// `RelayerReceipt` using this method. Coin representing the relaying
    /// incentive from this receipt is returned. This method is meant to be
    /// simple. It allows for a coordination with calling `authorize_upgrade`
    /// before a method that implements `redeem_relayer_payout` in a transaction
    /// block to consume this receipt.
    ///
    /// NOTE: Integrators of Token Bridge collecting relayer fee payouts from
    /// these token transfers should be calling only this method from their
    /// contracts. This method is not  guarded by version control (thus not
    /// requiring a reference to the Token Bridge `State` object), so it is
    /// intended to work for any package version.
    public fun redeem_relayer_payout<CoinType>(
        receipt: RelayerReceipt<CoinType>
    ): Coin<CoinType> {
        let RelayerReceipt { payout } = receipt;

        payout
    }

    /// This is a privileged method only used by `complete_transfer` and
    /// `complete_transfer_with_payload` modules. This method validates the
    /// encoded token info with the passed in coin type via the `TokenRegistry`.
    /// The transfer amount is denormalized and either mints balance of
    /// wrapped asset or withdraws balance from native asset custody.
    ///
    /// Depending on whether this coin is a Token Bridge wrapped asset or a
    /// natively existing asset on Sui, the coin is either minted or withdrawn
    /// from Token Bridge's custody.
    public(friend) fun verify_and_bridge_out<CoinType>(
        latest_only: &LatestOnly,
        token_bridge_state: &mut State,
        token_chain: u16,
        token_address: ExternalAddress,
        target_chain: u16,
        amount: NormalizedAmount
    ): (
        VerifiedAsset<CoinType>,
        Balance<CoinType>
    ) {
        // Verify that the intended chain ID for this transfer is for Sui.
        assert!(
            target_chain == wormhole::state::chain_id(),
            E_TARGET_NOT_SUI
        );

        let asset_info = state::verified_asset<CoinType>(token_bridge_state);
        assert!(
            (
                token_chain == token_registry::token_chain(&asset_info) &&
                token_address == token_registry::token_address(&asset_info)
            ),
            E_CANONICAL_TOKEN_INFO_MISMATCH
        );

        // De-normalize amount in preparation to take `Balance`.
        let raw_amount =
            normalized_amount::to_raw(
                amount,
                token_registry::coin_decimals(&asset_info)
            );

        // If the token is wrapped by Token Bridge, we will mint these tokens.
        // Otherwise, we will withdraw from custody.
        let bridged_out = {
            let registry =
                state::borrow_mut_token_registry(
                    latest_only,
                    token_bridge_state
                );
            if (token_registry::is_wrapped(&asset_info)) {
                wrapped_asset::mint(
                    token_registry::borrow_mut_wrapped(registry),
                    raw_amount
                )
            } else {
                native_asset::withdraw(
                    token_registry::borrow_mut_native(registry),
                    raw_amount
                )
            }
        };

        (asset_info, bridged_out)
    }

    /// This method emits source information of the token transfer. Off-chain
    /// processes may want to observe when transfers have been redeemed.
    public(friend) fun emit_transfer_redeemed(msg: &TokenBridgeMessage): u16 {
        let emitter_chain = vaa::emitter_chain(msg);

        // Emit Sui event with `TransferRedeemed`.
        sui::event::emit(
            TransferRedeemed {
                emitter_chain,
                emitter_address: vaa::emitter_address(msg),
                sequence: vaa::sequence(msg)
            }
        );

        emitter_chain
    }

    fun handle_complete_transfer<CoinType>(
        latest_only: &LatestOnly,
        token_bridge_state: &mut State,
        transfer_vaa_payload: vector<u8>,
        ctx: &mut TxContext
    ): RelayerReceipt<CoinType> {
        let (
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            relayer_fee
        ) = transfer::unpack(transfer::deserialize(transfer_vaa_payload));

        let (
            asset_info,
            bridged_out
        ) =
            verify_and_bridge_out(
                latest_only,
                token_bridge_state,
                token_chain,
                token_address,
                recipient_chain,
                amount
            );

        let recipient = external_address::to_address(recipient);

        // If the recipient did not redeem his own transfer, Token Bridge will
        // split the withdrawn coins and send a portion to the transaction
        // relayer.
        let payout = if (
            normalized_amount::value(&relayer_fee) == 0 ||
            recipient == tx_context::sender(ctx)
        ) {
            balance::zero()
        } else {
            let payout_amount =
                normalized_amount::to_raw(
                    relayer_fee,
                    token_registry::coin_decimals(&asset_info)
                );
            balance::split(&mut bridged_out, payout_amount)
        };

        // Transfer tokens to the recipient.
        sui::transfer::public_transfer(
            coin::from_balance(bridged_out, ctx),
            recipient
        );

        // Finally produce the receipt that a relayer can consume via
        // `redeem_relayer_payout`.
        RelayerReceipt {
            payout: coin::from_balance(payout, ctx)
        }
    }

    #[test_only]
    public fun burn<CoinType>(receipt: RelayerReceipt<CoinType>) {
        coin::burn_for_testing(redeem_relayer_payout(receipt));
    }
}

#[test_only]
module token_bridge::complete_transfer_tests {
    use sui::coin::{Self, Coin};
    use sui::test_scenario::{Self};
    use wormhole::state::{chain_id};
    use wormhole::wormhole_scenario::{parse_and_verify_vaa};

    use token_bridge::coin_wrapped_12::{Self, COIN_WRAPPED_12};
    use token_bridge::coin_wrapped_7::{Self, COIN_WRAPPED_7};
    use token_bridge::coin_native_10::{Self, COIN_NATIVE_10};
    use token_bridge::coin_native_4::{Self, COIN_NATIVE_4};
    use token_bridge::complete_transfer::{Self};
    use token_bridge::dummy_message::{Self};
    use token_bridge::native_asset::{Self};
    use token_bridge::state::{Self};
    use token_bridge::token_bridge_scenario::{
        set_up_wormhole_and_token_bridge,
        register_dummy_emitter,
        return_state,
        take_state,
        three_people,
        two_people
    };
    use token_bridge::token_registry::{Self};
    use token_bridge::transfer::{Self};
    use token_bridge::vaa::{Self};
    use token_bridge::wrapped_asset::{Self};

    struct OTHER_COIN_WITNESS has drop {}

    #[test]
    /// An end-to-end test for complete transfer native with VAA.
    fun test_complete_transfer_native_10_relayer_fee() {
        use token_bridge::complete_transfer::{
            authorize_transfer,
            redeem_relayer_payout
        };

        let transfer_vaa =
            dummy_message::encoded_transfer_vaa_native_with_fee();

        let (expected_recipient, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        let custody_amount = 500000;
        coin_native_10::init_register_and_deposit(
            scenario,
            coin_deployer,
            custody_amount
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let token_bridge_state = take_state(scenario);

        // These will be checked later.
        let expected_relayer_fee = 100000;
        let expected_recipient_amount = 200000;
        let expected_amount = expected_relayer_fee + expected_recipient_amount;

        // Scope to allow immutable reference to `TokenRegistry`.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == custody_amount, 0);

            // Verify transfer parameters.
            let parsed =
                transfer::deserialize_test_only(
                    wormhole::vaa::take_payload(
                        parse_and_verify_vaa(scenario, transfer_vaa)
                    )
                );

            let asset_info =
                token_registry::verified_asset<COIN_NATIVE_10>(registry);
            let expected_token_chain = token_registry::token_chain(&asset_info);
            let expected_token_address =
                token_registry::token_address(&asset_info);
            assert!(transfer::token_chain(&parsed) == expected_token_chain, 0);
            assert!(
                transfer::token_address(&parsed) == expected_token_address,
                0
            );

            let coin_meta = test_scenario::take_shared(scenario);

            let decimals = coin::get_decimals<COIN_NATIVE_10>(&coin_meta);

            test_scenario::return_shared(coin_meta);

            assert!(
                transfer::raw_amount(&parsed, decimals) == expected_amount,
                0
            );

            assert!(
                transfer::raw_relayer_fee(&parsed, decimals) == expected_relayer_fee,
                0
            );
            assert!(
                transfer::recipient_as_address(&parsed) == expected_recipient,
                0
            );
            assert!(transfer::recipient_chain(&parsed) == chain_id(), 0);

            // Clean up.
            transfer::destroy(parsed);
        };

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let receipt =
            authorize_transfer<COIN_NATIVE_10>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );
        let payout = redeem_relayer_payout(receipt);
        assert!(coin::value(&payout) == expected_relayer_fee, 0);

        // TODO: Check for one event? `TransferRedeemed`.
        let _effects = test_scenario::next_tx(scenario, tx_relayer);

        // Check recipient's `Coin`.
        let received =
            test_scenario::take_from_address<Coin<COIN_NATIVE_10>>(
                scenario,
                expected_recipient
            );
        assert!(coin::value(&received) == expected_recipient_amount, 0);

        // And check remaining amount in custody.
        let registry = state::borrow_token_registry(&token_bridge_state);
        let remaining = custody_amount - expected_amount;
        {
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == remaining, 0);
        };

        // Clean up.
        coin::burn_for_testing(payout);
        coin::burn_for_testing(received);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    /// An end-to-end test for complete transfer native with VAA.
    fun test_complete_transfer_native_4_relayer_fee() {
        use token_bridge::complete_transfer::{
            authorize_transfer,
            redeem_relayer_payout
        };

        let transfer_vaa =
            dummy_message::encoded_transfer_vaa_native_with_fee();

        let (expected_recipient, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        let custody_amount = 5000;
        coin_native_4::init_register_and_deposit(
            scenario,
            coin_deployer,
            custody_amount
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let token_bridge_state = take_state(scenario);

        // These will be checked later.
        let expected_relayer_fee = 1000;
        let expected_recipient_amount = 2000;
        let expected_amount = expected_relayer_fee + expected_recipient_amount;

        // Scope to allow immutable reference to `TokenRegistry`.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_4>(registry);
            assert!(native_asset::custody(asset) == custody_amount, 0);

            // Verify transfer parameters.
            let parsed =
                transfer::deserialize_test_only(
                    wormhole::vaa::take_payload(
                        parse_and_verify_vaa(scenario, transfer_vaa)
                    )
                );

            let asset_info =
                token_registry::verified_asset<COIN_NATIVE_4>(registry);
            let expected_token_chain = token_registry::token_chain(&asset_info);
            let expected_token_address =
                token_registry::token_address(&asset_info);
            assert!(transfer::token_chain(&parsed) == expected_token_chain, 0);
            assert!(
                transfer::token_address(&parsed) == expected_token_address,
                0
            );

            let coin_meta = test_scenario::take_shared(scenario);
            let decimals = coin::get_decimals<COIN_NATIVE_4>(&coin_meta);
            test_scenario::return_shared(coin_meta);

            assert!(
                transfer::raw_amount(&parsed, decimals) == expected_amount,
                0
            );

            assert!(
                transfer::raw_relayer_fee(&parsed, decimals) == expected_relayer_fee,
                0
            );
            assert!(
                transfer::recipient_as_address(&parsed) == expected_recipient,
                0
            );
            assert!(transfer::recipient_chain(&parsed) == chain_id(), 0);

            // Clean up.
            transfer::destroy(parsed);
        };

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let receipt =
            authorize_transfer<COIN_NATIVE_4>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );
        let payout = redeem_relayer_payout(receipt);
        assert!(coin::value(&payout) == expected_relayer_fee, 0);

        // TODO: Check for one event? `TransferRedeemed`.
        let _effects = test_scenario::next_tx(scenario, tx_relayer);

        // Check recipient's `Coin`.
        let received =
            test_scenario::take_from_address<Coin<COIN_NATIVE_4>>(
                scenario,
                expected_recipient
            );
        assert!(coin::value(&received) == expected_recipient_amount, 0);

        // And check remaining amount in custody.
        let registry = state::borrow_token_registry(&token_bridge_state);
        let remaining = custody_amount - expected_amount;
        {
            let asset = token_registry::borrow_native<COIN_NATIVE_4>(registry);
            assert!(native_asset::custody(asset) == remaining, 0);
        };

        // Clean up.
        coin::burn_for_testing(payout);
        coin::burn_for_testing(received);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    /// An end-to-end test for complete transfer wrapped with VAA.
    fun test_complete_transfer_wrapped_7_relayer_fee() {
        use token_bridge::complete_transfer::{
            authorize_transfer,
            redeem_relayer_payout
        };

        let transfer_vaa =
            dummy_message::encoded_transfer_vaa_wrapped_7_with_fee();

        let (expected_recipient, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        coin_wrapped_7::init_and_register(scenario, coin_deployer);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let token_bridge_state = take_state(scenario);

        // These will be checked later.
        let expected_relayer_fee = 1000;
        let expected_recipient_amount = 2000;
        let expected_amount = expected_relayer_fee + expected_recipient_amount;

        // Scope to allow immutable reference to `TokenRegistry`.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset =
                token_registry::borrow_wrapped<COIN_WRAPPED_7>(registry);
            assert!(wrapped_asset::total_supply(asset) == 0, 0);

            // Verify transfer parameters.
            let parsed =
                transfer::deserialize_test_only(
                    wormhole::vaa::take_payload(
                        parse_and_verify_vaa(scenario, transfer_vaa)
                    )
                );

            let asset_info =
                token_registry::verified_asset<COIN_WRAPPED_7>(registry);
            let expected_token_chain = token_registry::token_chain(&asset_info);
            let expected_token_address =
                token_registry::token_address(&asset_info);
            assert!(transfer::token_chain(&parsed) == expected_token_chain, 0);
            assert!(
                transfer::token_address(&parsed) == expected_token_address,
                0
            );

            let coin_meta = test_scenario::take_shared(scenario);
            let decimals = coin::get_decimals<COIN_WRAPPED_7>(&coin_meta);
            test_scenario::return_shared(coin_meta);

            assert!(
                transfer::raw_amount(&parsed, decimals) == expected_amount,
                0
            );

            assert!(
                transfer::raw_relayer_fee(&parsed, decimals) == expected_relayer_fee,
                0
            );
            assert!(
                transfer::recipient_as_address(&parsed) == expected_recipient,
                0
            );
            assert!(transfer::recipient_chain(&parsed) == chain_id(), 0);

            // Clean up.
            transfer::destroy(parsed);
        };

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let receipt =
            authorize_transfer<COIN_WRAPPED_7>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );
        let payout = redeem_relayer_payout(receipt);
        assert!(coin::value(&payout) == expected_relayer_fee, 0);

        // TODO: Check for one event? `TransferRedeemed`.
        let _effects = test_scenario::next_tx(scenario, tx_relayer);

        // Check recipient's `Coin`.
        let received =
            test_scenario::take_from_address<Coin<COIN_WRAPPED_7>>(
                scenario,
                expected_recipient
            );
        assert!(coin::value(&received) == expected_recipient_amount, 0);

        // And check that the amount is the total wrapped supply.
        let registry = state::borrow_token_registry(&token_bridge_state);
        {
            let asset =
                token_registry::borrow_wrapped<COIN_WRAPPED_7>(registry);
            assert!(wrapped_asset::total_supply(asset) == expected_amount, 0);
        };

        // Clean up.
        coin::burn_for_testing(payout);
        coin::burn_for_testing(received);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    /// An end-to-end test for complete transfer wrapped with VAA.
    fun test_complete_transfer_wrapped_12_relayer_fee() {
        use token_bridge::complete_transfer::{
            authorize_transfer,
            redeem_relayer_payout
        };

        let transfer_vaa =
            dummy_message::encoded_transfer_vaa_wrapped_12_with_fee();

        let (expected_recipient, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        coin_wrapped_12::init_and_register(scenario, coin_deployer);

        // Ignore effects.
        //
        // NOTE: `tx_relayer` != `expected_recipient`.
        assert!(expected_recipient != tx_relayer, 0);
        test_scenario::next_tx(scenario, tx_relayer);

        let token_bridge_state = take_state(scenario);

        // These will be checked later.
        let expected_relayer_fee = 1000;
        let expected_recipient_amount = 2000;
        let expected_amount = expected_relayer_fee + expected_recipient_amount;

        // Scope to allow immutable reference to `TokenRegistry`.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset =
                token_registry::borrow_wrapped<COIN_WRAPPED_12>(registry);
            assert!(wrapped_asset::total_supply(asset) == 0, 0);

            // Verify transfer parameters.
            let parsed =
                transfer::deserialize_test_only(
                    wormhole::vaa::take_payload(
                        parse_and_verify_vaa(scenario, transfer_vaa)
                    )
                );

            let asset_info =
                token_registry::verified_asset<COIN_WRAPPED_12>(registry);
            let expected_token_chain = token_registry::token_chain(&asset_info);
            let expected_token_address =
                token_registry::token_address(&asset_info);
            assert!(transfer::token_chain(&parsed) == expected_token_chain, 0);
            assert!(transfer::token_address(&parsed) == expected_token_address, 0);

            let coin_meta = test_scenario::take_shared(scenario);
            let decimals = coin::get_decimals<COIN_WRAPPED_12>(&coin_meta);
            test_scenario::return_shared(coin_meta);

            assert!(transfer::raw_amount(&parsed, decimals) == expected_amount, 0);

            assert!(
                transfer::raw_relayer_fee(&parsed, decimals) == expected_relayer_fee,
                0
            );
            assert!(
                transfer::recipient_as_address(&parsed) == expected_recipient,
                0
            );
            assert!(transfer::recipient_chain(&parsed) == chain_id(), 0);

            // Clean up.
            transfer::destroy(parsed);
        };

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let receipt =
            authorize_transfer<COIN_WRAPPED_12>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );
        let payout = redeem_relayer_payout(receipt);
        assert!(coin::value(&payout) == expected_relayer_fee, 0);

        // TODO: Check for one event? `TransferRedeemed`.
        let _effects = test_scenario::next_tx(scenario, tx_relayer);

        // Check recipient's `Coin`.
        let received =
            test_scenario::take_from_address<Coin<COIN_WRAPPED_12>>(
                scenario,
                expected_recipient
            );
        assert!(coin::value(&received) == expected_recipient_amount, 0);

        // And check that the amount is the total wrapped supply.
        let registry = state::borrow_token_registry(&token_bridge_state);
        {
            let asset = token_registry::borrow_wrapped<COIN_WRAPPED_12>(registry);
            assert!(wrapped_asset::total_supply(asset) == expected_amount, 0);
        };

        // Clean up.
        coin::burn_for_testing(payout);
        coin::burn_for_testing(received);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    /// An end-to-end test for complete transfer native with VAA. The encoded VAA
    /// specifies a nonzero fee, however the `recipient` should receive the full
    /// amount for self redeeming the transfer.
    fun test_complete_transfer_native_10_relayer_fee_self_redemption() {
        use token_bridge::complete_transfer::{
            authorize_transfer,
            redeem_relayer_payout
        };

        let transfer_vaa =
            dummy_message::encoded_transfer_vaa_native_with_fee();

        let (expected_recipient, _, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(expected_recipient);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        let custody_amount = 500000;
        coin_native_10::init_register_and_deposit(
            scenario,
            coin_deployer,
            custody_amount
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, expected_recipient);

        let token_bridge_state = take_state(scenario);

        // NOTE: Although there is a fee encoded in the VAA, the relayer
        // shouldn't receive this fee. The `expected_relayer_fee` should
        // go to the recipient.
        //
        // These values will be used later.
        let expected_relayer_fee = 0;
        let encoded_relayer_fee = 100000;
        let expected_recipient_amount = 300000;
        let expected_amount = expected_relayer_fee + expected_recipient_amount;

        // Scope to allow immutable reference to `TokenRegistry`.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == custody_amount, 0);

            // Verify transfer parameters.
            let parsed =
                transfer::deserialize_test_only(
                    wormhole::vaa::take_payload(
                        parse_and_verify_vaa(scenario, transfer_vaa)
                    )
                );

            let asset_info =
                token_registry::verified_asset<COIN_NATIVE_10>(registry);
            let expected_token_chain = token_registry::token_chain(&asset_info);
            let expected_token_address =
                token_registry::token_address(&asset_info);
            assert!(transfer::token_chain(&parsed) == expected_token_chain, 0);
            assert!(transfer::token_address(&parsed) == expected_token_address, 0);

            let coin_meta = test_scenario::take_shared(scenario);

            let decimals = coin::get_decimals<COIN_NATIVE_10>(&coin_meta);

            test_scenario::return_shared(coin_meta);

            assert!(transfer::raw_amount(&parsed, decimals) == expected_amount, 0);
            assert!(
                transfer::raw_relayer_fee(&parsed, decimals) == encoded_relayer_fee,
                0
            );
            assert!(
                transfer::recipient_as_address(&parsed) == expected_recipient,
                0
            );
            assert!(transfer::recipient_chain(&parsed) == chain_id(), 0);

            // Clean up.
            transfer::destroy(parsed);
        };

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects.
        test_scenario::next_tx(scenario, expected_recipient);

        let receipt =
            authorize_transfer<COIN_NATIVE_10>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );
        let payout = redeem_relayer_payout(receipt);
        assert!(coin::value(&payout) == expected_relayer_fee, 0);

        // TODO: Check for one event? `TransferRedeemed`.
        let _effects = test_scenario::next_tx(scenario, expected_recipient);

        // Check recipient's `Coin`.
        let received =
            test_scenario::take_from_address<Coin<COIN_NATIVE_10>>(
                scenario,
                expected_recipient
            );
        assert!(coin::value(&received) == expected_recipient_amount, 0);

        // And check remaining amount in custody.
        let registry = state::borrow_token_registry(&token_bridge_state);
        let remaining = custody_amount - expected_amount;
        {
            let asset = token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(native_asset::custody(asset) == remaining, 0);
        };

        // Clean up.
        coin::burn_for_testing(payout);
        coin::burn_for_testing(received);
        return_state(token_bridge_state);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = complete_transfer::E_CANONICAL_TOKEN_INFO_MISMATCH
    )]
    /// This test verifies that `authorize_transfer` reverts when called with
    /// a native COIN_TYPE that's not encoded in the VAA.
    fun test_cannot_authorize_transfer_native_invalid_coin_type() {
        use token_bridge::complete_transfer::{authorize_transfer};

        let transfer_vaa =
            dummy_message::encoded_transfer_vaa_native_with_fee();

        let (_, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        let custody_amount_coin_10 = 500000;
        coin_native_10::init_register_and_deposit(
            scenario,
            coin_deployer,
            custody_amount_coin_10
        );

        // Register a second native asset.
        let custody_amount_coin_4 = 69420;
        coin_native_4::init_register_and_deposit(
            scenario,
            coin_deployer,
            custody_amount_coin_4
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let token_bridge_state = take_state(scenario);

        // Scope to allow immutable reference to `TokenRegistry`. This verifies
        // that both coin types have been registered.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);

            // COIN_10.
            let coin_10 =
                token_registry::borrow_native<COIN_NATIVE_10>(registry);
            assert!(
                native_asset::custody(coin_10) == custody_amount_coin_10,
                0
            );

            // COIN_4.
            let coin_4 = token_registry::borrow_native<COIN_NATIVE_4>(registry);
            assert!(native_asset::custody(coin_4) == custody_amount_coin_4, 0);
        };

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        // NOTE: this call should revert since the transfer VAA is for
        // a coin of type COIN_NATIVE_10. However, the `complete_transfer`
        // method is called using the COIN_NATIVE_4 type.
        let receipt =
            authorize_transfer<COIN_NATIVE_4>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        complete_transfer::burn(receipt);

        abort 42
    }

    #[test]
    #[expected_failure(
        abort_code = complete_transfer::E_CANONICAL_TOKEN_INFO_MISMATCH
    )]
    /// This test verifies that `authorize_transfer` reverts when called with
    /// a wrapped COIN_TYPE that's not encoded in the VAA.
    fun test_cannot_authorize_transfer_wrapped_invalid_coin_type() {
        use token_bridge::complete_transfer::{authorize_transfer};

        let transfer_vaa = dummy_message::encoded_transfer_vaa_wrapped_12_with_fee();

        let (expected_recipient, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        // Register both wrapped coin types (12 and 7).
        coin_wrapped_12::init_and_register(scenario, coin_deployer);
        coin_wrapped_7::init_and_register(scenario, coin_deployer);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        // NOTE: `tx_relayer` != `expected_recipient`.
        assert!(expected_recipient != tx_relayer, 0);

        let token_bridge_state = take_state(scenario);

        // Scope to allow immutable reference to `TokenRegistry`. This verifies
        // that both coin types have been registered.
        {
            let registry = state::borrow_token_registry(&token_bridge_state);

            let coin_12 =
                token_registry::borrow_wrapped<COIN_WRAPPED_12>(registry);
            assert!(wrapped_asset::total_supply(coin_12) == 0, 0);

            let coin_7 =
                token_registry::borrow_wrapped<COIN_WRAPPED_7>(registry);
            assert!(wrapped_asset::total_supply(coin_7) == 0, 0);
        };

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        // NOTE: this call should revert since the transfer VAA is for
        // a coin of type COIN_WRAPPED_12. However, the `authorize_transfer`
        // method is called using the COIN_WRAPPED_7 type.
        let receipt =
            authorize_transfer<COIN_WRAPPED_7>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        complete_transfer::burn(receipt);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = complete_transfer::E_TARGET_NOT_SUI)]
    /// This test verifies that `authorize_transfer` reverts when a transfer is
    /// sent to the wrong target blockchain (chain ID != 21).
    fun test_cannot_authorize_transfer_wrapped_12_invalid_target_chain() {
        use token_bridge::complete_transfer::{authorize_transfer};

        let transfer_vaa =
            dummy_message::encoded_transfer_vaa_wrapped_12_invalid_target_chain();

        let (expected_recipient, tx_relayer, coin_deployer) = three_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        coin_wrapped_12::init_and_register(scenario, coin_deployer);

        // Ignore effects.
        //
        // NOTE: `tx_relayer` != `expected_recipient`.
        assert!(expected_recipient != tx_relayer, 0);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let token_bridge_state = take_state(scenario);

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        // NOTE: this call should revert since the target chain encoded is
        // chain 69 instead of chain 21 (Sui).
        let receipt =
            authorize_transfer<COIN_WRAPPED_12>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        complete_transfer::burn(receipt);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_complete_transfer_outdated_version() {
        use token_bridge::complete_transfer::{authorize_transfer};

        let transfer_vaa =
            dummy_message::encoded_transfer_vaa_native_with_fee();

        let (tx_relayer, coin_deployer) = two_people();
        let my_scenario = test_scenario::begin(tx_relayer);
        let scenario = &mut my_scenario;

        // Set up contracts.
        let wormhole_fee = 350;
        set_up_wormhole_and_token_bridge(scenario, wormhole_fee);

        // Register foreign emitter on chain ID == 2.
        let expected_source_chain = 2;
        register_dummy_emitter(scenario, expected_source_chain);

        let custody_amount = 500000;
        coin_native_10::init_register_and_deposit(
            scenario,
            coin_deployer,
            custody_amount
        );

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

        let token_bridge_state = take_state(scenario);

        let verified_vaa = parse_and_verify_vaa(scenario, transfer_vaa);
        let msg = vaa::verify_only_once(&mut token_bridge_state, verified_vaa);

        // Ignore effects.
        test_scenario::next_tx(scenario, tx_relayer);

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
        let receipt =
            authorize_transfer<COIN_NATIVE_10>(
                &mut token_bridge_state,
                msg,
                test_scenario::ctx(scenario)
            );

        // Clean up.
        complete_transfer::burn(receipt);

        abort 42
    }
}
