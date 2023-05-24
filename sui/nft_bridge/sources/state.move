module nft_bridge::state {
    use sui::object::{Self, ID, UID};
    use sui::package::{UpgradeCap, UpgradeReceipt, UpgradeTicket};
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::consumed_vaas::{Self, ConsumedVAAs};
    use wormhole::emitter::{EmitterCap};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::package_utils::{Self};
    use wormhole::publish_message::{MessageTicket};

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

        /// TODO - Registry for native and wrapped assets.
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
        assert!(wormhole::emitter::sequence(&emitter_cap) == 0, E_USED_EMITTER);

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

        // Set first version and initialize package info. This will be used for
        // emitting information of successful migrations.
        let upgrade_cap = &state.upgrade_cap;
        package_utils::init_package_info(
            &mut state.id,
            version_control::current_version(),
            upgrade_cap
        );

        state
    }
}