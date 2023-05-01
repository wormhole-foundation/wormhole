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
    use sui::object::{Self, ID, UID};
    use sui::package::{UpgradeCap, UpgradeReceipt, UpgradeTicket};
    use sui::sui::{SUI};
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};

    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::consumed_vaas::{Self, ConsumedVAAs};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::fee_collector::{Self, FeeCollector};
    use wormhole::guardian::{Guardian};
    use wormhole::guardian_set::{Self, GuardianSet};
    use wormhole::package_utils::{Self};
    use wormhole::version_control::{Self};

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
    /// Build digest does not agree with current implementation.
    const E_INVALID_BUILD_DIGEST: u64 = 1;

    /// Sui's chain ID is hard-coded to one value.
    const CHAIN_ID: u16 = 21;

    /// Capability reflecting that the current build version is used to invoke
    /// state methods.
    struct LatestOnly has drop {}

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
        upgrade_cap: UpgradeCap
    }

    /// Create new `State`. This is only executed using the `setup` module.
    public(friend) fun new(
        upgrade_cap: UpgradeCap,
        governance_chain: u16,
        governance_contract: ExternalAddress,
        guardian_set_index: u32,
        initial_guardians: vector<Guardian>,
        guardian_set_seconds_to_live: u32,
        message_fee: u64,
        ctx: &mut TxContext
    ): State {
        // We need at least one guardian.
        assert!(vector::length(&initial_guardians) > 0, E_ZERO_GUARDIANS);

        let state = State {
            id: object::new(ctx),
            governance_chain,
            governance_contract,
            guardian_set_index,
            guardian_sets: table::new(ctx),
            guardian_set_seconds_to_live,
            consumed_vaas: consumed_vaas::new(ctx),
            fee_collector: fee_collector::new(message_fee),
            upgrade_cap
        };

        // Set first version and initialize package info. This will be used for
        // emitting information of successful migrations.
        let upgrade_cap = &state.upgrade_cap;
        package_utils::init_package_info(
            &mut state.id,
            version_control::current_version(),
            upgrade_cap
        );

        // Store the initial guardian set.
        add_new_guardian_set(
            &assert_latest_only(&state),
            &mut state,
            guardian_set::new(guardian_set_index, initial_guardians)
        );

        state
    }

    ////////////////////////////////////////////////////////////////////////////
    //
    //  Simple Getters
    //
    //  These methods do not require `LatestOnly` for access. Anyone is free to
    //  access these values.
    //
    ////////////////////////////////////////////////////////////////////////////

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

    /// Retrieve a particular Guardian set by its Guardian set index. This
    /// method is used when verifying a VAA.
    ///
    /// See `wormhole::vaa` for more info.
    public fun guardian_set_at(
        self: &State,
        index: u32
    ): &GuardianSet {
        table::borrow(&self.guardian_sets, index)
    }

    /// Retrieve current fee to send Wormhole message.
    public fun message_fee(self: &State): u64 {
        fee_collector::fee_amount(&self.fee_collector)
    }

    #[test_only]
    public fun fees_collected(self: &State): u64 {
        fee_collector::balance_value(&self.fee_collector)
    }

    #[test_only]
    public fun cache_latest_only_test_only(self: &State): LatestOnly {
        assert_latest_only(self)
    }

    #[test_only]
    public fun deposit_fee_test_only(self: &mut State, fee: Balance<SUI>) {
        deposit_fee(&assert_latest_only(self), self, fee)
    }

    #[test_only]
    public fun migrate_version_test_only<Old: store + drop, New: store + drop>(
        self: &mut State,
        old_version: Old,
        new_version: New
    ) {
        package_utils::update_version_type_test_only(
            &mut self.id,
            old_version,
            new_version
        );
    }

    #[test_only]
    public fun test_upgrade(self: &mut State) {
        let test_digest = bytes32::from_bytes(b"new build");
        let ticket = authorize_upgrade(self, test_digest);
        let receipt = sui::package::test_upgrade(ticket);
        commit_upgrade(self, receipt);
    }

    #[test_only]
    public fun reverse_migrate_version(self: &mut State) {
        package_utils::update_version_type_test_only(
            &mut self.id,
            version_control::current_version(),
            version_control::previous_version()
        );
    }

    ////////////////////////////////////////////////////////////////////////////
    //
    //  Privileged `State` Access
    //
    //  This section of methods require a `LatestOnly`, which can only be created
    //  within the Wormhole package. This capability allows special access to
    //  the `State` object.
    //
    //  NOTE: A lot of these methods are still marked as `(friend)` as a safety
    //  precaution. When a package is upgraded, friend modifiers can be
    //  removed.
    //
    ////////////////////////////////////////////////////////////////////////////

    /// Obtain a capability to interact with `State` methods. This method checks
    /// that we are running the current build.
    ///
    /// NOTE: This method allows caching the current version check so we avoid
    /// multiple checks to dynamic fields.
    public(friend) fun assert_latest_only(self: &State): LatestOnly {
        package_utils::assert_version(
            &self.id,
            version_control::current_version()
        );

        LatestOnly {}
    }

    /// Deposit fee when sending Wormhole message. This method does not
    /// necessarily have to be a `friend` to `wormhole::publish_message`. But
    /// we also do not want an integrator to mistakenly deposit fees outside
    /// of calling `publish_message`.
    ///
    /// See `wormhole::publish_message` for more info.
    public(friend) fun deposit_fee(
        _: &LatestOnly,
        self: &mut State,
        fee: Balance<SUI>
    ) {
        fee_collector::deposit_balance(&mut self.fee_collector, fee);
    }

    /// Withdraw collected fees when governance action to transfer fees to a
    /// particular recipient.
    ///
    /// See `wormhole::transfer_fee` for more info.
    public(friend) fun withdraw_fee(
        _: &LatestOnly,
        self: &mut State,
        amount: u64
    ): Balance<SUI> {
        fee_collector::withdraw_balance(&mut self.fee_collector, amount)
    }

    /// Store `VAA` hash as a way to claim a VAA. This method prevents a VAA
    /// from being replayed. For Wormhole, the only VAAs that it cares about
    /// being replayed are its governance actions.
    public(friend) fun borrow_mut_consumed_vaas(
        _: &LatestOnly,
        self: &mut State
    ): &mut ConsumedVAAs {
        borrow_mut_consumed_vaas_unchecked(self)
    }

    /// Store `VAA` hash as a way to claim a VAA. This method prevents a VAA
    /// from being replayed. For Wormhole, the only VAAs that it cares about
    /// being replayed are its governance actions.
    ///
    /// NOTE: This method does not require `LatestOnly`. Only methods in the
    /// `upgrade_contract` module requires this to be unprotected to prevent
    /// a corrupted upgraded contract from bricking upgradability.
    public(friend) fun borrow_mut_consumed_vaas_unchecked(
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
    public(friend) fun expire_guardian_set(
        _: &LatestOnly,
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
        _: &LatestOnly,
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
    public(friend) fun set_message_fee(
        _: &LatestOnly,
        self: &mut State,
        amount: u64
    ) {
        fee_collector::change_fee(&mut self.fee_collector, amount);
    }

    public(friend) fun current_package(_: &LatestOnly, self: &State): ID {
        package_utils::current_package(&self.id)
    }

    ////////////////////////////////////////////////////////////////////////////
    //
    //  Upgradability
    //
    //  A special space that controls upgrade logic. These methods are invoked
    //  via the `upgrade_contract` module.
    //
    //  Also in this section is managing contract migrations, which uses the
    //  `migrate` module to officially roll state access to the latest build.
    //  Only those methods that require `LatestOnly` will be affected by an
    //  upgrade.
    //
    ////////////////////////////////////////////////////////////////////////////

    /// Issue an `UpgradeTicket` for the upgrade.
    ///
    /// NOTE: The Sui VM performs a check that this method is executed from the
    /// latest published package. If someone were to try to execute this using
    /// a stale build, the transaction will revert with `PackageUpgradeError`,
    /// specifically `PackageIDDoesNotMatch`.
    public(friend) fun authorize_upgrade(
        self: &mut State,
        package_digest: Bytes32
    ): UpgradeTicket {
        let cap = &mut self.upgrade_cap;
        package_utils::authorize_upgrade(&mut self.id, cap, package_digest)
    }

    /// Finalize the upgrade that ran to produce the given `receipt`.
    ///
    /// NOTE: The Sui VM performs a check that this method is executed from the
    /// latest published package. If someone were to try to execute this using
    /// a stale build, the transaction will revert with `PackageUpgradeError`,
    /// specifically `PackageIDDoesNotMatch`.
    public(friend) fun commit_upgrade(
        self: &mut State,
        receipt: UpgradeReceipt
    ): (ID, ID) {
        let cap = &mut self.upgrade_cap;
        package_utils::commit_upgrade(&mut self.id, cap, receipt)
    }

    /// Method executed by the `migrate` module to roll access from one package
    /// to another. This method will be called from the upgraded package.
    public(friend) fun migrate_version(self: &mut State) {
        package_utils::migrate_version(
            &mut self.id,
            version_control::previous_version(),
            version_control::current_version()
        );
    }

    /// As a part of the migration, we verify that the upgrade contract VAA's
    /// encoded package digest used in `migrate` equals the one used to conduct
    /// the upgrade.
    public(friend) fun assert_authorized_digest(
        _: &LatestOnly,
        self: &State,
        digest: Bytes32
    ) {
        let authorized = package_utils::authorized_digest(&self.id);
        assert!(digest == authorized, E_INVALID_BUILD_DIGEST);
    }

    ////////////////////////////////////////////////////////////////////////////
    //
    //  Special State Interaction via Migrate
    //
    //  A VERY special space that manipulates `State` via calling `migrate`.
    //
    //  PLEASE KEEP ANY METHODS HERE AS FRIENDS. We want the ability to remove
    //  these for future builds.
    //
    ////////////////////////////////////////////////////////////////////////////

    /// This method is used to make modifications to `State` when `migrate` is
    /// called. This method name should change reflecting which version this
    /// contract is migrating to.
    ///
    /// NOTE: Please keep this method as public(friend) because we never want
    /// to expose this method as a public method.
    public(friend) fun migrate__v__0_2_0(_self: &mut State) {
        // Intentionally do nothing.
    }

    #[test_only]
    /// Bloody hack.
    ///
    /// This method is used to set up tests where we migrate to a new version,
    /// which is meant to test that modules protected by version control will
    /// break.
    public fun reverse_migrate__v__dummy(_self: &mut State) {
        // Intentionally do nothing.
    }

    ////////////////////////////////////////////////////////////////////////////
    //
    //  Deprecated
    //
    //  Dumping grounds for old structs and methods. These things should not
    //  be used in future builds.
    //
    ////////////////////////////////////////////////////////////////////////////
}
