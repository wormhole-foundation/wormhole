// contracts/Getters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./State.sol";

/// @title Getters for Wormhole State
/// @notice Provides read-only access to the state variables of Wormhole stored in the State contract.
contract Getters is State {
    /// @notice Fetches the guardian set for the given index
    /// @param index The index for the guardian set
    /// @return guardianSet Returns a guardian set
    function getGuardianSet(uint32 index) public view returns (Structs.GuardianSet memory) {
        return _state.guardianSets[index];
    }

    /// @notice Retrieves the index of the current guardian set
    /// @return index Returns the guardian set's index
    function getCurrentGuardianSetIndex() public view returns (uint32) {
        return _state.guardianSetIndex;
    }

    /// @notice Returns the expiration time for the current guardian set
    function getGuardianSetExpiry() public view returns (uint32) {
        return _state.guardianSetExpiry;
    }

    /// @notice Checks whether a governance action has already been consumed
    /// @return consumed Returns true if consumed
    function governanceActionIsConsumed(bytes32 hash) public view returns (bool) {
        return _state.consumedGovernanceActions[hash];
    }

    /// @notice Determines if the given contract implementation has been initialized
    /// @param impl The address of the contract implementation
    /// @return initialized Returns true if initialized
    function isInitialized(address impl) public view returns (bool) {
        return _state.initializedImplementations[impl];
    }

    /// @notice Returns the chain ID
    function chainId() public view returns (uint16) {
        return _state.provider.chainId;
    }

    /// @notice Returns the EVM chain ID
    function evmChainId() public view returns (uint256) {
        return _state.evmChainId;
    }

    /// @notice Checks if the current chain is a fork
    function isFork() public view returns (bool) {
        return evmChainId() != block.chainid;
    }

    /// @notice Returns the governance chain ID
    function governanceChainId() public view returns (uint16){
        return _state.provider.governanceChainId;
    }

    /// @notice Returns the address of the governance contract in bytes32
    function governanceContract() public view returns (bytes32){
        return _state.provider.governanceContract;
    }

    /// @notice Gets the current message fee
    function messageFee() public view returns (uint256) {
        return _state.messageFee;
    }

    /// @notice Fetches the next sequence number for a given emitter address
    function nextSequence(address emitter) public view returns (uint64) {
        return _state.sequences[emitter];
    }
}