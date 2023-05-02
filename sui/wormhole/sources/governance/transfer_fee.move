// SPDX-License-Identifier: Apache 2

/// This module implements handling a governance VAA to enact transferring some
/// amount of collected fees to an intended recipient.
module wormhole::transfer_fee {
    use sui::coin::{Self};
    use sui::transfer::{Self};
    use sui::tx_context::{TxContext};

    use wormhole::bytes32::{Self};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self};
    use wormhole::governance_message::{Self, DecreeTicket, DecreeReceipt};
    use wormhole::state::{Self, State, LatestOnly};

    /// Specific governance payload ID (action) for setting Wormhole fee.
    const ACTION_TRANSFER_FEE: u8 = 4;

    struct GovernanceWitness has drop {}

    struct TransferFee {
        amount: u64,
        recipient: address
    }

    public fun authorize_governance(
        wormhole_state: &State
    ): DecreeTicket<GovernanceWitness> {
        governance_message::authorize_verify_local(
            GovernanceWitness {},
            state::governance_chain(wormhole_state),
            state::governance_contract(wormhole_state),
            state::governance_module(),
            ACTION_TRANSFER_FEE
        )
    }

    /// Redeem governance VAA to transfer collected Wormhole fees to the
    /// recipient encoded in its Wormhole governance message. This governance
    /// message is only relevant for Sui because fee administration is only
    /// relevant to one particular network (in this case Sui).
    ///
    /// NOTE: This method is guarded by a minimum build version check. This
    /// method could break backward compatibility on an upgrade.
    public fun transfer_fee(
        wormhole_state: &mut State,
        receipt: DecreeReceipt<GovernanceWitness>,
        ctx: &mut TxContext
    ): u64 {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(wormhole_state);

        let payload =
            governance_message::take_payload(
                state::borrow_mut_consumed_vaas(&latest_only, wormhole_state),
                receipt
            );

        // Proceed with setting the new message fee.
        handle_transfer_fee(&latest_only, wormhole_state, payload, ctx)
    }

    fun handle_transfer_fee(
        latest_only: &LatestOnly,
        wormhole_state: &mut State,
        governance_payload: vector<u8>,
        ctx: &mut TxContext
    ): u64 {
        // Deserialize the payload as amount to withdraw and to whom SUI should
        // be sent.
        let TransferFee { amount, recipient } = deserialize(governance_payload);

        transfer::public_transfer(
            coin::from_balance(
                state::withdraw_fee(latest_only, wormhole_state, amount),
                ctx
            ),
            recipient
        );

        amount
    }

    fun deserialize(payload: vector<u8>): TransferFee {
        let cur = cursor::new(payload);

        // This amount cannot be greater than max u64.
        let amount = bytes32::to_u64_be(bytes32::take_bytes(&mut cur));

        // Recipient must be non-zero address.
        let recipient = external_address::take_nonzero(&mut cur);

        cursor::destroy_empty(cur);

        TransferFee {
            amount: (amount as u64),
            recipient: external_address::to_address(recipient)
        }
    }

    #[test_only]
    public fun action(): u8 {
        ACTION_TRANSFER_FEE
    }
}

#[test_only]
module wormhole::transfer_fee_tests {
    use sui::balance::{Self};
    use sui::coin::{Self, Coin};
    use sui::sui::{SUI};
    use sui::test_scenario::{Self};

    use wormhole::bytes::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self};
    use wormhole::governance_message::{Self};
    use wormhole::state::{Self};
    use wormhole::transfer_fee::{Self};
    use wormhole::vaa::{Self};
    use wormhole::version_control::{Self};
    use wormhole::wormhole_scenario::{
        person,
        return_clock,
        return_state,
        set_up_wormhole,
        take_clock,
        take_state,
        two_people,
        upgrade_wormhole
    };

    const VAA_TRANSFER_FEE_1: vector<u8> =
        x"01000000000100a96aee105d7683266d98c9b274eddb20391378adddcefbc7a5266b4be78bc6eb582797741b65617d796c6c613ae7a4dad52a8b4aa4659842dcc4c9b3891549820100bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f726504001500000000000000000000000000000000000000000000000000000000000004b0000000000000000000000000000000000000000000000000000000000000b0b2";
    const VAA_TRANSFER_FEE_OVERFLOW: vector<u8> =
        x"01000000000100529b407a673f8917ccb9bb6f8d46d0f729c1ff845b0068ef5e0a3de464670b2e379a8994b15362785e52d73e01c880dbcdf432ef3702782d17d352fb07ed86830100bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f72650400150000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000b0b2";
    const VAA_TRANSFER_FEE_ZERO_ADDRESS: vector<u8> =
        x"0100000000010032b2ab65a690ae4af8c85903d7b22239fc272183eefdd5a4fa784664f82aa64b381380cc03859156e88623949ce4da4435199aaac1cb09e52a09d6915725a5e70100bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f726504001500000000000000000000000000000000000000000000000000000000000004b00000000000000000000000000000000000000000000000000000000000000000";

    #[test]
    fun test_transfer_fee() {
        // Testing this method.
        use wormhole::transfer_fee::{transfer_fee};

        // Set up.
        let (caller, recipient) = two_people();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `transfer_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Double-check current fee (from setup).
        assert!(state::message_fee(&worm_state) == wormhole_fee, 0);

        // Deposit fee several times.
        let (i, n) = (0, 8);
        while (i < n) {
            state::deposit_fee_test_only(
                &mut worm_state,
                balance::create_for_testing(wormhole_fee)
            );
            i = i + 1;
        };

        // Double-check balance.
        let total_deposited = n * wormhole_fee;
        assert!(state::fees_collected(&worm_state) == total_deposited, 0);

        let verified_vaa =
            vaa::parse_and_verify(&worm_state, VAA_TRANSFER_FEE_1, &the_clock);
        let ticket = transfer_fee::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        let withdrawn =
            transfer_fee(
                &mut worm_state,
                receipt,
                test_scenario::ctx(scenario)
            );
        assert!(withdrawn == 1200, 0);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        // Verify that the recipient received the withdrawal.
        let withdrawn_coin =
            test_scenario::take_from_address<Coin<SUI>>(scenario, recipient);
        assert!(coin::value(&withdrawn_coin) == withdrawn, 0);

        // And there is still a balance on Wormhole's fee collector.
        let remaining = total_deposited - withdrawn;
        assert!(state::fees_collected(&worm_state) == remaining, 0);

        // Clean up.
        coin::burn_for_testing(withdrawn_coin);
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_transfer_fee_after_upgrade() {
        // Testing this method.
        use wormhole::transfer_fee::{transfer_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Upgrade.
        upgrade_wormhole(scenario);

        // Prepare test to execute `transfer_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Double-check current fee (from setup).
        assert!(state::message_fee(&worm_state) == wormhole_fee, 0);

        // Deposit fee several times.
        let (i, n) = (0, 8);
        while (i < n) {
            state::deposit_fee_test_only(
                &mut worm_state,
                balance::create_for_testing(wormhole_fee)
            );
            i = i + 1;
        };

        // Double-check balance.
        let total_deposited = n * wormhole_fee;
        assert!(state::fees_collected(&worm_state) == total_deposited, 0);

        let verified_vaa =
            vaa::parse_and_verify(&worm_state, VAA_TRANSFER_FEE_1, &the_clock);
        let ticket = transfer_fee::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        let withdrawn =
            transfer_fee(
                &mut worm_state,
                receipt,
                test_scenario::ctx(scenario)
            );
        assert!(withdrawn == 1200, 0);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wormhole::set::E_KEY_ALREADY_EXISTS)]
    fun test_cannot_transfer_fee_with_same_vaa() {
        // Testing this method.
        use wormhole::transfer_fee::{transfer_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `transfer_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Double-check current fee (from setup).
        assert!(state::message_fee(&worm_state) == wormhole_fee, 0);

        // Deposit fee several times.
        let (i, n) = (0, 8);
        while (i < n) {
            state::deposit_fee_test_only(
                &mut worm_state,
                balance::create_for_testing(wormhole_fee)
            );
            i = i + 1;
        };

        // Transfer once.
        let verified_vaa =
            vaa::parse_and_verify(&worm_state, VAA_TRANSFER_FEE_1, &the_clock);
        let ticket = transfer_fee::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        transfer_fee(&mut worm_state, receipt, test_scenario::ctx(scenario));

        let verified_vaa =
            vaa::parse_and_verify(&worm_state, VAA_TRANSFER_FEE_1, &the_clock);
        let ticket = transfer_fee::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        // You shall not pass!
        transfer_fee(&mut worm_state, receipt, test_scenario::ctx(scenario));

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = sui::balance::ENotEnough)]
    fun test_cannot_transfer_fee_insufficient_balance() {
        // Testing this method.
        use wormhole::transfer_fee::{transfer_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `transfer_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Show balance is zero.
        assert!(state::fees_collected(&worm_state) == 0, 0);

        // Show that the encoded fee is greater than zero.
        let verified_vaa =
            vaa::parse_and_verify(&worm_state, VAA_TRANSFER_FEE_1, &the_clock);
        let payload =
            governance_message::take_decree(vaa::payload(&verified_vaa));
        let cur = cursor::new(payload);

        let amount = bytes::take_u256_be(&mut cur);
        assert!(amount > 0, 0);
        cursor::take_rest(cur);

        let ticket = transfer_fee::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        // You shall not pass!
        transfer_fee(&mut worm_state, receipt, test_scenario::ctx(scenario));

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = external_address::E_ZERO_ADDRESS)]
    fun test_cannot_transfer_fee_recipient_zero_address() {
        // Testing this method.
        use wormhole::transfer_fee::{transfer_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `transfer_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Show balance is zero.
        assert!(state::fees_collected(&worm_state) == 0, 0);

        // Show that the encoded fee is greater than zero.
        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_TRANSFER_FEE_ZERO_ADDRESS,
                &the_clock
            );
        let payload =
            governance_message::take_decree(vaa::payload(&verified_vaa));
        let cur = cursor::new(payload);

        bytes::take_u256_be(&mut cur);

        // Confirm recipient is zero address.
        let addr = bytes32::take_bytes(&mut cur);
        assert!(!bytes32::is_nonzero(&addr), 0);
        cursor::destroy_empty(cur);

        let ticket = transfer_fee::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        // You shall not pass!
        transfer_fee(&mut worm_state, receipt, test_scenario::ctx(scenario));

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::bytes32::E_U64_OVERFLOW)]
    fun test_cannot_transfer_fee_withdraw_amount_overflow() {
        // Testing this method.
        use wormhole::transfer_fee::{transfer_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `transfer_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Show balance is zero.
        assert!(state::fees_collected(&worm_state) == 0, 0);

        // Show that the encoded fee is greater than zero.
        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_TRANSFER_FEE_OVERFLOW,
                &the_clock
            );
        let payload =
            governance_message::take_decree(vaa::payload(&verified_vaa));
        let cur = cursor::new(payload);

        let amount = bytes::take_u256_be(&mut cur);
        assert!(amount > 0xffffffffffffffff, 0);
        cursor::take_rest(cur);

        let ticket = transfer_fee::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        // You shall not pass!
        transfer_fee(&mut worm_state, receipt, test_scenario::ctx(scenario));

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_set_fee_outdated_version() {
        // Testing this method.
        use wormhole::transfer_fee::{transfer_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `transfer_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Double-check current fee (from setup).
        assert!(state::message_fee(&worm_state) == wormhole_fee, 0);

        // Deposit fee several times.
        let (i, n) = (0, 8);
        while (i < n) {
            state::deposit_fee_test_only(
                &mut worm_state,
                balance::create_for_testing(wormhole_fee)
            );
            i = i + 1;
        };

        // Double-check balance.
        let total_deposited = n * wormhole_fee;
        assert!(state::fees_collected(&worm_state) == total_deposited, 0);

        // Prepare test to execute `transfer_fee`.
        test_scenario::next_tx(scenario, caller);

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut worm_state);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        state::migrate_version_test_only(
            &mut worm_state,
            version_control::previous_version_test_only(),
            version_control::next_version()
        );

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_TRANSFER_FEE_1,
                &the_clock
            );
        let ticket = transfer_fee::authorize_governance(&worm_state);
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);
        // You shall not pass!
        transfer_fee(&mut worm_state, receipt, test_scenario::ctx(scenario));

        abort 42
    }
}
