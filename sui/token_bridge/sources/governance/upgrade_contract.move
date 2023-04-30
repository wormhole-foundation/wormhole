// SPDX-License-Identifier: Apache 2

/// This module implements handling a governance VAA to enact upgrading the
/// Token Bridge contract to a new build. The procedure to upgrade this contract
/// requires a Programmable Transaction, which includes the following procedure:
/// 1.  Load new build.
/// 2.  Authorize upgrade.
/// 3.  Upgrade.
/// 4.  Commit upgrade.
module token_bridge::upgrade_contract {
    use sui::object::{ID};
    use sui::package::{UpgradeReceipt, UpgradeTicket};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::cursor::{Self};
    use wormhole::governance_message::{Self, DecreeTicket, DecreeReceipt};

    use token_bridge::state::{Self, State};

    friend token_bridge::migrate;

    /// Digest is all zeros.
    const E_DIGEST_ZERO_BYTES: u64 = 0;

    /// Specific governance payload ID (action) to complete upgrading the
    /// contract.
    const ACTION_UPGRADE_CONTRACT: u8 = 2;

    struct GovernanceWitness has drop {}

    // Event reflecting package upgrade.
    struct ContractUpgraded has drop, copy {
        old_contract: ID,
        new_contract: ID
    }

    struct UpgradeContract {
        digest: Bytes32
    }

    public fun authorize_governance(
        token_bridge_state: &State
    ): DecreeTicket<GovernanceWitness> {
        governance_message::authorize_verify_local(
            GovernanceWitness {},
            state::governance_chain(token_bridge_state),
            state::governance_contract(token_bridge_state),
            state::governance_module(),
            ACTION_UPGRADE_CONTRACT
        )
    }

    /// Redeem governance VAA to issue an `UpgradeTicket` for the upgrade given
    /// a contract upgrade VAA. This governance message is only relevant for Sui
    /// because a contract upgrade is only relevant to one particular network
    /// (in this case Sui), whose build digest is encoded in this message.
    public fun authorize_upgrade(
        token_bridge_state: &mut State,
        receipt: DecreeReceipt<GovernanceWitness>
    ): UpgradeTicket {
        // current package checking when consuming VAA hashes. This is because
        // upgrades are protected by the Sui VM, enforcing the latest package
        // is the one performing the upgrade.
        let consumed =
            state::borrow_mut_consumed_vaas_unchecked(token_bridge_state);

        // And consume.
        let payload = governance_message::take_payload(consumed, receipt);

        // Proceed with processing new implementation version.
        handle_upgrade_contract(token_bridge_state, payload)
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
        sui::event::emit(ContractUpgraded { old_contract, new_contract });
    }

    /// Privileged method only to be used by this module and `migrate` module.
    ///
    /// During migration, we make sure that the digest equals what we expect by
    /// passing in the same VAA used to upgrade the package.
    public(friend) fun take_digest(governance_payload: vector<u8>): Bytes32 {
        // Deserialize the payload as the build digest.
        let UpgradeContract { digest } = deserialize(governance_payload);

        digest
    }

    fun handle_upgrade_contract(
        wormhole_state: &mut State,
        payload: vector<u8>
    ): UpgradeTicket {
        state::authorize_upgrade(wormhole_state, take_digest(payload))
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
