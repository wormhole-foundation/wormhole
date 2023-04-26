// SPDX-License-Identifier: Apache 2

/// This module implements utilities that supplement those methods implemented
/// in `sui::package`.
module wormhole::package_utils {
    use std::type_name::{Self};
    use sui::dynamic_field::{Self as field};
    use sui::object::{Self, UID};
    use sui::package::{Self, UpgradeCap};

    /// `UpgradeCap` is not from the same package as `T`.
    const E_INVALID_UPGRADE_CAP: u64 = 0;
    /// Build is not current.
    const E_OUTDATED_VERSION: u64 = 1;
    /// Old version to update from is wrong.
    const E_INCORRECT_OLD_VERSION: u64 = 2;
    /// Old and new are the same version.
    const E_SAME_VERSION: u64 = 3;
    /// Current version is misconfigured.
    const E_INVALID_VERSION: u64 = 4;
    /// Version types must come from this module.
    const E_TYPE_NOT_ALLOWED: u64 = 5;

    /// Key for version dynamic fields.
    struct CurrentVersion has store, drop, copy {}

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

    /// Update from version n to n+1. We enforce that the versions be kept in
    /// a module called "version_control".
    public fun update_version_type<
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

    public fun assert_version<Version: store + drop>(id: &UID, _version: Version) {
        assert!(
            field::exists_with_type<CurrentVersion, Version>(id, CurrentVersion {}),
            E_OUTDATED_VERSION
        )
    }

    public fun init_version<Version: store>(id: &mut UID, version: Version) {
        field::add(id, CurrentVersion {}, version);
    }
}
