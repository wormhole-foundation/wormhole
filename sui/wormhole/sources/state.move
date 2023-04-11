// SPDX-License-Identifier: Apache 2

/// This module implements the global state variables for Wormhole as a shared
/// object. The `State` object is used to perform anything that requires access
/// to data that defines the Wormhole contract. Examples of which are publishing
/// Wormhole messages (requires depositing a message fee), verifying `VAA` by
/// checking signatures versus an existing Guardian set, and generating new
/// emitters for Wormhole integrators.
module wormhole::state {
    use std::vector::{Self};
    use sui::balance::{Balance};
    use sui::clock::{Clock};
    use sui::dynamic_field::{Self as field};
    use sui::object::{Self, ID, UID};
    use sui::package::{Self, UpgradeCap, UpgradeReceipt, UpgradeTicket};
    use sui::sui::{SUI};
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};

    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::consumed_vaas::{Self, ConsumedVAAs};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::fee_collector::{Self, FeeCollector};
    use wormhole::guardian::{Self};
    use wormhole::guardian_set::{Self, GuardianSet};
    use wormhole::required_version::{Self, RequiredVersion};
    use wormhole::version_control::{Self as control};

    friend wormhole::emitter;
    friend wormhole::governance_message;
    friend wormhole::migrate;
    friend wormhole::publish_message;
    friend wormhole::set_fee;
    friend wormhole::setup;
    friend wormhole::transfer_fee;
    friend wormhole::update_guardian_set;
    friend wormhole::upgrade_contract;
    friend wormhole::vaa;

    /// Cannot initialize state with zero guardians.
    const E_ZERO_GUARDIANS: u64 = 0;
    /// Build does not agree with expected upgrade.
    const E_BUILD_VERSION_MISMATCH: u64 = 1;

    /// Sui's chain ID is hard-coded to one value.
    const CHAIN_ID: u16 = 21;

    /// Used as key to finish upgrade process after upgrade has been committed.
    ///
    /// See `migrate` module for more info.
    struct MigrateTicket has store {}

    /// Container for all state variables for Wormhole.
    struct State has key, store {
        id: UID,

        /// Governance chain ID.
        governance_chain: u16,

        /// Governance contract address.
        governance_contract: ExternalAddress,

        /// Current active guardian set index.
        guardian_set_index: u32,

        /// All guardian sets (including expired ones).
        guardian_sets: Table<u32, GuardianSet>,

        /// Period for which a guardian set stays active after it has been
        /// replaced.
        ///
        /// NOTE: `Clock` timestamp is in units of ms while this value is in
        /// terms of seconds. See `guardian_set` module for more info.
        guardian_set_seconds_to_live: u32,

        /// Consumed VAA hashes to protect against replay. VAAs relevant to
        /// Wormhole are just governance VAAs.
        consumed_vaas: ConsumedVAAs,

        /// Wormhole fee collector.
        fee_collector: FeeCollector,

        /// Upgrade capability.
        upgrade_cap: UpgradeCap,

        /// Contract build version tracker.
        required_version: RequiredVersion
    }

    /// Create new `State`. This is only executed using the `setup` module.
    public(friend) fun new(
        upgrade_cap: UpgradeCap,
        governance_chain: u16,
        governance_contract: vector<u8>,
        initial_guardians: vector<vector<u8>>,
        guardian_set_seconds_to_live: u32,
        message_fee: u64,
        ctx: &mut TxContext
    ): State {
        // We need at least one guardian.
        assert!(vector::length(&initial_guardians) > 0, E_ZERO_GUARDIANS);

        // First guardian set index is zero. New guardian sets must increment
        // from the last recorded index.
        let guardian_set_index = 0;

        let governance_contract =
            external_address::new_nonzero(
                bytes32::from_bytes(governance_contract)
            );
        let state = State {
            id: object::new(ctx),
            governance_chain,
            governance_contract,
            guardian_set_index,
            guardian_sets: table::new(ctx),
            guardian_set_seconds_to_live,
            consumed_vaas: consumed_vaas::new(ctx),
            fee_collector: fee_collector::new(message_fee),
            upgrade_cap,
            required_version: required_version::new(control::version(), ctx)
        };

        let guardians = {
            let out = vector::empty();
            let cur = cursor::new(initial_guardians);
            while (!cursor::is_empty(&cur)) {
                vector::push_back(
                    &mut out,
                    guardian::new(cursor::poke(&mut cur))
                );
            };
            cursor::destroy_empty(cur);
            out
        };

        // Store the initial guardian set.
        add_new_guardian_set(
            &mut state,
            guardian_set::new(guardian_set_index, guardians)
        );

        let tracker = &mut state.required_version;
        required_version::add<control::Emitter>(tracker);
        required_version::add<control::GovernanceMessage>(tracker);
        required_version::add<control::Migrate>(tracker);
        required_version::add<control::PublishMessage>(tracker);
        required_version::add<control::SetFee>(tracker);
        required_version::add<control::TransferFee>(tracker);
        required_version::add<control::UpdateGuardianSet>(tracker);
        required_version::add<control::Vaa>(tracker);

        state
    }

    /// Convenience method to get hard-coded Wormhole chain ID (recognized by
    /// the Wormhole network).
    public fun chain_id(): u16 {
        CHAIN_ID
    }

    /// Retrieve governance module name.
    public fun governance_module(): Bytes32 {
        // A.K.A. "Core".
        bytes32::new(
            x"00000000000000000000000000000000000000000000000000000000436f7265"
        )
    }

    /// Retrieve current build version of latest upgrade.
    public fun current_version(self: &State): u64 {
        required_version::current(&self.required_version)
    }

    /// Issue an `UpgradeTicket` for the upgrade.
    public(friend) fun authorize_upgrade(
        self: &mut State,
        implementation_digest: Bytes32
    ): UpgradeTicket {
        // Check that the hard-coded version version agrees with the version
        // number in the `UpgradeCap`. We should only be allowed to upgrade
        // using the latest build.
        assert!(
            package::version(&self.upgrade_cap) == control::version(),
            E_BUILD_VERSION_MISMATCH
        );

        let policy = package::upgrade_policy(&self.upgrade_cap);

        // Finally authorize upgrade.
        package::authorize_upgrade(
            &mut self.upgrade_cap,
            policy,
            bytes32::to_bytes(implementation_digest),
        )
    }

    /// Finalize the upgrade that ran to produce the given `receipt`.
    public(friend) fun commit_upgrade(
        self: &mut State,
        receipt: UpgradeReceipt
    ): ID {
        // Check that the hard-coded version version agrees with the version
        // number in the `UpgradeCap`. We should only be allowed to upgrade
        // using the latest build.
        assert!(
            package::version(&self.upgrade_cap) == control::version(),
            E_BUILD_VERSION_MISMATCH
        );

        // Uptick the upgrade cap version number using this receipt.
        package::commit_upgrade(&mut self.upgrade_cap, receipt);

        // Update global version.
        required_version::update_latest(
            &mut self.required_version,
            &self.upgrade_cap
        );

        // Require that `migrate` be called only from the current build.
        require_current_version<control::Migrate>(self);

        // We require that a `MigrateTicket` struct be destroyed as the final
        // step to an upgrade by calling `migrate` from the `migrate` module.
        //
        // A separate method is required because `state` is a dependency of
        // `migrate`. This method warehouses state modifications required
        // for the new implementation plus enabling any methods required to be
        // gated by the current implementation version. In most cases `migrate`
        // is a no-op.
        //
        // The only case where this would fail is if `migrate` were not called
        // from a previous upgrade.
        //
        // See `migrate` module for more info.
        field::add(&mut self.id, b"migrate", MigrateTicket {});

        // Return the latest package ID.
        package::upgrade_package(&self.upgrade_cap)
    }

    /// Enforce a particular method to use the current build version as its
    /// minimum required version. This method ensures that a method is not
    /// backwards compatible with older builds.
    public(friend) fun require_current_version<ControlType>(self: &mut State) {
        required_version::require_current_version<ControlType>(
            &mut self.required_version,
        )
    }

    /// Check whether a particular method meets the minimum build version for
    /// the latest Wormhole implementation.
    public(friend) fun check_minimum_requirement<ControlType>(self: &State) {
        required_version::check_minimum_requirement<ControlType>(
            &self.required_version,
            control::version()
        )
    }

    /// After committing an upgrade, destroy `MigrateTicket`.
    ///
    /// See `wormhole::migrate` module for more info.
    public(friend) fun consume_migrate_ticket(self: &mut State) {
        let MigrateTicket {} = field::remove(&mut self.id, b"migrate");
    }

    /// Retrieve governance chain ID, which is governance's emitter chain ID.
    public fun governance_chain(self: &State): u16 {
        self.governance_chain
    }

    /// Retrieve governance emitter address.
    public fun governance_contract(self: &State): ExternalAddress {
        self.governance_contract
    }

    /// Retrieve current Guardian set index. This value is important for
    /// verifying VAA signatures and especially important for governance VAAs.
    public fun guardian_set_index(self: &State): u32 {
        self.guardian_set_index
    }

    /// Retrieve how long after a Guardian set can live for in terms of Sui
    /// timestamp (in seconds).
    public fun guardian_set_seconds_to_live(self: &State): u32 {
        self.guardian_set_seconds_to_live
    }

    /// Retrieve current fee to send Wormhole message.
    public fun message_fee(self: &State): u64 {
        fee_collector::fee_amount(&self.fee_collector)
    }

    /// Deposit fee when sending Wormhole message. This method does not
    /// necessarily have to be a `friend` to `wormhole::publish_message`. But
    /// we also do not want an integrator to mistakenly deposit fees outside
    /// of calling `publish_message`.
    ///
    /// See `wormhole::publish_message` for more info.
    public(friend) fun deposit_fee(self: &mut State, fee: Balance<SUI>) {
        fee_collector::deposit_balance(&mut self.fee_collector, fee);
    }

    #[test_only]
    public fun deposit_fee_test_only(self: &mut State, fee: Balance<SUI>) {
        deposit_fee(self, fee)
    }

    /// Withdraw collected fees when governance action to transfer fees to a
    /// particular recipient.
    ///
    /// See `wormhole::transfer_fee` for more info.
    public(friend) fun withdraw_fee(
        self: &mut State,
        amount: u64
    ): Balance<SUI> {
        fee_collector::withdraw_balance(&mut self.fee_collector, amount)
    }

    /// Store `VAA` hash as a way to claim a VAA. This method prevents a VAA
    /// from being replayed. For Wormhole, the only VAAs that it cares about
    /// being replayed are its governance actions.
    public(friend) fun borrow_mut_consumed_vaas(
        self: &mut State
    ): &mut ConsumedVAAs {
        &mut self.consumed_vaas
    }

    /// When a new guardian set is added to `State`, part of the process
    /// involves setting the last known Guardian set's expiration time based
    /// on how long a Guardian set can live for.
    ///
    /// See `guardian_set_epochs_to_live` for the parameter that determines how
    /// long a Guardian set can live for.
    ///
    /// See `wormhole::update_guardian_set` for more info.
    ///
    /// TODO: Use `Clock` instead of `TxContext`.
    public(friend) fun expire_guardian_set(
        self: &mut State,
        the_clock: &Clock
    ) {
        guardian_set::set_expiration(
            table::borrow_mut(&mut self.guardian_sets, self.guardian_set_index),
            self.guardian_set_seconds_to_live,
            the_clock
        );
    }

    /// Add the latest Guardian set from the governance action to update the
    /// current guardian set.
    ///
    /// See `wormhole::update_guardian_set` for more info.
    public(friend) fun add_new_guardian_set(
        self: &mut State,
        new_guardian_set: GuardianSet
    ) {
        self.guardian_set_index = guardian_set::index(&new_guardian_set);
        table::add(
            &mut self.guardian_sets,
            self.guardian_set_index,
            new_guardian_set
        );
    }

    /// Modify the cost to send a Wormhole message via governance.
    ///
    /// See `wormhole::set_fee` for more info.
    public(friend) fun set_message_fee(self: &mut State, amount: u64) {
        fee_collector::change_fee(&mut self.fee_collector, amount);
    }

    /// Retrieve a particular Guardian set by its Guardian set index. This
    /// method is used when verifying a VAA.
    ///
    /// See `wormhole::vaa` for more info.
    public fun guardian_set_at(self: &State, index: u32): &GuardianSet {
        table::borrow(&self.guardian_sets, index)
    }

    #[test_only]
    public fun fees_collected(self: &State): u64 {
        fee_collector::balance_value(&self.fee_collector)
    }

    #[test_only]
    public fun set_required_version<ControlType>(
        self: &mut State,
        version: u64
    ) {
        required_version::set_required_version<ControlType>(
            &mut self.required_version,
            version
        )
    }

    #[test_only]
    public fun test_upgrade(self: &mut State) {
        use sui::hash::{keccak256};

        let ticket =
            authorize_upgrade(self, bytes32::new(keccak256(&b"new build")));
        let receipt = package::test_upgrade(ticket);

        commit_upgrade(self, receipt);

        // Destroy migration key to wrap things up.
        consume_migrate_ticket(self);
    }
}
