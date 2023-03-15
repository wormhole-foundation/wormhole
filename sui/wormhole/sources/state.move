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
    use sui::dynamic_field::{Self as field};
    use sui::object::{Self, UID};
    use sui::package::{Self, UpgradeCap, UpgradeReceipt, UpgradeTicket};
    use sui::sui::{SUI};
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};

    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::cursor::{Self};
    use wormhole::emitter::{Self, EmitterCap, EmitterRegistry};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::fee_collector::{Self, FeeCollector};
    use wormhole::guardian::{Self};
    use wormhole::guardian_set::{Self, GuardianSet};
    use wormhole::required_version::{Self, RequiredVersion};
    use wormhole::set::{Self, Set};
    use wormhole::version_control::{
        Self as control,
        NewEmitter as NewEmitterControl
    };

    friend wormhole::migrate;
    friend wormhole::publish_message;
    friend wormhole::set_fee;
    friend wormhole::setup;
    friend wormhole::transfer_fee;
    friend wormhole::update_guardian_set;
    friend wormhole::upgrade_contract;
    friend wormhole::vaa;

    const E_INVALID_UPGRADE_CAP_VERSION: u64 = 0;
    const E_ZERO_GUARDIANS: u64 = 1;
    const E_VAA_ALREADY_CONSUMED: u64 = 2;
    const E_BUILD_VERSION_MISMATCH: u64 = 3;

    /// Sui's chain ID is hard-coded to one value.
    const CHAIN_ID: u16 = 21;

    struct MigrationControl has store, drop, copy {}

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
        /// Currently in terms of Sui epochs until we have access to a clock
        /// with unix timestamp.
        guardian_set_epochs_to_live: u32,

        /// Consumed VAA hashes to protect against replay. VAAs relevant to
        /// Wormhole are just governance VAAs.
        consumed_vaa_hashes: Set<Bytes32>,

        /// Registry for new emitter caps (`EmitterCap`).
        emitter_registry: EmitterRegistry,

        /// Wormhole fee collector.
        fee_collector: FeeCollector,

        upgrade_cap: UpgradeCap,

        /// Contract upgrade tracker.
        required_version: RequiredVersion
    }

    public(friend) fun new(
        upgrade_cap: UpgradeCap,
        governance_chain: u16,
        governance_contract: vector<u8>,
        initial_guardians: vector<vector<u8>>,
        guardian_set_epochs_to_live: u32,
        message_fee: u64,
        ctx: &mut TxContext
    ): State {
        // Validate that the upgrade_cap equals the build version defined in
        // the `version_control` module.
        //
        // When the contract is first published and `State` is being created,
        // this is expected to be `1`.
        assert!(
            (
                control::version() == 1 &&
                package::version(&upgrade_cap) == control::version()
            ),
            E_INVALID_UPGRADE_CAP_VERSION
        );
        assert!(vector::length(&initial_guardians) > 0, E_ZERO_GUARDIANS);

        // First guardian set index is zero. New guardian sets must increment
        // from the last recorded index.
        let guardian_set_index = 0;

        let governance_contract =
            external_address::from_nonzero_bytes(
                governance_contract
            );
        let state = State {
            id: object::new(ctx),
            governance_chain,
            governance_contract,
            guardian_set_index,
            guardian_sets: table::new(ctx),
            guardian_set_epochs_to_live,
            consumed_vaa_hashes: set::new(ctx),
            emitter_registry: emitter::new_registry(),
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
        store_guardian_set(
            &mut state,
            guardian_set::new(guardian_set_index, guardians)
        );

        // Add dynamic field to control whether someone can call `migrate`. Set
        // this value to `false` by default.
        //
        // See `migrate` module for more info.
        field::add(&mut state.id, MigrationControl {}, false);

        let tracker = &mut state.required_version;
        required_version::add<control::NewEmitter>(tracker);
        required_version::add<control::ParseAndVerify>(tracker);
        required_version::add<control::PublishMessage>(tracker);
        required_version::add<control::SetFee>(tracker);
        required_version::add<control::TransferFee>(tracker);
        required_version::add<control::UpdateGuardianSet>(tracker);

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
        let policy = package::upgrade_policy(&self.upgrade_cap);
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
    ) {
        // Uptick the upgrade cap version number using this receipt.
        package::commit_upgrade(&mut self.upgrade_cap, receipt);

        // Check that the upticked hard-coded version version agrees with the
        // upticked version number.
        assert!(
            package::version(&self.upgrade_cap) == control::version() + 1,
            E_BUILD_VERSION_MISMATCH
        );

        // Update global version.
        required_version::update_latest(
            &mut self.required_version,
            &self.upgrade_cap
        );

        // Enable `migrate` to be called after commiting the upgrade.
        //
        // A separate method is required because `state` is a dependency of
        // `migrate`. This method warehouses state modifications required
        // for the new implementation plus enabling any methods required to be
        // gated by the current implementation version. In most cases `migrate`
        // is a no-op. But it still must be called in order to reset the
        // migration control to `false`.
        //
        // See `migrate` module for more info.
       enable_migration(self);
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

    /// Check whether `migrate` can be called.
    ///
    /// See `wormhole::migrate` module for more info.
    public fun can_migrate(self: &State): bool {
        *field::borrow(&self.id, MigrationControl {})
    }

    /// Allow `migrate` to be called after upgrade.
    ///
    /// See `wormhole::migrate` module for more info.
    public(friend) fun enable_migration(self: &mut State) {
        *field::borrow_mut(&mut self.id, MigrationControl {}) = true;
    }

    /// Disallow `migrate` to be called.
    ///
    /// See `wormhole::migrate` module for more info.
    public(friend) fun disable_migration(self: &mut State) {
        *field::borrow_mut(&mut self.id, MigrationControl {}) = false;
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
    /// epoch.
    ///
    /// TODO: Change this to be in terms of unix timestamp when `Clock` gets
    /// added to `vaa::parse_and_verify`.
    public fun guardian_set_epochs_to_live(self: &State): u32 {
        self.guardian_set_epochs_to_live
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
    public(friend) fun consume_vaa_hash(self: &mut State, vaa_hash: Bytes32) {
        let consumed = &mut self.consumed_vaa_hashes;
        assert!(!set::contains(consumed, vaa_hash), E_VAA_ALREADY_CONSUMED);
        set::add(consumed, vaa_hash);
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
    public(friend) fun expire_guardian_set(self: &mut State, ctx: &TxContext) {
        let expiring =
            table::borrow_mut(&mut self.guardian_sets, self.guardian_set_index);
        guardian_set::set_expiration(
            expiring,
            self.guardian_set_epochs_to_live,
            ctx
        );
    }

    /// Add the latest Guardian set from the governance action to update the
    /// current guardian set.
    ///
    /// See `wormhole::update_guardian_set` for more info.
    public(friend) fun store_guardian_set(
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

    /// Check whether a particular Guardian set is valid.
    ///
    /// See `wormhole::vaa` for more info.
    public fun is_guardian_set_active(
        self: &State,
        set: &GuardianSet,
        ctx: &TxContext
    ): bool {
        (
            self.guardian_set_index == guardian_set::index(set) ||
            guardian_set::is_active(set, ctx)
        )
    }

    /// Generate a new `EmitterCap`.
    ///
    /// NOTE: This method is guarded by a minimum build version check. This
    /// method could break backward compatibility on an upgrade.
    public fun new_emitter(self: &mut State, ctx: &mut TxContext): EmitterCap {
        check_minimum_requirement<NewEmitterControl>(self);

        emitter::new_cap(&mut self.emitter_registry, ctx)
    }

    public(friend) fun use_emitter_sequence(emitter_cap: &mut EmitterCap): u64 {
        emitter::use_sequence(emitter_cap)
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
        commit_upgrade(self, receipt)
    }
}
