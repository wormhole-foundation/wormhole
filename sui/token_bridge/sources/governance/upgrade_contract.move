// SPDX-License-Identifier: Apache 2

/// This module implements handling a governance VAA to enact upgrading the
/// Token Bridge contract to a new build. The procedure to upgrade this contract
/// requires a Programmable Transaction, which includes the following procedure:
/// 1.  Load new build.
/// 2.  Authorize upgrade.
/// 3.  Upgrade.
/// 4.  Commit upgrade.
module token_bridge::upgrade_contract {
    use sui::clock::{Clock};
    use sui::event::{Self};
    use sui::object::{Self, ID};
    use sui::package::{UpgradeReceipt, UpgradeTicket};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::cursor::{Self};
    use wormhole::governance_message::{Self, GovernanceMessage};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{Self, State};

    /// Digest is all zeros.
    const E_DIGEST_ZERO_BYTES: u64 = 0;

    /// Specific governance payload ID (action) to complete upgrading the
    /// contract.
    const ACTION_UPGRADE_CONTRACT: u8 = 1;

    // Event reflecting package upgrade.
    struct ContractUpgraded has drop, copy {
        old_contract: ID,
        new_contract: ID
    }

    struct UpgradeContract {
        digest: Bytes32
    }

    /// Redeem governance VAA to issue an `UpgradeTicket` for the upgrade given
    /// a contract upgrade VAA. This governance message is only relevant for Sui
    /// because a contract upgrade is only relevant to one particular network
    /// (in this case Sui), whose build digest is encoded in this message.
    ///
    /// NOTE: This method is guarded by a minimum build version check. This
    /// method could break backward compatibility on an upgrade.
    public fun upgrade_contract(
        token_bridge_state: &mut State,
        wormhole_state: &WormholeState,
        vaa_buf: vector<u8>,
        the_clock: &Clock
    ): UpgradeTicket {
        // Do not allow this VAA to be replayed.
        let msg =
            governance_message::parse_verify_and_consume_vaa(
                state::borrow_mut_consumed_vaas(token_bridge_state),
                wormhole_state,
                vaa_buf,
                the_clock
            );

        // Proceed with processing new implementation version.
        handle_upgrade_contract(token_bridge_state, msg)
    }

    /// Finalize the upgrade that ran to produce the given `receipt`. This
    /// method invokes `state::commit_upgrade` which interacts with
    /// `sui::package`.
    public fun commit_upgrade(
        self: &mut State,
        receipt: UpgradeReceipt,
    ) {
        let latest_package_id = state::commit_upgrade(self, receipt);

        // Emit an event reflecting package ID change.
        event::emit(
            ContractUpgraded {
                old_contract: object::id_from_address(@token_bridge),
                new_contract: latest_package_id
            }
        );
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
        let digest = bytes32::take_bytes(&mut cur);
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
module token_bridge::upgrade_contract_tests {
    // TODO
}
