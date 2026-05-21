// SPDX-License-Identifier: Apache 2

/// This module implements the mechanism to publish the Token Bridge contract
/// and initialize `State` as a shared object.
module token_bridge::setup {
    use sui::object::{Self, UID};
    use sui::package::{Self, UpgradeCap};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};
    use wormhole::emitter::{EmitterCap};

    use token_bridge::state::{Self};

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
    /// Dummy package address for testing.
    ///
    /// With the new Move package management system (Sui 1.63+), packages no
    /// longer have explicit addresses in Move.toml. Instead, the package
    /// address is 0x0 at compile time and gets assigned at publish time.
    ///
    /// This causes issues in tests because:
    /// 1. `sui::package::test_publish` creates an UpgradeCap with the given
    ///    package ID
    /// 2. If we pass 0x0, `authorize_upgrade` fails with `EAlreadyAuthorized`
    ///    because it checks `cap.package != 0x0`
    /// 3. We must use a non-zero dummy address for the test UpgradeCap
    ///
    /// We also need `complete_test_only` because `assert_package_upgrade_cap`
    /// derives the expected package address from a type (which is 0x0 at test
    /// time), but our UpgradeCap has this non-zero test address.
    const TEST_PACKAGE_ADDR: address = @0x200;

    #[test_only]
    public fun init_test_only(ctx: &mut TxContext) {
        // NOTE: This exists to mock up sui::package for proposed upgrades.
        use sui::package::{Self};

        init(ctx);

        // This will be created and sent to the transaction sender
        // automatically when the contract is published.
        transfer::public_transfer(
            package::test_publish(object::id_from_address(TEST_PACKAGE_ADDR), ctx),
            tx_context::sender(ctx)
        );
    }

    #[allow(lint(share_owned))]
    /// Only the owner of the `DeployerCap` can call this method. This
    /// method destroys the capability and shares the `State` object.
    public fun complete(
        deployer: DeployerCap,
        upgrade_cap: UpgradeCap,
        emitter_cap: EmitterCap,
        governance_chain: u16,
        governance_contract: vector<u8>,
        ctx: &mut TxContext
    ) {
        wormhole::package_utils::assert_package_upgrade_cap<DeployerCap>(
            &upgrade_cap,
            package::compatible_policy(),
            1
        );

        // Destroy deployer cap.
        let DeployerCap { id } = deployer;
        object::delete(id);

        // Share new state.
        transfer::public_share_object(
            state::new(
                emitter_cap,
                upgrade_cap,
                governance_chain,
                wormhole::external_address::new_nonzero(
                    wormhole::bytes32::from_bytes(governance_contract)
                ),
                ctx
            ));
    }

    #[test_only]
    #[allow(lint(share_owned))]
    /// Test-only version of `complete` that skips the UpgradeCap package check.
    public fun complete_test_only(
        deployer: DeployerCap,
        upgrade_cap: UpgradeCap,
        emitter_cap: EmitterCap,
        governance_chain: u16,
        governance_contract: vector<u8>,
        ctx: &mut TxContext
    ) {
        // Skip assert_package_upgrade_cap check for tests.

        // Destroy deployer cap.
        let DeployerCap { id } = deployer;
        object::delete(id);

        // Share new state.
        transfer::public_share_object(
            state::new(
                emitter_cap,
                upgrade_cap,
                governance_chain,
                wormhole::external_address::new_nonzero(
                    wormhole::bytes32::from_bytes(governance_contract)
                ),
                ctx
            ));
    }
}
