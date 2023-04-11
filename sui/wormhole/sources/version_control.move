// SPDX-License-Identifier: Apache 2

/// This module implements dynamic field keys as empty structs. These keys with
/// `RequiredVersion` are used to determine minimum build requirements for
/// particular Wormhole methods and breaking backward compatibility for these
/// methods if an upgrade requires the latest upgrade version for its
/// functionality.
///
/// See `wormhole::required_version` and `wormhole::state` for more info.
module wormhole::version_control {
    /// This value tracks the current version of the Wormhole version. We are
    /// placing this constant value at the top, which goes against Move style
    /// guides so that we bring special attention to changing this value when
    /// a new implementation is built for a contract upgrade.
    const CURRENT_BUILD_VERSION: u64 = 1;

    /// Key used to check minimum version requirement for `emitter` module.
    struct Emitter {}

    /// Key used to check minimum version requirement for `governance_message`
    /// module.
    struct GovernanceMessage {}

    /// Key used to check minimum version requirement for `migrate` module.
    struct Migrate {}

    /// Key used to check minimum version requirement for `publish_module`
    /// module.
    struct PublishMessage {}

    /// Key used to check minimum version requirement for `set_fee` module.
    struct SetFee {}

    /// Key used to check minimum version requirement for `transfer_fee` module.
    struct TransferFee {}

    /// Key used to check minimum version requirement for `update_guardian_set`
    /// module.
    struct UpdateGuardianSet {}

    /// Key used to check minimum version requirement for `vaa` module.
    struct Vaa {}

    /// Return const value `CURRENT_BUILD_VERSION` for this particular build.
    /// This value is used to determine whether this implementation meets
    /// minimum requirements for various Wormhole methods required by `State`.
    public fun version(): u64 {
        CURRENT_BUILD_VERSION
    }
}
