// SPDX-License-Identifier: Apache 2

/// This module implements the global state variables for Token Bridge as a
/// shared object. The `State` object is used to perform anything that requires
/// access to data that defines the Token Bridge contract. Examples of which are
/// accessing registered assets and verifying `VAA` intended for Token Bridge by
/// checking the emitter against its own registered emitters.
module token_bridge::state {
    use std::option::{Self, Option};
    use sui::clock::{Clock};
    use sui::coin::{Coin};
    use sui::dynamic_field::{Self as field};
    use sui::object::{Self, ID, UID};
    use sui::package::{Self, UpgradeCap, UpgradeReceipt, UpgradeTicket};
    use sui::sui::{SUI};
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::consumed_vaas::{Self, ConsumedVAAs};
    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::required_version::{Self, RequiredVersion};
    use wormhole::state::{State as WormholeState};
    use wormhole::vaa::{Self, VAA};

    use token_bridge::token_registry::{Self, TokenRegistry, VerifiedAsset};
    use token_bridge::version_control::{Self as control};

    /// For a given chain ID, Token Bridge is non-existent.
    const E_UNREGISTERED_EMITTER: u64 = 0;
    /// Cannot register chain ID == 0.
    const E_INVALID_EMITTER_CHAIN: u64 = 1;
    /// Emitter already exists for a given chain ID.
    const E_EMITTER_ALREADY_REGISTERED: u64 = 2;
    /// Encoded emitter address does not match registered Token Bridge.
    const E_EMITTER_ADDRESS_MISMATCH: u64 = 3;
    /// VAA hash already exists in `Set`.
    const E_VAA_ALREADY_CONSUMED: u64 = 4;
    /// Build does not agree with expected upgrade.
    const E_BUILD_VERSION_MISMATCH: u64 = 5;

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

    /// Used as key for dynamic field reflecting whether `migrate` can be
    /// called.
    ///
    /// See `migrate` module for more info.
    struct MigrationControl has store, drop, copy {}

    /// Container for all state variables for Token Bridge.
    struct State has key, store {
        id: UID,

        /// Set of consumed VAA hashes.
        consumed_vaas: ConsumedVAAs,

        /// Emitter capability required to publish Wormhole messages.
        emitter_cap: EmitterCap,

        /// Registry for foreign Token Bridge contracts.
        emitter_registry: Table<u16, ExternalAddress>,

        /// Registry for native and wrapped assets.
        token_registry: TokenRegistry,

        /// Upgrade capability.
        upgrade_cap: UpgradeCap,

        /// Contract build version tracker.
        required_version: RequiredVersion
    }

    /// Create new `State`. This is only executed using the `setup` module.
    public(friend) fun new(
        worm_state: &WormholeState,
        upgrade_cap: UpgradeCap,
        ctx: &mut TxContext
    ): State {
        let state = State {
            id: object::new(ctx),
            consumed_vaas: consumed_vaas::new(ctx),
            emitter_cap: emitter::new(worm_state, ctx),
            emitter_registry: table::new(ctx),
            token_registry: token_registry::new(ctx),
            upgrade_cap,
            required_version: required_version::new(control::version(), ctx)
        };

        // Add dynamic field to control whether someone can call `migrate`. Set
        // this value to `false` by default.
        //
        // See `migrate` module for more info.
        field::add(&mut state.id, MigrationControl {}, false);

        let tracker = &mut state.required_version;
        required_version::add<control::AttestToken>(tracker);
        required_version::add<control::CompleteTransfer>(tracker);
        required_version::add<control::CompleteTransferWithPayload>(tracker);
        required_version::add<control::CreateWrapped>(tracker);
        required_version::add<control::RegisterChain>(tracker);
        required_version::add<control::TransferTokens>(tracker);
        required_version::add<control::TransferTokensWithPayload>(tracker);
        required_version::add<control::Vaa>(tracker);

        state
    }

    /// Retrieve governance module name.
    public fun governance_module(): Bytes32 {
        // A.K.A. "TokenBridge".
        bytes32::new(
            x"000000000000000000000000000000000000000000546f6b656e427269646765"
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

        // TODO: grab package ID from `UpgradeCap` and store it
        // in a dynamic field. This will be the old ID after the upgrade.
        // Both IDs will be emitted in a `ContractUpgraded` event.
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
    /// the latest Token Bridge implementation.
    public(friend) fun check_minimum_requirement<ControlType>(self: &State) {
        check_minimum_requirement_specified<ControlType>(
            self,
            control::version()
        )
    }

    /// Check whether a particular method meets the minimum build version for
    /// a specified build version checked outside of this module.
    ///
    /// See `create_wrapped` module for an example of how this is used.
    public(friend) fun check_minimum_requirement_specified<ControlType>(
        self: &State,
        build_version: u64
    ) {
        required_version::check_minimum_requirement<ControlType>(
            &self.required_version,
            build_version
        )
    }

    /// Check whether `migrate` can be called.
    ///
    /// See `token_bridge::migrate` module for more info.
    public fun can_migrate(self: &State): bool {
        *field::borrow(&self.id, MigrationControl {})
    }

    /// Allow `migrate` to be called after upgrade.
    ///
    /// See `token_bridge::migrate` module for more info.
    public(friend) fun enable_migration(self: &mut State) {
        *field::borrow_mut(&mut self.id, MigrationControl {}) = true;
    }

    /// Disallow `migrate` to be called.
    ///
    /// See `token_bridge::migrate` module for more info.
    public(friend) fun disable_migration(self: &mut State) {
        *field::borrow_mut(&mut self.id, MigrationControl {}) = false;
    }

    /// Publish Wormhole message using Token Bridge's `EmitterCap`.
    public(friend) fun publish_wormhole_message(
        self: &mut State,
        worm_state: &mut WormholeState,
        nonce: u32,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
        the_clock: &Clock
    ): u64 {
        use wormhole::publish_message::{publish_message};

        publish_message(
            worm_state,
            &mut self.emitter_cap,
            nonce,
            payload,
            message_fee,
            the_clock
        )
    }

    /// Retrieve immutable reference to `TokenRegistry`.
    public fun borrow_token_registry(self: &State): &TokenRegistry {
        &self.token_registry
    }

    /// Retrieve mutable reference to `TokenRegistry`.
    public(friend) fun borrow_mut_token_registry(
        self: &mut State
    ): &mut TokenRegistry {
        &mut self.token_registry
    }

    #[test_only]
    public fun borrow_mut_token_registry_test_only(
        self: &mut State
    ): &mut TokenRegistry {
        borrow_mut_token_registry(self)
    }

    /// Retrieve mutable reference to `ConsumedVAAs`.
    public(friend) fun borrow_mut_consumed_vaas(
        self: &mut State
    ): &mut ConsumedVAAs {
        &mut self.consumed_vaas
    }

    /// Assert that a given emitter equals one that is registered as a foreign
    /// Token Bridge.
    public fun assert_registered_emitter(self: &State, parsed: &VAA) {
        let chain = vaa::emitter_chain(parsed);
        let registry = &self.emitter_registry;
        assert!(table::contains(registry, chain), E_UNREGISTERED_EMITTER);

        let registered = table::borrow(registry, chain);
        let emitter_addr = vaa::emitter_address(parsed);
        assert!(*registered == emitter_addr, E_EMITTER_ADDRESS_MISMATCH);
    }

    /// Add a new Token Bridge emitter to the registry. This method will abort
    /// if an emitter is already registered for a particular chain ID.
    ///
    /// See `register_chain` module for more info.
    public(friend) fun register_new_emitter(
        self: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        assert!(chain != 0, E_INVALID_EMITTER_CHAIN);

        let registry = &mut self.emitter_registry;
        assert!(
            !table::contains(registry, chain),
            E_EMITTER_ALREADY_REGISTERED
        );
        table::add(registry, chain, contract_address);
    }

    #[test_only]
    public fun register_new_emitter_test_only(
        self: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        register_new_emitter(self, chain, contract_address);
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
    public fun borrow_emitter_registry(
        self: &State
    ): &Table<u16, ExternalAddress> {
        &self.emitter_registry
    }
}
