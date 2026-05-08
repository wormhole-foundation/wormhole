// contracts/Getters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./State.sol";

/// @title Getters
/// @notice Read-only accessors for the Wormhole core bridge state.
contract Getters is State {
    /// @notice Returns the guardian set at the given version index.
    /// @dev Guardian sets are versioned starting at 0 and increment by 1 on each rotation.
    ///      An expired guardian set (expirationTime < block.timestamp) may still be returned
    ///      but will fail VAA verification. Old sets expire 24 hours after a new one is installed.
    /// @param index The version number of the guardian set to retrieve.
    /// @return The GuardianSet struct containing the guardian keys and expiration time.
    function getGuardianSet(
        uint32 index
    ) public view returns (Structs.GuardianSet memory) {
        return _state.guardianSets[index];
    }

    /// @notice Returns the index of the currently active guardian set.
    /// @dev This is the version number of the guardian set that must sign governance packets
    ///      and that is used to validate freshly submitted VAAs.
    /// @return The index of the current guardian set.
    function getCurrentGuardianSetIndex() public view returns (uint32) {
        return _state.guardianSetIndex;
    }

    /// @notice Returns the duration (in seconds) an old guardian set remains valid after rotation.
    /// @dev When a new guardian set is installed, the previous set's `expirationTime` is set to
    ///      `block.timestamp + getGuardianSetExpiry()`. This window allows in-flight VAAs signed
    ///      by the old set to be delivered before the set becomes invalid.
    /// @return The expiry duration in seconds (currently 86400 = 24 hours).
    function getGuardianSetExpiry() public view returns (uint32) {
        return _state.guardianSetExpiry;
    }

    /// @notice Returns whether a governance VAA has already been executed.
    /// @dev Governance actions are tracked by their VAA hash to prevent replay attacks.
    ///      Once a governance action is consumed, the same VAA can never be re-submitted.
    /// @param hash The double-keccak256 hash of the governance VAA body.
    /// @return True if the governance action has been executed, false otherwise.
    function governanceActionIsConsumed(
        bytes32 hash
    ) public view returns (bool) {
        return _state.consumedGovernanceActions[hash];
    }

    /// @notice Returns whether a given implementation address has been initialized.
    /// @dev Each implementation contract must call `initialize()` exactly once after being upgraded to.
    ///      This flag prevents double-initialization if `initialize()` is called multiple times.
    /// @param impl The address of the implementation contract to check.
    /// @return True if the implementation has been initialized, false otherwise.
    function isInitialized(
        address impl
    ) public view returns (bool) {
        return _state.initializedImplementations[impl];
    }

    /// @notice Returns the Wormhole chain ID of this chain.
    /// @dev IMPORTANT: This is the Wormhole-specific chain ID, NOT the EVM chain ID (`block.chainid`).
    ///      Wormhole maintains its own chain ID registry across all supported networks.
    ///      For example, Ethereum mainnet has Wormhole chain ID 2, BSC has 4, Polygon has 5.
    ///      See https://wormhole.com/docs/products/reference/chain-ids/ for the full registry.
    ///      To get the native EVM chain ID, use `evmChainId()`.
    /// @return The Wormhole chain ID for this chain.
    function chainId() public view returns (uint16) {
        return _state.provider.chainId;
    }

    /// @notice Returns the native EVM chain ID of this chain (`block.chainid`).
    /// @dev Unlike `chainId()` which returns the Wormhole chain ID, this returns the canonical
    ///      EVM chain ID as defined by EIP-155. Used to detect hard forks via `isFork()`.
    ///      This value is set during `initialize()` and must match `block.chainid` at all times
    ///      on a non-forked chain.
    /// @return The EVM chain ID (e.g. 1 for Ethereum mainnet, 56 for BSC).
    function evmChainId() public view returns (uint256) {
        return _state.evmChainId;
    }

    /// @notice Returns whether this chain is a hard fork of another chain.
    /// @dev Returns true when the stored `evmChainId` does not match the current `block.chainid`.
    ///      This can happen after an unexpected hard fork (e.g. ETH/ETC split). In a fork scenario,
    ///      governance actions and new guardian set upgrades on the original chain would otherwise
    ///      be replayable on the forked chain. Use `submitRecoverChainId` to re-synchronize.
    /// @return True if the current `block.chainid` differs from the stored EVM chain ID.
    function isFork() public view returns (bool) {
        return evmChainId() != block.chainid;
    }

    /// @notice Returns the Wormhole chain ID of the chain hosting the governance contract.
    /// @dev Governance VAAs must originate from this chain. Currently set to Solana (Wormhole chain ID 1).
    ///      All governance instructions (guardian set upgrades, fee changes, contract upgrades) must
    ///      be signed by the current guardian set and emitted from this chain.
    /// @return The Wormhole chain ID of the governance chain (currently 1 for Solana).
    function governanceChainId() public view returns (uint16) {
        return _state.provider.governanceChainId;
    }

    /// @notice Returns the address of the governance contract on the governance chain, left-padded to 32 bytes.
    /// @dev On Solana, this is the Wormhole program's governance account padded to 32 bytes (0x4 left-padded).
    ///      Governance VAAs are only accepted if their emitter address exactly matches this value.
    /// @return The 32-byte governance contract address.
    function governanceContract() public view returns (bytes32) {
        return _state.provider.governanceContract;
    }

    /// @notice Returns the fee in wei required to publish a Wormhole message on this chain.
    /// @dev Callers must send exactly `messageFee()` wei when calling `publishMessage()`.
    ///      The fee can be updated via the `submitSetMessageFee` governance action.
    ///      Accumulated fees can be transferred to a recipient via `submitTransferFees`.
    /// @return The current message publishing fee in wei.
    function messageFee() public view returns (uint256) {
        return _state.messageFee;
    }

    /// @notice Returns the next sequence number that will be assigned to a message from the given emitter.
    /// @dev Sequence numbers are monotonically increasing per emitter address, starting at 0.
    ///      Together with the emitter chain ID and address, the sequence number uniquely identifies
    ///      a VAA. Integrators on the destination chain use this for replay protection.
    /// @param emitter The address of the message emitter to query.
    /// @return The next sequence number that `publishMessage` will assign to this emitter.
    function nextSequence(
        address emitter
    ) public view returns (uint64) {
        return _state.sequences[emitter];
    }
}
