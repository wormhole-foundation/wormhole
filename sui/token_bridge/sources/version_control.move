// SPDX-License-Identifier: Apache 2

/// This module implements dynamic field keys as empty structs. These keys are
/// used to determine the latest version for this build. If the current version
/// is not this build's, then paths through the `state` module will abort.
///
/// See `token_bridge::state` and `wormhole::package_utils` for more info.
module token_bridge::version_control {
    ////////////////////////////////////////////////////////////////////////////
    //
    //  Hard-coded Version Control
    //
    //  Before upgrading, please set the types for `current_version` and
    //  `previous_version` to match the correct types (current being the latest
    //  version reflecting this build).
    //
    ////////////////////////////////////////////////////////////////////////////

    public(friend) fun current_version(): V__0_2_0 {
       V__0_2_0 {}
    }

    #[test_only]
    public fun current_version_test_only(): V__0_2_0 {
        current_version()
    }

    public(friend) fun previous_version(): V__DUMMY {
        V__DUMMY {}
    }

    #[test_only]
    public fun previous_version_test_only(): V__DUMMY {
        previous_version()
    }

    ////////////////////////////////////////////////////////////////////////////
    //
    //  Change Log
    //
    //  Please write release notes as doc strings for each version struct. These
    //  notes will be our attempt at tracking upgrades. Wish us luck.
    //
    ////////////////////////////////////////////////////////////////////////////

    /// First published package on Sui mainnet.
    struct V__0_2_0 has store, drop, copy {}

    // Dummy.
    struct V__DUMMY has store, drop, copy {}

    ////////////////////////////////////////////////////////////////////////////
    //
    //  Implementation and Test-Only Methods
    //
    ////////////////////////////////////////////////////////////////////////////

    friend token_bridge::state;

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
}
