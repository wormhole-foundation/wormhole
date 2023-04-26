// SPDX-License-Identifier: Apache 2

/// This module implements the global state variables for Token Bridge as a
/// shared object. The `State` object is used to perform anything that requires
/// access to data that defines the Token Bridge contract. Examples of which are
/// accessing registered assets and verifying `VAA` intended for Token Bridge by
/// checking the emitter against its own registered emitters.
module token_bridge::state {
    use std::option::{Self, Option};
    use sui::dynamic_field::{Self as field};
    use sui::object::{Self, ID, UID};
    use sui::package::{Self, UpgradeCap, UpgradeReceipt, UpgradeTicket};
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::consumed_vaas::{Self, ConsumedVAAs};
    use wormhole::emitter::{EmitterCap};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::package_utils::{Self};
    use wormhole::publish_message::{MessageTicket};

    use token_bridge::token_registry::{Self, TokenRegistry, VerifiedAsset};
    use token_bridge::version_control::{Self};

    /// Build digest does not agree with current implementation.
    const E_INVALID_BUILD_DIGEST: u64 = 0;
    /// Specified version does not match this build's version.
    const E_VERSION_MISMATCH: u64 = 1;

    friend token_bridge::attest_token;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::create_wrapped;
    friend token_bridge::migrate;
    friend token_bridge::register_chain;
    friend token_bridge::setup;
    friend token_bridge::transfer_tokens;
    friend token_bridge::transfer_tokens_with_payload;
    friend token_bridge::upgrade_contract;
    friend token_bridge::vaa;

    /// TODO: write something meaningful here
    struct CurrentDigest has store, drop, copy {}

    /// Capability reflecting that the current build version is used to invoke
    /// state methods.
    struct LatestOnly has drop {}

    /// Container for all state variables for Token Bridge.
    struct State has key, store {
        id: UID,

        /// Governance chain ID.
        governance_chain: u16,

        /// Governance contract address.
        governance_contract: ExternalAddress,

        /// Set of consumed VAA hashes.
        consumed_vaas: ConsumedVAAs,

        /// Emitter capability required to publish Wormhole messages.
        emitter_cap: EmitterCap,

        /// Registry for foreign Token Bridge contracts.
        emitter_registry: Table<u16, ExternalAddress>,

        /// Registry for native and wrapped assets.
        token_registry: TokenRegistry,

        /// Upgrade capability.
        upgrade_cap: UpgradeCap
    }

    /// Create new `State`. This is only executed using the `setup` module.
    public(friend) fun new(
        emitter_cap: EmitterCap,
        upgrade_cap: UpgradeCap,
        governance_chain: u16,
        governance_contract: ExternalAddress,
        ctx: &mut TxContext
    ): State {
        let state = State {
            id: object::new(ctx),
            governance_chain,
            governance_contract,
            consumed_vaas: consumed_vaas::new(ctx),
            emitter_cap,
            emitter_registry: table::new(ctx),
            token_registry: token_registry::new(ctx),
            upgrade_cap
        };

        // Set first version for this package.
        package_utils::init_version(
            &mut state.id,
            version_control::current_version()
        );

        // Add dummy hash since this is the first time the package is published.
        field::add(&mut state.id, CurrentDigest {}, bytes32::default());

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

    /// Retrieve governance module name.
    public fun governance_module(): Bytes32 {
        // A.K.A. "TokenBridge".
        bytes32::new(
            x"000000000000000000000000000000000000000000546f6b656e427269646765"
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

    /// Retrieve immutable reference to `TokenRegistry`.
    public fun borrow_token_registry(
        self: &State
    ): &TokenRegistry {
        &self.token_registry
    }

    public fun borrow_emitter_registry(
        self: &State
    ): &Table<u16, ExternalAddress> {
        &self.emitter_registry
    }

    public fun maybe_verified_asset<CoinType>(
        self: &State
    ): Option<VerifiedAsset<CoinType>> {
        let registry = &self.token_registry;
        if (token_registry::has<CoinType>(registry)) {
            option::some(token_registry::verified_asset<CoinType>(registry))
        } else {
            option::none()
        }
    }

    public fun verified_asset<CoinType>(
        self: &State
    ): VerifiedAsset<CoinType> {
        token_registry::assert_has<CoinType>(&self.token_registry);
        token_registry::verified_asset(&self.token_registry)
    }

    /// Retrieve decimals from for a given coin type in `TokenRegistry`.
    public fun coin_decimals<CoinType>(self: &State): u8 {
        token_registry::coin_decimals(&verified_asset<CoinType>(self))
    }

    #[test_only]
    public fun borrow_mut_token_registry_test_only(
        self: &mut State
    ): &mut TokenRegistry {
        borrow_mut_token_registry(&cache_latest_only(self), self)
    }

    #[test_only]
    public fun migrate_version_test_only<Old: store + drop, New: store + drop>(
        self: &mut State,
        old_version: Old,
        new_version: New
    ) {
        wormhole::package_utils::update_version_type<Old, New>(
            &mut self.id,
            old_version,
            new_version
        );
    }

    #[test_only]
    public fun test_upgrade(self: &mut State) {
        let test_digest = bytes32::from_bytes(b"new build");
        let ticket = authorize_upgrade(self, test_digest);
        let receipt = package::test_upgrade(ticket);
        commit_upgrade(self, receipt);
    }

    ////////////////////////////////////////////////////////////////////////////
    //
    //  Privileged `State` Access
    //
    //  This section of methods require a `LatestOnly`, which can only be
    //  created within the Token Bridge package. This capability allows special
    //  access to the `State` object where we require that the latest build is
    //  used for these interactions.
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
    public(friend) fun cache_latest_only(self: &State): LatestOnly {
        package_utils::assert_version(
            &self.id,
            version_control::current_version()
        );

        LatestOnly {}
    }

    /// Obtain a capability to interact with `State` methods. This method checks
    /// that we are running the current build and that the specified `Version`
    /// equals the current version. This method is useful when external modules
    /// invoke Token Bridge and we need to check that the external module's
    /// version is up-to-date (e.g. `create_wrapped::prepare_registration`).
    ///
    /// NOTE: This method allows caching the current version check so we avoid
    /// multiple checks to dynamic fields.
    public(friend) fun cache_latest_only_specified<Version>(
        self: &State
    ): LatestOnly {
        use std::type_name::{get};

        // Explicitly check the type names.
        let current_type =
            package_utils::type_of_version(version_control::current_version());
        assert!(current_type == get<Version>(), E_VERSION_MISMATCH);

        cache_latest_only(self)
    }

    /// A more expressive method to enforce that the current build version is
    /// used.
    public(friend) fun assert_latest_only(self: &State) {
        cache_latest_only(self);
    }

    /// Store `VAA` hash as a way to claim a VAA. This method prevents a VAA
    /// from being replayed.
    public(friend) fun borrow_mut_consumed_vaas(
        _: &LatestOnly,
        self: &mut State
    ): &mut ConsumedVAAs {
        borrow_mut_consumed_vaas_unchecked(self)
    }

    /// Store `VAA` hash as a way to claim a VAA. This method prevents a VAA
    /// from being replayed.
    ///
    /// NOTE: This method does not require `LatestOnly`. Only methods in the
    /// `upgrade_contract` module requires this to be unprotected to prevent
    /// a corrupted upgraded contract from bricking upgradability.
    public(friend) fun borrow_mut_consumed_vaas_unchecked(
        self: &mut State
    ): &mut ConsumedVAAs {
        &mut self.consumed_vaas
    }

    /// Publish Wormhole message using Token Bridge's `EmitterCap`.
    public(friend) fun prepare_wormhole_message(
        _: &LatestOnly,
        self: &mut State,
        nonce: u32,
        payload: vector<u8>
    ): MessageTicket {
        wormhole::publish_message::prepare_message(
            &mut self.emitter_cap,
            nonce,
            payload,
        )
    }

    /// Retrieve mutable reference to `TokenRegistry`.
    public(friend) fun borrow_mut_token_registry(
        _: &LatestOnly,
        self: &mut State
    ): &mut TokenRegistry {
        &mut self.token_registry
    }

    public(friend) fun borrow_mut_emitter_registry(
        _: &LatestOnly,
        self: &mut State
    ): &mut Table<u16, ExternalAddress> {
        &mut self.emitter_registry
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
        implementation_digest: Bytes32
    ): UpgradeTicket {
        // Save current package ID before committing upgrade.
        field::add(
            &mut self.id,
            b"current_package_id",
            package::upgrade_package(&self.upgrade_cap)
        );

        let policy = package::upgrade_policy(&self.upgrade_cap);

        // Manage saving the current digest.
        let _: Bytes32 = field::remove(&mut self.id, CurrentDigest {});
        field::add(&mut self.id, CurrentDigest {}, implementation_digest);

        // Finally authorize upgrade.
        package::authorize_upgrade(
            &mut self.upgrade_cap,
            policy,
            bytes32::to_bytes(implementation_digest),
        )
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
        // Uptick the upgrade cap version number using this receipt.
        package::commit_upgrade(&mut self.upgrade_cap, receipt);

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

        // Return the package IDs.
        (
            field::remove(&mut self.id, b"current_package_id"),
            package::upgrade_package(&self.upgrade_cap)
        )
    }

    public(friend) fun migrate_version(self: &mut State) {
        package_utils::update_version_type(
            &mut self.id,
            version_control::previous_version(),
            version_control::current_version()
        );
    }

    public(friend) fun assert_current_digest(
        _: &LatestOnly,
        self: &State,
        digest: Bytes32
    ) {
        let current = *field::borrow(&self.id, CurrentDigest {});
        assert!(digest == current, E_INVALID_BUILD_DIGEST);
    }

    #[test_only]
    public fun reverse_migrate_version(self: &mut State) {
        package_utils::update_version_type(
            &mut self.id,
            version_control::current_version(),
            version_control::dummy()
        );
    }
}
