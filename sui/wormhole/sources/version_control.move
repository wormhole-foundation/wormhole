// SPDX-License-Identifier: Apache 2

/// This module implements dynamic field keys as empty structs. These keys are
/// used to determine the latest version for this build. If the current version
/// is not this build's, then paths through the `state` module will abort.
///
/// See `wormhole::state` and `wormhole::package_utils` for more info.
module wormhole::version_control {
    ////////////////////////////////////////////////////////////////////////////
    //
    //  Hard-coded Version Control
    //
    //  Before upgrading, please set the types for `current_version` and
    //  `previous_version` to match the correct types (current being the latest
    //  version reflecting this build).
    //
    ////////////////////////////////////////////////////////////////////////////

    public(friend) fun current_version(): V__0_1_2 {
       V__0_1_2 {}
    }

    public(friend) fun previous_version(): V__0_1_1 {
        V__0_1_1 {}
    }

    ////////////////////////////////////////////////////////////////////////////
    //
    //  Change Log
    //
    //  Please write release notes as doc strings for each version struct. These
    //  notes will be our attempt at tracking upgrades. Wish us luck.
    //
    ////////////////////////////////////////////////////////////////////////////

    /// RELEASE NOTES
    ///
    /// - Fix `package_utils` to initialize pending package ID to be the same as
    ///   the current package ID (both from the `UpgradeCap`).
    ///
    /// Also added `migrate__v__0_1_2 in `wormhole::state`, which is a no-op.
    struct V__0_1_2 has store, drop, copy {}

    /// RELEASE NOTES
    ///
    /// - Add `PackageInfo` to `wormhole::package_utils`, which allows tracking
    ///   of the most relevant package ID.
    /// - Refactor package management into `wormhole::package_utils`.
    /// - Add `MigrateComplete` event in `migrate`.
    ///
    /// Also added `migrate__v__0_1_1` in `wormhole::state`, which is
    /// meant to perform a one-time `State` modification via `migrate`.
    ///
    /// https://github.com/wormhole-foundation/wormhole/tree/5be93774da4c6d8062521933be29250aed97245d
    struct V__0_1_1 has store, drop, copy {}

    /// First published package.
    ///
    /// NOTE: This version is published on Sui testnet.
    ///
    /// https://github.com/wormhole-foundation/wormhole/tree/03ff1b24cf913ed04ce59fe26b5d3abd53015f28
    struct V__0_1_0 has store, drop, copy {}

    // Dummy.
    struct V__DUMMY has store, drop, copy {}

    ////////////////////////////////////////////////////////////////////////////
    //
    //  Implementation and Test-Only Methods
    //
    ////////////////////////////////////////////////////////////////////////////

    friend wormhole::state;

    #[test_only]
    friend wormhole::package_utils_tests;

    #[test_only]
    public fun dummy(): V__DUMMY {
        V__DUMMY {}
    }

    #[test_only]
    struct V__MIGRATED has store, drop, copy {}

    #[test_only]
    public fun next_version(): V__MIGRATED {
        V__MIGRATED {}
    }

    #[test_only]
    public fun previous_version_test_only(): V__0_1_1 {
        previous_version()
    }
}
