// SPDX-License-Identifier: Apache 2

/// This module implements utilities that supplement those methods implemented
/// in `sui::package`.
module wormhole::package_utils {
    use sui::object::{Self};
    use sui::package::{Self, UpgradeCap};

    /// `UpgradeCap` is not from the same package as `T`.
    const E_INVALID_UPGRADE_CAP: u64 = 0;

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
}
