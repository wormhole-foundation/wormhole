// SPDX-License-Identifier: Apache 2

/// This module implements a mechanism for version control. While keeping track
/// of the latest version of a package build, `RequiredVersion` manages the
/// minimum required version number for any method in that package. For any
/// upgrade where a particular method can have backward compatibility, the
/// minimum version would not have to change (because the method should work the
/// same way with the previous version or current version).
///
/// If there happens to be a breaking change for a particular method, this
/// module can force that the method's minimum requirement be the latest build.
/// If a previous build were used, the method would abort if a check is in place
/// with `RequiredVersion`.
///
/// There is no magic behind the way ths module works. `RequiredVersion` is
/// intended to live in a package's shared object that gets passed into its
/// methods (e.g. Wormhole's `State` object).
module wormhole::required_version {
    use sui::dynamic_field::{Self as field};
    use sui::object::{Self, UID};
    use sui::tx_context::{TxContext};

    // NOTE: This exists to mock up sui::package for proposed upgrades.
    use wormhole::dummy_sui_package::{Self as package, UpgradeCap};

    /// Build version passed does not meet method's minimum required version.
    const E_OUTDATED_VERSION: u64 = 0;

    /// Container to keep track of latest build version. Dynamic fields are
    /// associated with its `id`.
    struct RequiredVersion has store {
        id: UID,
        latest_version: u64
    }

    struct Key<phantom MethodType> has store, drop, copy {}

    /// Create new `RequiredVersion` with a configured starting version.
    public fun new(version: u64, ctx: &mut TxContext): RequiredVersion {
        RequiredVersion {
            id: object::new(ctx),
            latest_version: version
        }
    }

    /// Retrieve latest build version.
    public fun current(self: &RequiredVersion): u64 {
        self.latest_version
    }

    /// Add specific method handling via custom `MethodType`. At the time a
    /// method is added, the minimum build version associated with this method
    /// by default is the latest version.
    public fun add<MethodType>(self: &mut RequiredVersion) {
        field::add(&mut self.id, Key<MethodType> {}, self.latest_version)
    }

    /// This method will abort if the version for a particular `MethodType` is
    /// not up-to-date with the version of the current build.
    ///
    /// For example, if the minimum requirement for `foobar` module (with an
    /// appropriately named `MethodType` like `FooBar`) is `1` and the current
    /// implementation version is `2`, this method will succeed because the
    /// build meets the minimum required version of `1` in order for `foobar` to
    /// work. So if someone were to use an older build like version `1`, this
    /// method will succeed.
    ///
    /// But if `check_minimum_requirement` were invoked for `foobar` when the
    /// minimum requirement is `2` and the current build is only version `1`,
    /// then this method will abort because the build does not meet the minimum
    /// version requirement for `foobar`.
    ///
    /// This method also assumes that the `MethodType` being checked for is
    /// already a dynamic field (using `add`) during initialization.
    public fun check_minimum_requirement<MethodType>(
        self: &RequiredVersion,
        build_version: u64
    ) {
        assert!(
            build_version >= minimum_for<MethodType>(self),
            E_OUTDATED_VERSION
        );
    }

    /// At `commit_upgrade`, use this method to update the tracker's knowledge
    /// of the latest upgrade (build) version, which is obtained from the
    /// `UpgradeCap` in `sui::package`.
    public fun update_latest(
        self: &mut RequiredVersion,
        upgrade_cap: &UpgradeCap
    ) {
        self.latest_version = package::version(upgrade_cap);
    }

    /// Once the global version is updated via `commit_upgrade` and there is a
    /// particular method that has a breaking change, use this method to uptick
    /// that method's minimum required version to the latest.
    public fun require_current_version<MethodType>(self: &mut RequiredVersion) {
        let min_version = field::borrow_mut(&mut self.id, Key<MethodType> {});
        *min_version = self.latest_version;
    }

    /// Retrieve the minimum required version for a particular method (via
    /// `MethodType`).
    public fun minimum_for<MethodType>(self: &RequiredVersion): u64 {
        *field::borrow(&self.id, Key<MethodType> {})
    }

    #[test_only]
    public fun set_required_version<MethodType>(
        self: &mut RequiredVersion,
        version: u64
    ) {
        *field::borrow_mut(&mut self.id, Key<MethodType> {}) = version;
    }

    #[test_only]
    public fun destroy(req: RequiredVersion) {
        let RequiredVersion { id, latest_version: _} = req;
        object::delete(id);
    }
}

#[test_only]
module wormhole::required_version_test {
    use sui::hash::{keccak256};
    use sui::object::{Self};
    use sui::tx_context::{Self};

    use wormhole::required_version::{Self};

    // NOTE: This exists to mock up sui::package for proposed upgrades.
    use wormhole::dummy_sui_package::{Self as package};

    struct SomeMethod {}
    struct AnotherMethod {}

    #[test]
    public fun test_check_minimum_requirement() {
        let ctx = &mut tx_context::dummy();

        let version = 1;
        let req = required_version::new(version, ctx);
        assert!(required_version::current(&req) == version, 0);

        required_version::add<SomeMethod>(&mut req);
        assert!(required_version::minimum_for<SomeMethod>(&req) == version, 0);

        // Should not abort here.
        required_version::check_minimum_requirement<SomeMethod>(&req, version);

        // And should not abort if the version is anything greater than the
        // current.
        let new_version = version + 1;
        required_version::check_minimum_requirement<SomeMethod>(
            &req,
            new_version
        );

        // Uptick based on new upgrade.
        let upgrade_cap = package::test_publish(
            object::id_from_address(@wormhole),
            ctx
        );
        let digest = keccak256(&x"DEADBEEF");
        let policy = package::upgrade_policy(&upgrade_cap);
        let upgrade_ticket =
            package::authorize_upgrade(&mut upgrade_cap, policy, digest);
        let upgrade_receipt = package::test_upgrade(upgrade_ticket);
        package::commit_upgrade(&mut upgrade_cap, upgrade_receipt);
        assert!(package::version(&upgrade_cap) == new_version, 0);

        // Update to the latest version.
        required_version::update_latest(&mut req, &upgrade_cap);
        assert!(required_version::current(&req) == new_version, 0);

        // Should still not abort here.
        required_version::check_minimum_requirement<SomeMethod>(
            &req,
            new_version
        );

        // Require new version for `SomeMethod` and show that
        // `check_minimum_requirement` still succeeds.
        required_version::require_current_version<SomeMethod>(&mut req);
        assert!(
            required_version::minimum_for<SomeMethod>(&req) == new_version,
            0
        );
        required_version::check_minimum_requirement<SomeMethod>(
            &req,
            new_version
        );

        // If another method gets added to the mix, it should automatically meet
        // the minimum requirement because its version will be the latest.
        required_version::add<AnotherMethod>(&mut req);
        assert!(
            required_version::minimum_for<AnotherMethod>(&req) == new_version,
            0
        );
        required_version::check_minimum_requirement<SomeMethod>(
            &req,
            new_version
        );

        // Clean up.
        package::make_immutable(upgrade_cap);
        required_version::destroy(req);
    }

    #[test]
    #[expected_failure(abort_code = required_version::E_OUTDATED_VERSION)]
    public fun test_cannot_check_minimum_requirement_with_outdated_version() {
        let ctx = &mut tx_context::dummy();

        let version = 1;
        let req = required_version::new(version, ctx);
        assert!(required_version::current(&req) == version, 0);

        required_version::add<SomeMethod>(&mut req);

        // Should not abort here.
        required_version::check_minimum_requirement<SomeMethod>(&req, version);

        // Uptick minimum requirement and fail at `check_minimum_requirement`.
        let new_version = 10;
        required_version::set_required_version<SomeMethod>(
            &mut req,
            new_version
        );
        let old_version = new_version - 1;
        required_version::check_minimum_requirement<SomeMethod>(
            &req,
            old_version
        );

        // Clean up.
        required_version::destroy(req);
    }
}
