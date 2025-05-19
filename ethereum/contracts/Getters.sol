// contracts/Getters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./State.sol";

contract Getters is State {
    /**
     * @notice Returns the GuardianSet for a given index.
     * @param index The index of the GuardianSet.
     * @return The GuardianSet struct.
     */
    function getGuardianSet(uint32 index) public view returns (Structs.GuardianSet memory) {
        return _state.guardianSets[index];
    }

    /**
     * @notice Returns the current GuardianSet index.
     * @return The current GuardianSet index.
     */
    function getCurrentGuardianSetIndex() public view returns (uint32) {
        return _state.guardianSetIndex;
    }

    /**
     * @notice Returns the expiry time for the current GuardianSet.
     * @return The expiry timestamp (in seconds since epoch).
     */
    function getGuardianSetExpiry() public view returns (uint32) {
        return _state.guardianSetExpiry;
    }

    /**
     * @notice Checks if a governance action has already been consumed.
     * @param hash The hash of the governance action.
     * @return True if the action has been consumed, false otherwise.
     */
    function governanceActionIsConsumed(bytes32 hash) public view returns (bool) {
        return _state.consumedGovernanceActions[hash];
    }

    /**
     * @notice Checks if an implementation address has been initialized.
     * @param impl The implementation address.
     * @return True if initialized, false otherwise.
     */
    function isInitialized(address impl) public view returns (bool) {
        return _state.initializedImplementations[impl];
    }

    /**
     * @notice Returns the Wormhole chain ID.
     * @return The chain ID.
     */
    function chainId() public view returns (uint16) {
        return _state.provider.chainId;
    }

    /**
     * @notice Returns the EVM chain ID.
     * @return The EVM chain ID.
     */
    function evmChainId() public view returns (uint256) {
        return _state.evmChainId;
    }

    /**
     * @notice Returns true if the contract is running on a forked chain.
     * @return True if on a fork, false otherwise.
     */
    function isFork() public view returns (bool) {
        return evmChainId() != block.chainid;
    }

    /**
     * @notice Returns the governance chain ID.
     * @return The governance chain ID.
     */
    function governanceChainId() public view returns (uint16){
        return _state.provider.governanceChainId;
    }

    /**
     * @notice Returns the governance contract address.
     * @return The governance contract address (bytes32).
     */
    function governanceContract() public view returns (bytes32){
        return _state.provider.governanceContract;
    }

    /**
     * @notice Returns the current message fee.
     * @return The message fee in wei.
     */
    function messageFee() public view returns (uint256) {
        return _state.messageFee;
    }

    /**
     * @notice Returns the next sequence number for a given emitter address.
     * @param emitter The emitter address.
     * @return The next sequence number.
     */
    function nextSequence(address emitter) public view returns (uint64) {
        return _state.sequences[emitter];
    }
}
