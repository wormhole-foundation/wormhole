// SPDX-License-Identifier: Apache 2

/// This module implements the mechanism to publish the Token Bridge contract
/// and initialize `State` as a shared object.
module token_bridge::setup {
    use sui::object::{Self, UID};
    use sui::package::{Self, UpgradeCap};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};
    use wormhole::state::{State as WormholeState};

    use token_bridge::state::{Self};

    /// `UpgradeCap` is not as expected when initializing `State`.
    const E_INVALID_UPGRADE_CAP: u64 = 0;
    /// Build version for setup must only be `1`.
    const E_INVALID_BUILD_VERSION: u64 = 1;

    /// Capability created at `init`, which will be destroyed once
    /// `init_and_share_state` is called. This ensures only the deployer can
    /// create the shared `State`.
    struct DeployerCap has key, store {
        id: UID
    }

    /// Called automatically when module is first published. Transfers
    /// `DeployerCap` to sender.
    ///
    /// Only `setup::init_and_share_state` requires `DeployerCap`.
    fun init(ctx: &mut TxContext) {
        let deployer = DeployerCap { id: object::new(ctx) };
        transfer::transfer(deployer, tx_context::sender(ctx));
    }

    #[test_only]
    public fun init_test_only(ctx: &mut TxContext) {
        // NOTE: This exists to mock up sui::package for proposed upgrades.
        use sui::package::{Self};

        init(ctx);

        // This will be created and sent to the transaction sender
        // automatically when the contract is published.
        transfer::public_transfer(
            package::test_publish(object::id_from_address(@token_bridge), ctx),
            tx_context::sender(ctx)
        );
    }

    /// Only the owner of the `DeployerCap` can call this method. This
    /// method destroys the capability and shares the `State` object.
    public fun complete(
        worm_state: &WormholeState,
        deployer: DeployerCap,
        upgrade_cap: UpgradeCap,
        ctx: &mut TxContext
    ) {
        let version = token_bridge::version_control::version();
        assert!(version == 1, E_INVALID_BUILD_VERSION);

        wormhole::package_utils::assert_package_upgrade_cap<DeployerCap>(
            &upgrade_cap,
            package::compatible_policy(),
            version
        );

        // Destroy deployer cap.
        let DeployerCap { id } = deployer;
        object::delete(id);

        // Share new state.
        transfer::public_share_object(state::new(worm_state, upgrade_cap, ctx));
    }
}
