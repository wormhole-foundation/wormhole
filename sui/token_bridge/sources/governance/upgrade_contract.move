// SPDX-License-Identifier: Apache 2

/// This module implements handling a governance VAA to enact upgrading the
/// Token Bridge contract to a new build. The procedure to upgrade this contract
/// requires a Programmable Transaction, which includes the following procedure:
/// 1.  Load new build.
/// 2.  Authorize upgrade.
/// 3.  Upgrade.
/// 4.  Commit upgrade.
module token_bridge::upgrade_contract {
    use sui::event::{Self};
    use sui::object::{ID};
    use sui::package::{UpgradeReceipt, UpgradeTicket};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::consumed_vaas::{Self};
    use wormhole::cursor::{Self};
    use wormhole::governance_message::{Self, GovernanceMessage};

    use token_bridge::state::{Self, State};

    friend token_bridge::migrate;

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
    public fun authorize_upgrade(
        token_bridge_state: &mut State,
        msg: GovernanceMessage
    ): UpgradeTicket {
        // Do not allow this VAA to be replayed.
        consumed_vaas::consume(
            state::borrow_mut_consumed_vaas_unchecked(token_bridge_state),
            governance_message::vaa_hash(&msg)
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
        let (old_contract, new_contract) = state::commit_upgrade(self, receipt);

        // Emit an event reflecting package ID change.
        event::emit(ContractUpgraded { old_contract, new_contract });
    }

    fun handle_upgrade_contract(
        wormhole_state: &mut State,
        msg: GovernanceMessage
    ): UpgradeTicket {
        state::authorize_upgrade(wormhole_state, take_digest(msg))
    }

    /// Privileged method only to be used by this module and `migrate` module.
    ///
    /// During migration, we make sure that the digest equals what we expect by
    /// passing in the same VAA used to upgrade the package.
    public(friend) fun take_digest(msg: GovernanceMessage): Bytes32 {
        // Verify that this governance message is to update the Wormhole fee.
        let governance_payload =
            governance_message::take_local_action(
                msg,
                state::governance_module(),
                ACTION_UPGRADE_CONTRACT
            );

        // Deserialize the payload as the build digest.
        let UpgradeContract { digest } = deserialize(governance_payload);

        digest
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
