// SPDX-License-Identifier: Apache 2

/// This module implements dynamic field keys as empty structs. These keys with
/// `RequiredVersion` are used to determine minimum build requirements for
/// particular Token Bridge methods and breaking backward compatibility for
/// these methods if an upgrade requires the latest upgrade version for its
/// functionality.
///
/// See `wormhole::required_version` and `token_bridge::state` for more info.
module token_bridge::version_control {
    /// This value tracks the current version of the Token Bridge version. We
    /// are placing this constant value at the top, which goes against Move
    /// style guides so that we bring special attention to changing this value
    /// when a new implementation is built for a contract upgrade.
    const CURRENT_BUILD_VERSION: u64 = 1;

    /// Key used to check minimum version requirement for `attest_token` module.
    struct AttestToken {}

    /// Key used to check minimum version requirement for `complete_transfer`
    /// module.
    struct CompleteTransfer {}

    /// Key used to check minimum version requirement for
    /// `complete_transfer_with_payload` module.
    struct CompleteTransferWithPayload {}

    /// Key used to check minimum version requirement for `create_wrapped`
    /// module.
    struct CreateWrapped {}

    /// Key used to check minimum version requirement for `migrate` module.
    struct Migrate {}

    /// Key used to check minimum version requirement for `register_chain`
    /// module.
    struct RegisterChain {}

    /// Key used to check minimum version requirement for `transfer_tokens`
    /// module.
    struct TransferTokens {}

    /// Key used to check minimum version requirement for
    /// `transfer_tokens_with_payload` module.
    struct TransferTokensWithPayload {}

    /// Key used to check minimum version requirement for `vaa` module.
    struct Vaa {}

    /// Return const value `CURRENT_BUILD_VERSION` for this particular build.
    /// This value is used to determine whether this implementation meets
    /// minimum requirements for various Wormhole methods required by `State`.
    public fun version(): u64 {
        CURRENT_BUILD_VERSION
    }
}
