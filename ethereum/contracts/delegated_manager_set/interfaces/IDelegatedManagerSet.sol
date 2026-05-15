// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

interface IDelegatedManagerSet {
    /// @notice The manager set for a chain has been updated.
    /// @dev Topic0
    ///      0x923bb3e008cbe2e6e13010f8fc996a0c8c62d29f8cf1b252f663d8f396843b9d.
    /// @param chain The chain id for which the manager set has been set.
    /// @param index The index that was set.
    event NewManagerSet(uint16 chain, uint32 index);

    struct ManagerSetUpdate {
        // Governance Header
        // module: "DelegatedManager" left-padded
        bytes32 module;
        // governance action: 1
        uint8 action;
        // governance packet chain id: this or 0
        uint16 chainId;

        // Chain ID
        uint16 managerChainId;
        // Manager Set Index
        uint32 managerSetIndex;
        // New Manager Set
        bytes managerSet;
    }

    function submitNewManagerSet(
        bytes memory encodedVM
    ) external;
    function getManagerSet(
        uint16 chainId,
        uint32 index
    ) external view returns (bytes memory);
    function getCurrentManagerSetIndex(
        uint16 chainId
    ) external view returns (uint32);
    function getCurrentManagerSet(
        uint16 chainId
    ) external view returns (bytes memory);
}
