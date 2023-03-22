/// Borrowed from `sui::package` for mocking up upgrade logic.
module wormhole::dummy_sui_package {
    use sui::object::{Self, ID, UID};
    use sui::tx_context::{TxContext};

    /// Tried to create a `Publisher` using a type that isn't a
    /// one-time witness.
    const ENotOneTimeWitness: u64 = 0;
    /// Tried to set a less restrictive policy than currently in place.
    const ETooPermissive: u64 = 1;
    /// This `UpgradeCap` has already authorized a pending upgrade.
    const EAlreadyAuthorized: u64 = 2;
    /// This `UpgradeCap` has not authorized an upgrade.
    const ENotAuthorized: u64 = 3;
    /// Trying to commit an upgrade to the wrong `UpgradeCap`.
    const EWrongUpgradeCap: u64 = 4;

    /// Update any part of the package (function implementations, add new
    /// functions or types, change dependencies)
    const COMPATIBLE: u8 = 0;
    /// Add new functions or types, or change dependencies, existing
    /// functions can't change.
    const ADDITIVE: u8 = 1;
    /// Only be able to change dependencies.
    const DEP_ONLY: u8 = 2;

    /// Capability controlling the ability to upgrade a package.
    struct UpgradeCap has key, store {
        id: UID,
        /// (Mutable) ID of the package that can be upgraded.
        package: ID,
        /// (Mutable) The number of upgrades that have been applied
        /// successively to the original package.  Initially 0.
        version: u64,
        /// What kind of upgrades are allowed.
        policy: u8,
    }

    /// Permission to perform a particular upgrade (for a fixed version of
    /// the package, bytecode to upgrade with and transitive dependencies to
    /// depend against).
    ///
    /// An `UpgradeCap` can only issue one ticket at a time, to prevent races
    /// between concurrent updates or a change in its upgrade policy after
    /// issuing a ticket, so the ticket is a "Hot Potato" to preserve forward
    /// progress.
    struct UpgradeTicket {
        /// (Immutable) ID of the `UpgradeCap` this originated from.
        cap: ID,
        /// (Immutable) ID of the package that can be upgraded.
        package: ID,
        /// (Immutable) The policy regarding what kind of upgrade this ticket
        /// permits.
        policy: u8,
        /// (Immutable) SHA256 digest of the bytecode and transitive
        /// dependencies that will be used in the upgrade.
        digest: vector<u8>,
    }

    /// Issued as a result of a successful upgrade, containing the
    /// information to be used to update the `UpgradeCap`.  This is a "Hot
    /// Potato" to ensure that it is used to update its `UpgradeCap` before
    /// the end of the transaction that performed the upgrade.
    struct UpgradeReceipt {
        /// (Immutable) ID of the `UpgradeCap` this originated from.
        cap: ID,
        /// (Immutable) ID of the package after it was upgraded.
        package: ID,
    }

    /// Issue a ticket authorizing an upgrade to a particular new bytecode
    /// (identified by its digest).  A ticket will only be issued if one has
    /// not already been issued, and if the `policy` requested is at least as
    /// restrictive as the policy set out by the `cap`.
    ///
    /// The `digest` supplied and the `policy` will both be checked by
    /// validators when running the upgrade.  I.e. the bytecode supplied in
    /// the upgrade must have a matching digest, and the changes relative to
    /// the parent package must be compatible with the policy in the ticket
    /// for the upgrade to succeed.
    public fun authorize_upgrade(
        cap: &mut UpgradeCap,
        policy: u8,
        digest: vector<u8>
    ): UpgradeTicket {
        let id_zero = object::id_from_address(@0x0);
        assert!(cap.package != id_zero, EAlreadyAuthorized);
        assert!(policy >= cap.policy, ETooPermissive);

        let package = cap.package;
        cap.package = id_zero;

        UpgradeTicket {
            cap: object::id(cap),
            package,
            policy: cap.policy,
            digest,
        }
    }

    /// The most recent version of the package, increments by one for each
    /// successfully applied upgrade.
    public fun version(cap: &UpgradeCap): u64 {
        cap.version
    }

    /// The most permissive kind of upgrade currently supported by this
    /// `cap`.
    public fun upgrade_policy(cap: &UpgradeCap): u8 {
        cap.policy
    }

    /// Discard the `UpgradeCap` to make a package immutable.
    public entry fun make_immutable(cap: UpgradeCap) {
        let UpgradeCap { id, package: _, version: _, policy: _ } = cap;
        object::delete(id);
    }

    /// Consume an `UpgradeReceipt` to update its `UpgradeCap`, finalizing
    /// the upgrade.
    public fun commit_upgrade(
        cap: &mut UpgradeCap,
        receipt: UpgradeReceipt,
    ) {
        let UpgradeReceipt { cap: cap_id, package } = receipt;

        assert!(object::id(cap) == cap_id, EWrongUpgradeCap);
        assert!(object::id_to_address(&cap.package) == @0x0, ENotAuthorized);

        cap.package = package;
        cap.version = cap.version + 1;
    }

    public fun mock_new_upgrade_cap(
        package: ID, 
        ctx: &mut TxContext
    ): UpgradeCap {
        UpgradeCap {
            id: object::new(ctx),
            package,
            version: 1,
            policy: COMPATIBLE,
        }
    }

    #[test_only]
    /// Test-only function to simulate publishing a package at address
    /// `ID`, to create an `UpgradeCap`.
    public fun test_publish(package: ID, ctx: &mut TxContext): UpgradeCap {
        UpgradeCap {
            id: object::new(ctx),
            package,
            version: 1,
            policy: COMPATIBLE,
        }
    }

    #[test_only]
    /// Test-only function that takes the role of the actual `Upgrade`
    /// command, converting the ticket for the pending upgrade to a
    /// receipt for a completed upgrade.
    public fun test_upgrade(ticket: UpgradeTicket): UpgradeReceipt {
        use std::vector::{Self};

        let UpgradeTicket { cap, package, policy: _, digest: _ } = ticket;

        // Generate a fake package ID for the upgraded package by
        // hashing the existing package and cap ID.
        let data = object::id_to_bytes(&cap);
        std::vector::append(&mut data, object::id_to_bytes(&package));

        let hash = std::hash::sha3_256(data);
        vector::reverse(&mut hash);
        let i = 0;
        while (i < 12) {
            vector::pop_back(&mut hash);
            i = i + 1;
        };
        vector::reverse(&mut hash);
        let package = object::id_from_bytes(hash);

        UpgradeReceipt {
            cap, package
        }
    }
}
