// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./Structs.sol";

/// @title Events for Wormhole State contract
/// @notice Defines events related to changes in guardian sets and message publication
contract Events {
    /// @notice Event emitted when the guardian set changes
    /// @param oldGuardianIndex Index of the guardian set before the change
    /// @param newGuardianIndex Index of the guardian set after the change
    event LogGuardianSetChanged(
        uint32 oldGuardianIndex,
        uint32 newGuardianIndex
    );

    /// @notice Event emitted when a message is published
    /// @param emitter_address The address from which the message is sent
    /// @param nonce A nonce to ensure message uniqueness
    /// @param payload The data payload of the message
    event LogMessagePublished(
        address emitter_address,
        uint32 nonce,
        bytes payload
    );
}

/// @title Storage for Wormhole State contract
/// @notice Contains the storage states used by the Wormhole protocol
contract Storage {
    /// @dev Internal structure for maintaining Wormhole's state
    struct WormholeState {
        Structs.Provider provider;

        /// @dev Mapping of guardian_set_index => guardian set
        mapping(uint32 => Structs.GuardianSet) guardianSets;

        /// @dev Current active guardian set index
        uint32 guardianSetIndex;

        /// @dev Period for which a guardian set stays active after it has been replaced
        uint32 guardianSetExpiry;

        /// @dev Sequence numbers per emitter
        mapping(address => uint64) sequences;

        /// @dev Mapping of consumed governance actions
        mapping(bytes32 => bool) consumedGovernanceActions;

        /// @dev Mapping of initialized implementations
        mapping(address => bool) initializedImplementations;

        /// @dev Fee required to publish a message
        uint256 messageFee;

        /// @dev EIP-155 Chain ID
        uint256 evmChainId;
    }
}

/// @title State contract for Wormhole
/// @notice Manages the state variables for the Wormhole protocol
contract State {
    /// @dev State variable containing all Wormhole state information
    Storage.WormholeState _state;
}
