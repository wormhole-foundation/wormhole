module token_bridge::state {
    use sui::balance::{Balance};
    use sui::clock::{Clock};
    use sui::dynamic_field::{Self as field};
    use sui::event::{Self};
    use sui::object::{Self, ID, UID};
    use sui::package::{Self, UpgradeCap, UpgradeReceipt, UpgradeTicket};
    use sui::sui::{SUI};
    use sui::tx_context::{TxContext};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::emitter::{Self, EmitterCap};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::required_version::{Self, RequiredVersion};
    use wormhole::set::{Self, Set};
    use wormhole::state::{State as WormholeState};

    use token_bridge::emitter_registry::{Self, EmitterRegistry};
    use token_bridge::token_registry::{Self, TokenRegistry};
    use token_bridge::version_control::{Self as control};

    const E_UNREGISTERED_EMITTER: u64 = 0;
    const E_EMITTER_ALREADY_REGISTERED: u64 = 1;
    const E_VAA_ALREADY_CONSUMED: u64 = 2;
    const E_BUILD_VERSION_MISMATCH: u64 = 3;
    const E_INVALID_UPGRADE_CAP: u64 = 4;

    friend token_bridge::attest_token;
    friend token_bridge::complete_transfer;
    friend token_bridge::complete_transfer_with_payload;
    friend token_bridge::create_wrapped;
    friend token_bridge::migrate;
    friend token_bridge::register_chain;
    friend token_bridge::setup;
    friend token_bridge::transfer_tokens;
    friend token_bridge::transfer_tokens_with_payload;
    friend token_bridge::vaa;

    // Event reflecting package upgrade.
    struct ContractUpgraded has drop, copy {
        old_contract: ID,
        new_contract: ID
    }

    /// Used as key for dynamic field reflecting whether `migrate` can be
    /// called.
    ///
    /// See `migrate` module for more info.
    struct MigrationControl has store, drop, copy {}

    /// Treasury caps, token stores, consumed VAAs, registered emitters, etc.
    /// are stored as dynamic fields of bridge state.
    struct State has key, store {
        id: UID,

        /// Set of consumed VAA hashes.
        consumed_vaa_hashes: Set<Bytes32>,

        /// Token bridge owned emitter capability.
        emitter_cap: EmitterCap,

        /// Registery for foreign Token Bridge contracts.
        emitter_registry: EmitterRegistry,

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
        // Verify that this `UpgradeCap` belongs to the Token Bridge package and
        // equals the build version defined in the `version_control` module.
        //
        // When the contract is first published and `State` is being created,
        // this is expected to be `1`.
        let package_id = package::upgrade_package(&upgrade_cap);
        assert!(
            (
                package_id == object::id_from_address(@token_bridge) &&
                control::version() == 1 &&
                package::version(&upgrade_cap) == control::version()
            ),
            E_INVALID_UPGRADE_CAP
        );

        let state = State {
            id: object::new(ctx),
            consumed_vaa_hashes: set::new(ctx),
            emitter_cap: emitter::new(worm_state, ctx),
            emitter_registry: emitter_registry::new(ctx),
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

        // Emit an event reflecting package ID change.
        event::emit(
            ContractUpgraded {
                old_contract: object::id_from_address(@token_bridge),
                new_contract: package::upgrade_package(&self.upgrade_cap)
            }
        );
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
        message_fee: Balance<SUI>,
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
    public(friend) fun borrow_token_registry_mut(
        self: &mut State
    ): &mut TokenRegistry {
        &mut self.token_registry
    }

    #[test_only]
    public fun borrow_token_registry_mut_test_only(
        self: &mut State
    ): &mut TokenRegistry {
        borrow_token_registry_mut(self)
    }

    /// For a deserialized VAA, consume its hash so this VAA cannot be redeemed
    /// again. This protects against replay attacks.
    public(friend) fun consume_vaa_hash(self: &mut State, vaa_hash: Bytes32) {
        let consumed = &mut self.consumed_vaa_hashes;
        assert!(!set::contains(consumed, vaa_hash), E_VAA_ALREADY_CONSUMED);
        set::add(consumed, vaa_hash);
    }

    public fun registered_emitter(
        self: &State,
        chain: u16
    ): ExternalAddress {
        emitter_registry::emitter_address(&self.emitter_registry, chain)
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
        emitter_registry::add(
            &mut self.emitter_registry,
            chain,
            contract_address
        );
    }

    #[test_only]
    public fun register_new_emitter_test_only(
        self: &mut State,
        chain: u16,
        contract_address: ExternalAddress
    ) {
        register_new_emitter(self, chain, contract_address);
    }

    /// Retrieve decimals from for a given coin type in `TokenRegistry`.
    public fun coin_decimals<CoinType>(self: &State): u8 {
        let registry = borrow_token_registry(self);
        let cap = token_registry::new_asset_cap<CoinType>(registry);
        token_registry::checked_decimals(&cap, registry)
    }
}
