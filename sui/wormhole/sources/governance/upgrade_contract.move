module wormhole::upgrade_contract {
    use sui::tx_context::{TxContext};

    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::cursor::{Self};
    use wormhole::governance_message::{Self, GovernanceMessage};
    use wormhole::state::{Self, State};

    // NOTE: This exists to mock up sui::package for proposed upgrades.
    use wormhole::dummy_sui_package::{UpgradeReceipt, UpgradeTicket};

    const E_DIGEST_ZERO_BYTES: u64 = 0;

    /// Specific governance payload ID (action) to complete upgrading the
    /// contract.
    const ACTION_UPGRADE_CONTRACT: u8 = 1;

    struct UpgradeContract {
        digest: Bytes32
    }

    /// Issue an `UpgradeTicket` for the upgrade given a contract upgrade VAA.
    public fun upgrade_contract(
        wormhole_state: &mut State,
        vaa_buf: vector<u8>,
        ctx: &TxContext
    ): UpgradeTicket {
        let msg =
            governance_message::parse_and_verify_vaa(
                wormhole_state,
                vaa_buf,
                ctx
            );

        // Do not allow this VAA to be replayed.
        state::consume_vaa_hash(
            wormhole_state,
            governance_message::vaa_hash(&msg)
        );

        // Proceed with processing new implementation version.
        handle_upgrade_contract(wormhole_state, msg)
    }

    /// Finalize the upgrade that ran to produce the given `receipt`. This
    /// method invokes `state::commit_upgrade` which interacts with
    /// `sui::package`.
    public fun commit_upgrade(
        self: &mut State,
        receipt: UpgradeReceipt,
    ) {
        state::commit_upgrade(self, receipt)
    }

    fun handle_upgrade_contract(
        wormhole_state: &mut State,
        msg: GovernanceMessage
    ): UpgradeTicket {
        // Verify that this governance message is to update the Wormhole fee.
        let governance_payload =
            governance_message::take_local_action(
                msg,
                state::governance_module(),
                ACTION_UPGRADE_CONTRACT
            );

        // Deserialize the payload as amount to change the Wormhole fee.
        let UpgradeContract { digest } = deserialize(governance_payload);

        state::authorize_upgrade(wormhole_state, digest)
    }

    fun deserialize(payload: vector<u8>): UpgradeContract {
        let cur = cursor::new(payload);

        // This amount cannot be greater than max u64.
        let digest = bytes32::take(&mut cur);
        assert!(bytes32::is_nonzero(&digest), E_DIGEST_ZERO_BYTES);

        cursor::destroy_empty(cur);

        UpgradeContract { digest }
    }

    #[test_only]
    public fun action(): u8 {
        ACTION_UPGRADE_CONTRACT
    }
}

#[test_only]
module wormhole::upgrade_contract_test {
    use sui::test_scenario::{Self};

    use wormhole::dummy_sui_package::{test_upgrade};
    use wormhole::state::{Self, State};
    use wormhole::wormhole_scenario::{
        person,
        set_up_wormhole,
        //upgrade_wormhole
    };
    use wormhole::upgrade_contract::{Self};

    #[test]
    /// In this test, we test the following sequence of methods (values in
    /// parentheses are arguments return by the previous method and passed into
    /// the next method):
    ///
    /// upgrade_contract -> (UpgradeTicket) -> test_upgrade -> (UpgradeReceipt) -> commit_upgrade
    ///
    public fun test_update_contract() {

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 0;
        set_up_wormhole(scenario, wormhole_fee);

        test_scenario::next_tx(scenario, caller);

        let worm_state = test_scenario::take_shared<State>(scenario);

        // TODO - put a legitimate VAA here
        let vaa_buf = x"00";

        // TODO - do we need to authorize upgrade first?

        // Obtain an upgrade_ticket.
        let upgrade_ticket = upgrade_contract::upgrade_contract(
            &mut worm_state,
            vaa_buf,
            test_scenario::ctx(&mut my_scenario)
        );

        // test_upgrade generates a fake package ID for the new package and
        // converts the ticket to a receipt.
        let upgrade_receipt = test_upgrade(upgrade_ticket);

        // Clean up.
        test_scenario::return_shared(worm_state);

        // Done.
        test_scenario::end(my_scenario);
    }
}
