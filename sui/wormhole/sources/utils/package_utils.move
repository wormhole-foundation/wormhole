// SPDX-License-Identifier: Apache 2

/// This module implements utilities that supplement those methods implemented
/// in `sui::package`.
module wormhole::package_utils {
    use std::type_name::{Self, TypeName};
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

    public fun assert_version<Version: store + drop>(
        id: &UID,
        _version: Version
    ) {
        assert!(
            field::exists_with_type<CurrentVersion, Version>(
                id,
                CurrentVersion {}
            ),
            E_OUTDATED_VERSION
        )
    }

    public fun init_version<Version: store>(id: &mut UID, version: Version) {
        field::add(id, CurrentVersion {}, version);
    }

    public fun type_of_version<Version: drop>(_version: Version): TypeName {
        type_name::get<Version>()
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
        package_utils::update_version_type(
            &mut state.id,
            version_control::next_version(),
            version_control::next_version()
        );

        // Clean up.
        let State { id } = state;
        object::delete(id);
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
        package_utils::update_version_type(
            &mut state.id,
            version_control::current_version(),
            version_control::current_version()
        );

        // Clean up.
        let State { id } = state;
        object::delete(id);
    }

    #[test]
    #[expected_failure(abort_code = package_utils::E_OUTDATED_VERSION)]
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
        package_utils::update_version_type(
            &mut state.id,
            version_control::current_version(),
            version_control::next_version()
        );

        // You shall not pass!
        package_utils::assert_version(
            &state.id,
            version_control::current_version()
        );

        // Clean up.
        let State { id } = state;
        object::delete(id);
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
        package_utils::update_version_type(
            &mut state.id,
            version_control::current_version(),
            V_DUMMY {}
        );

        // Clean up.
        let State { id } = state;
        object::delete(id);
    }

    #[test]
    fun test_latest_version_different_from_previous() {
        let prev = version_control::previous_version();
        let curr = version_control::current_version();
        assert!(package_utils::type_of_version(prev) != package_utils::type_of_version(curr), 0);
    }
}
