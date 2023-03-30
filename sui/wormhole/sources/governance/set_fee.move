// SPDX-License-Identifier: Apache 2

/// This module implements handling a governance VAA to enact setting the
/// Wormhole message fee to another amount.
module wormhole::set_fee {
    use sui::clock::{Clock};

    use wormhole::bytes32::{Self};
    use wormhole::cursor::{Self};
    use wormhole::governance_message::{Self, GovernanceMessage};
    use wormhole::state::{Self, State};
    use wormhole::version_control::{SetFee as SetFeeControl};

    /// Specific governance payload ID (action) for setting Wormhole fee.
    const ACTION_SET_FEE: u8 = 3;

    struct SetFee {
        amount: u64
    }

    /// Redeem governance VAA to configure Wormhole message fee amount in SUI
    /// denomination. This governance message is only relevant for Sui because
    /// fee administration is only relevant to one particular network (in this
    /// case Sui).
    ///
    /// NOTE: This method is guarded by a minimum build version check. This
    /// method could break backward compatibility on an upgrade.
    public fun set_fee(
        wormhole_state: &mut State,
        vaa_buf: vector<u8>,
        the_clock: &Clock
    ): u64 {
        state::check_minimum_requirement<SetFeeControl>(wormhole_state);

        let msg =
            governance_message::parse_and_verify_vaa(
                wormhole_state,
                vaa_buf,
                the_clock
            );

        // Do not allow this VAA to be replayed.
        state::consume_vaa_hash(
            wormhole_state,
            governance_message::vaa_hash(&msg)
        );

        // Proceed with setting the new message fee.
        handle_set_fee(wormhole_state, msg)
    }

    fun handle_set_fee(
        wormhole_state: &mut State,
        msg: GovernanceMessage
    ): u64 {
        // Verify that this governance message is to update the Wormhole fee.
        let governance_payload =
            governance_message::take_local_action(
                msg,
                state::governance_module(),
                ACTION_SET_FEE
            );

        // Deserialize the payload as amount to change the Wormhole fee.
        let SetFee { amount } = deserialize(governance_payload);

        state::set_message_fee(wormhole_state, amount);

        amount
    }

    fun deserialize(payload: vector<u8>): SetFee {
        let cur = cursor::new(payload);

        // This amount cannot be greater than max u64.
        let amount = bytes32::to_u64_be(bytes32::take_bytes(&mut cur));

        cursor::destroy_empty(cur);

        SetFee { amount: (amount as u64) }
    }

    #[test_only]
    public fun action(): u8 {
        ACTION_SET_FEE
    }
}

#[test_only]
module wormhole::set_fee_tests {
    use sui::balance::{Self};
    use sui::test_scenario::{Self};

    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::governance_message::{Self};
    use wormhole::required_version::{Self};
    use wormhole::set_fee::{Self};
    use wormhole::state::{Self};
    use wormhole::version_control::{Self as control};
    use wormhole::wormhole_scenario::{
        person,
        return_clock,
        return_state,
        set_up_wormhole,
        take_clock,
        take_state,
        upgrade_wormhole
    };

    const VAA_SET_FEE_1: vector<u8> =
        x"01000000000100181aa27fd44f3060fad0ae72895d42f97c45f7a5d34aa294102911370695e91e17ae82caa59f779edde2356d95cd46c2c381cdeba7a8165901a562374f212d750000bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f7265030015000000000000000000000000000000000000000000000000000000000000015e";
    const VAA_SET_FEE_MAX: vector<u8> =
        x"01000000000100b0697fd31572e11b2256cf46d5934f38fbb90e6265e999bee50950846bf9f94d5b86f247cce20e3cc158163be7b5ae21ebaaf67e20d597229ca04d505fd4bc1c0000bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f7265030015000000000000000000000000000000000000000000000000ffffffffffffffff";
    const VAA_BOGUS_TARGET_CHAIN: vector<u8> =
        x"010000000001000d34e2f56f1558252796b631f12b4f85b991d3ef52de57df6d45c8ad998c58202655d5cab2d6613688c0fcd7ae1504fc474eb96ec0591598339c133cbf8b68240000bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f7265030002000000000000000000000000000000000000000000000000000000000000015e";
    const VAA_BOGUS_ACTION: vector<u8> =
        x"01000000000100bd79122fe306c2edd56ac5938acabfdadaae7906dfbfc140bd2f76f3996b20210fcda38caee5901d19d7f8b70169a46cf605f3f234d387c72a03accaf96f01ef0100bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f7265010015000000000000000000000000000000000000000000000000000000000000015e";
    const VAA_SET_FEE_OVERFLOW: vector<u8> =
        x"01000000000100950a509a797c9b40a678a5d6297f5b74e1ce1794b3c012dad5774c395e65e8b0773cf160113f571f1452ee98d10aa61273b6bc8aefa74a3c8f7e2c9c89fb25fa0000bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f72650300150000000000000000000000000000000000000000000000010000000000000000";

    #[test]
    public fun test_set_fee() {
        // Testing this method.
        use wormhole::set_fee::{set_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 420;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `set_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Double-check current fee (from setup).
        assert!(state::message_fee(&worm_state) == wormhole_fee, 0);

        let fee_amount = set_fee(&mut worm_state, VAA_SET_FEE_1, &the_clock);
        assert!(wormhole_fee != fee_amount, 0);

        // Confirm the fee changed.
        assert!(state::message_fee(&worm_state) == fee_amount, 0);

        // And confirm that we can deposit the new fee amount.
        state::deposit_fee_test_only(
            &mut worm_state,
            balance::create_for_testing(fee_amount)
        );

        // Finally set the fee again to max u64 (this will effectively pause
        // Wormhole message publishing until the fee gets adjusted back to a
        // reasonable level again).
        let fee_amount = set_fee(&mut worm_state, VAA_SET_FEE_MAX, &the_clock);

        // Confirm.
        assert!(state::message_fee(&worm_state) == fee_amount, 0);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    public fun test_set_fee_after_upgrade() {
        // Testing this method.
        use wormhole::set_fee::{set_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 420;
        set_up_wormhole(scenario, wormhole_fee);

        // Upgrade.
        upgrade_wormhole(scenario);

        // Prepare test to execute `set_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Double-check current fee (from setup).
        assert!(state::message_fee(&worm_state) == wormhole_fee, 0);

        let fee_amount = set_fee(&mut worm_state, VAA_SET_FEE_1, &the_clock);

        // Confirm the fee changed.
        assert!(state::message_fee(&worm_state) == fee_amount, 0);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = state::E_VAA_ALREADY_CONSUMED)]
    public fun test_cannot_set_fee_with_same_vaa() {
        // Testing this method.
        use wormhole::set_fee::{set_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 420;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `set_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Set once.
        set_fee(&mut worm_state, VAA_SET_FEE_1, &the_clock);

        // You shall not pass!
        set_fee(&mut worm_state, VAA_SET_FEE_1, &the_clock);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_GOVERNANCE_TARGET_CHAIN_NOT_SUI
    )]
    public fun test_cannot_set_fee_invalid_target_chain() {
        // Testing this method.
        use wormhole::set_fee::{set_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 420;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `set_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Setting a new fee only applies to this chain since the denomination
        // is SUI.
        let msg =
            governance_message::parse_and_verify_vaa(
                &worm_state,
                VAA_BOGUS_TARGET_CHAIN,
                &the_clock
            );
        assert!(!governance_message::is_local_action(&msg), 0);
        governance_message::destroy(msg);

        // You shall not pass!
        set_fee(&mut worm_state, VAA_BOGUS_TARGET_CHAIN, &the_clock);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_INVALID_GOVERNANCE_ACTION
    )]
    public fun test_cannot_set_fee_invalid_action() {
        // Testing this method.
        use wormhole::set_fee::{set_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 420;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `set_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Setting a new fee only applies to this chain since the denomination
        // is SUI.
        let msg =
            governance_message::parse_and_verify_vaa(
                &worm_state,
                VAA_BOGUS_ACTION,
                &the_clock
            );
        assert!(governance_message::action(&msg) != set_fee::action(), 0);
        governance_message::destroy(msg);

        // You shall not pass!
        set_fee(&mut worm_state, VAA_BOGUS_ACTION, &the_clock);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wormhole::bytes32::E_U64_OVERFLOW)]
    public fun test_cannot_set_fee_with_overflow() {
        // Testing this method.
        use wormhole::set_fee::{set_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 420;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `set_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Show that the encoded fee is greater than u64 max.
        let msg =
            governance_message::parse_and_verify_vaa(
                &worm_state,
                VAA_SET_FEE_OVERFLOW,
                &the_clock
            );
        let payload = governance_message::take_payload(msg);
        let cur = cursor::new(payload);

        let fee_amount = bytes::take_u256_be(&mut cur);
        assert!(fee_amount > 0xffffffffffffffff, 0);

        cursor::destroy_empty(cur);

        // You shall not pass!
        set_fee(&mut worm_state, VAA_SET_FEE_OVERFLOW, &the_clock);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = required_version::E_OUTDATED_VERSION)]
    public fun test_cannot_set_fee_outdated_build() {
        // Testing this method.
        use wormhole::set_fee::{set_fee};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 420;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test to execute `set_fee`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        state::set_required_version<control::SetFee>(
            &mut worm_state,
            control::version() + 1
        );

        // You shall not pass!
        set_fee(&mut worm_state, VAA_SET_FEE_1, &the_clock);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }
}
