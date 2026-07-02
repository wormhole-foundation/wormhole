// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

/// @notice ERC-7201 namespaced storage for the token bridge `pauser` / `freezer` / `unpauser`
///         roles and the pause expiry.
/// @dev Decoupling these roles from `BridgeStorage.State` means adding pauser support does not
///      shift any pre-existing storage slot (notably `_status` from `ReentrancyGuard` at slot 13)
///      on an in-place proxy upgrade. The `paused` flag itself lives in `BridgeStorage.Provider`
///      so the hot-path `notPaused` check piggybacks on the SLOAD that already loads `chainId`.
///      `pauseExpiry` lives here (not in `Provider`) because it is only read by the rare,
///      non-hot-path `unpauseExpired`; keeping it out of `Provider` avoids touching that layout.
///      Namespace: "wormhole.tokenbridge.pauser.storage".
library BridgePauserStorage {
    struct Layout {
        // Address authorized to call pause(). May be address(0), in which case pause() reverts
        // before comparing msg.sender. See the "Pausing" section of whitepapers/0003_token_bridge.md.
        address pauser;
        // Address authorized to call unpause(). May be address(0), in which case unpause() reverts
        // before comparing msg.sender; recovery then requires governance to first assign a non-zero
        // unpauser via SetPauserAddresses.
        address unpauser;
        // NOTE: `freezer` and `pauseExpiry` are APPENDED (not inserted) so the slot assignment of
        // the pre-existing `pauser`/`unpauser` fields is preserved on an in-place upgrade. The
        // struct field order intentionally differs from the SetPauserAddresses wire order
        // (pauser, freezer, unpauser); each role is set independently by the governance handler.
        //
        // Address authorized to call freeze(). May be address(0) (unassigned), in which case
        // freeze() reverts before comparing msg.sender. The higher-trust counterpart to `pauser`.
        address freezer;
        // Timestamp (unix seconds) at which an active pause becomes eligible to be lifted
        // permissionlessly via unpauseExpired(). Set by pause()/freeze()/unpause(). Initializes to
        // zero and does not alias non-zero pre-existing storage. Packs into the `freezer` slot.
        uint64 pauseExpiry;
    }

    /// @dev `keccak256(abi.encode(uint256(keccak256("wormhole.tokenbridge.pauser.storage")) - 1))
    ///       & ~bytes32(uint256(0xff))` — precomputed per ERC-7201.
    bytes32 internal constant LAYOUT_SLOT =
        0x685f7dd8ace9c4fb94a4997fcd733e0d769273ee87b95731641e14d0cc4a6700;

    function data() internal pure returns (Layout storage l) {
        bytes32 slot = LAYOUT_SLOT;
        assembly {
            l.slot := slot
        }
    }
}
