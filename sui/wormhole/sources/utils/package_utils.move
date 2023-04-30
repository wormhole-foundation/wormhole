// SPDX-License-Identifier: Apache 2

/// This module implements utilities that supplement those methods implemented
/// in `sui::package`.
module wormhole::package_utils {
    use std::type_name::{Self, TypeName};
    use sui::dynamic_field::{Self as field};
    use sui::object::{Self, ID, UID};
    use sui::package::{Self, UpgradeCap, UpgradeTicket, UpgradeReceipt};

    use wormhole::bytes32::{Self, Bytes32};

    /// `UpgradeCap` is not from the same package as `T`.
    const E_INVALID_UPGRADE_CAP: u64 = 0;
    /// Build is not current.
    const E_NOT_CURRENT_VERSION: u64 = 1;
    /// Old version to update from is wrong.
    const E_INCORRECT_OLD_VERSION: u64 = 2;
    /// Old and new are the same version.
    const E_SAME_VERSION: u64 = 3;
    /// Version types must come from this module.
    const E_TYPE_NOT_ALLOWED: u64 = 4;

    /// Key for version dynamic fields.
    struct CurrentVersion has store, drop, copy {}

    /// Key for dynamic field reflecting current package info. Its value is
    /// `PackageInfo`.
    struct CurrentPackage has store, drop, copy {}
    struct PendingPackage has store, drop, copy {}

    struct PackageInfo has store, drop, copy {
        package: ID,
        digest: Bytes32
    }

    /// Retrieve current package ID, which should be the only one that anyone is
    /// allowed to interact with.
    public fun current_package(id: &UID): ID {
        let info: &PackageInfo = field::borrow(id, CurrentPackage {});
        info.package
    }

    /// Retrieve the build digest reflecting the current build.
    public fun current_digest(id: &UID): Bytes32 {
        let info: &PackageInfo = field::borrow(id, CurrentPackage {});
        info.digest
    }

    /// Retrieve the upgraded package ID, which was taken from `UpgradeCap`
    /// during `commit_upgrade`.
    public fun committed_package(id: &UID): ID {
        let info: &PackageInfo = field::borrow(id, PendingPackage {});
        info.package
    }

    /// Retrieve the build digest of the latest upgrade, which was the same
    /// digest used when `authorize_upgrade` is called.
    public fun authorized_digest(id: &UID): Bytes32 {
        let info: &PackageInfo = field::borrow(id, PendingPackage {});
        info.digest
    }

    /// Convenience method that can be used with any package that requires
    /// `UpgradeCap` to have certain preconditions before it is considered
    /// belonging to `T` object's package.
    public fun assert_package_upgrade_cap<T>(
        cap: &UpgradeCap,
        expected_policy: u8,
        expected_version: u64
    ) {
        let expected_package =
            sui::address::from_bytes(
                sui::hex::decode(
                    std::ascii::into_bytes(
                        std::type_name::get_address(
                            &std::type_name::get<T>()
                        )
                    )
                )
            );
        let cap_package =
            object::id_to_address(&package::upgrade_package(cap));
        assert!(
            (
                cap_package == expected_package &&
                package::upgrade_policy(cap) == expected_policy &&
                package::version(cap) == expected_version
            ),
            E_INVALID_UPGRADE_CAP
        );
    }

    /// Assert that the version type passed into this method is what exists
    /// as the current version.
    public fun assert_version<Version: store + drop>(
        id: &UID,
        _version: Version
    ) {
        assert!(
            field::exists_with_type<CurrentVersion, Version>(
                id,
                CurrentVersion {}
            ),
            E_NOT_CURRENT_VERSION
        )
    }

    // Retrieve the `TypeName` of a given version.
    public fun type_of_version<Version: drop>(_version: Version): TypeName {
        type_name::get<Version>()
    }

    /// Initialize package info and set the initial version. This should be done
    /// when a contract's state/storage shared object is created.
    public fun init_package_info<InitialVersion: store>(
        id: &mut UID,
        version: InitialVersion,
        upgrade_cap: &UpgradeCap
    ) {
        let package = package::upgrade_package(upgrade_cap);
        field::add(
            id,
            CurrentPackage {},
            PackageInfo { package, digest: bytes32::default() }
        );

        // Set placeholders for pending package. We don't ever plan on removing
        // this field.
        field::add(
            id,
            PendingPackage {},
            PackageInfo { package, digest: bytes32::default() }
        );

        // Set the initial version.
        field::add(id, CurrentVersion {}, version);
    }

    /// Perform the version switchover and copy package info from pending to
    /// current. This method should be executed after an upgrade (via a migrate
    /// method) from the upgraded package.
    ///
    /// NOTE: This method can only be called once with the same version type
    /// arguments.
    public fun migrate_version<
        Old: store + drop,
        New: store + drop
    >(
        id: &mut UID,
        old_version: Old,
        new_version: New
    ) {
        update_version_type(id, old_version, new_version);

        update_package_info_from_pending(id);
    }

    /// Helper for `sui::package::authorize_upgrade` to modify pending package
    /// info by updating its digest.
    ///
    /// NOTE: This digest will be copied over when `migrate_version` is called.
    public fun authorize_upgrade(
        id: &mut UID,
        upgrade_cap: &mut UpgradeCap,
        package_digest: Bytes32
    ): UpgradeTicket {
        let policy = package::upgrade_policy(upgrade_cap);

        // Manage saving the current digest.
        set_authorized_digest(id, package_digest);

        // Finally authorize upgrade.
        package::authorize_upgrade(
            upgrade_cap,
            policy,
            bytes32::to_bytes(package_digest),
        )
    }

    /// Helper for `sui::package::commit_upgrade` to modify pending package info
    /// by updating its package ID with from what exists in the `UpgradeCap`.
    /// This method returns the last package and the upgraded package IDs.
    ///
    /// NOTE: This package ID (second return value) will be copied over when
    /// `migrate_version` is called.
    public fun commit_upgrade(
        id: &mut UID,
        upgrade_cap: &mut UpgradeCap,
        receipt: UpgradeReceipt
    ): (ID, ID) {
        // Uptick the upgrade cap version number using this receipt.
        package::commit_upgrade(upgrade_cap, receipt);

        // Take the last pending package and replace it with the one now in
        // the upgrade cap.
        let previous_package = committed_package(id);
        set_commited_package(id, upgrade_cap);

        // Return the package IDs.
        (previous_package, committed_package(id))
    }

    fun set_commited_package(id: &mut UID, upgrade_cap: &UpgradeCap) {
        let info: &mut PackageInfo = field::borrow_mut(id, PendingPackage {});
        info.package = package::upgrade_package(upgrade_cap);
    }

    fun set_authorized_digest(id: &mut UID, digest: Bytes32) {
        let info: &mut PackageInfo = field::borrow_mut(id, PendingPackage {});
        info.digest = digest;
    }

    fun update_package_info_from_pending(id: &mut UID) {
        let pending: PackageInfo = *field::borrow(id, PendingPackage {});
        *field::borrow_mut(id, CurrentPackage {}) = pending;
    }

    /// Update from version n to n+1. We enforce that the versions be kept in
    /// a module called "version_control".
    fun update_version_type<
        Old: store + drop,
        New: store + drop
    >(
        id: &mut UID,
        _old_version: Old,
        new_version: New
    ) {
        use std::ascii::{into_bytes};

        assert!(
            field::exists_with_type<CurrentVersion, Old>(id, CurrentVersion {}),
            E_INCORRECT_OLD_VERSION
        );
        let _: Old = field::remove(id, CurrentVersion {});

        let new_type = type_name::get<New>();
        // Make sure the new type does not equal the old type, which means there
        // is no protection against either build.
        assert!(new_type != type_name::get<Old>(), E_SAME_VERSION);

        // Also make sure `New` originates from this module.
        let module_name = into_bytes(type_name::get_module(&new_type));
        assert!(module_name == b"version_control", E_TYPE_NOT_ALLOWED);

        // Finally add the new version.
        field::add(id, CurrentVersion {}, new_version);
    }

    #[test_only]
    public fun remove_package_info(id: &mut UID) {
        let _: PackageInfo = field::remove(id, CurrentPackage {});
        let _: PackageInfo = field::remove(id, PendingPackage {});
    }

    #[test_only]
    public fun init_version<Version: store>(
        id: &mut UID,
        version: Version
    ) {
        field::add(id, CurrentVersion {}, version);
    }

    #[test_only]
    public fun update_version_type_test_only<
        Old: store + drop,
        New: store + drop
    >(
        id: &mut UID,
        old_version: Old,
        new_version: New
    ) {
        update_version_type(id, old_version, new_version)
    }
}

#[test_only]
module wormhole::package_utils_tests {
    use sui::object::{Self, UID};
    use sui::tx_context::{Self};

    use wormhole::package_utils::{Self};
    use wormhole::version_control::{Self};

    struct State has key {
        id: UID
    }

    struct V_DUMMY has store, drop, copy {}

    #[test]
    fun test_assert_current() {
        // Create dummy state.
        let state = State { id: object::new(&mut tx_context::dummy()) };
        package_utils::init_version(
            &mut state.id,
            version_control::current_version()
        );

        package_utils::assert_version(
            &state.id,
            version_control::current_version()
        );

        // Clean up.
        let State { id } = state;
        object::delete(id);
    }

    #[test]
    #[expected_failure(abort_code = package_utils::E_INCORRECT_OLD_VERSION)]
    fun test_cannot_update_incorrect_old_version() {
        // Create dummy state.
        let state = State { id: object::new(&mut tx_context::dummy()) };
        package_utils::init_version(
            &mut state.id,
            version_control::current_version()
        );

        package_utils::assert_version(
            &state.id,
            version_control::current_version()
        );

        // You shall not pass!
        package_utils::update_version_type_test_only(
            &mut state.id,
            version_control::next_version(),
            version_control::next_version()
        );

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = package_utils::E_SAME_VERSION)]
    fun test_cannot_update_same_version() {
        // Create dummy state.
        let state = State { id: object::new(&mut tx_context::dummy()) };
        package_utils::init_version(
            &mut state.id,
            version_control::current_version()
        );

        package_utils::assert_version(
            &state.id,
            version_control::current_version()
        );

        // You shall not pass!
        package_utils::update_version_type_test_only(
            &mut state.id,
            version_control::current_version(),
            version_control::current_version()
        );

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_assert_current_outdated_version() {
        // Create dummy state.
        let state = State { id: object::new(&mut tx_context::dummy()) };
        package_utils::init_version(
            &mut state.id,
            version_control::current_version()
        );

        package_utils::assert_version(
            &state.id,
            version_control::current_version()
        );

        // Valid update.
        package_utils::update_version_type_test_only(
            &mut state.id,
            version_control::current_version(),
            version_control::next_version()
        );

        // You shall not pass!
        package_utils::assert_version(
            &state.id,
            version_control::current_version()
        );

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = package_utils::E_TYPE_NOT_ALLOWED)]
    fun test_cannot_update_type_not_allowed() {
        // Create dummy state.
        let state = State { id: object::new(&mut tx_context::dummy()) };
        package_utils::init_version(
            &mut state.id,
            version_control::current_version()
        );

        package_utils::assert_version(
            &state.id,
            version_control::current_version()
        );

        // You shall not pass!
        package_utils::update_version_type_test_only(
            &mut state.id,
            version_control::current_version(),
            V_DUMMY {}
        );

        abort 42
    }

    #[test]
    fun test_latest_version_different_from_previous() {
        let prev = version_control::previous_version();
        let curr = version_control::current_version();
        assert!(package_utils::type_of_version(prev) != package_utils::type_of_version(curr), 0);
    }
}
