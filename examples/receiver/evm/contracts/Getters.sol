// contracts/Getters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./State.sol";

contract Getters is State {
    function owner() public view returns (address) {
        return _state.owner;
    } 
    
    function getGuardianSet(uint32 index) public view returns (Structs.GuardianSet memory) {
        return _state.guardianSets[index];
    }

    function getCurrentGuardianSetIndex() public view returns (uint32) {
        return _state.guardianSetIndex;
    }

    function getGuardianSetExpiry() public view returns (uint32) {
        return _state.guardianSetExpiry;
    }

    function governanceActionIsConsumed(bytes32 hash) public view returns (bool) {
        return _state.consumedGovernanceActions[hash];
    }

    function isInitialized(address impl) public view returns (bool) {
        return _state.initializedImplementations[impl];
    }

    function governanceChainId() public view returns (uint16){
        return _state.provider.governanceChainId;
    }

    function governanceContract() public view returns (bytes32){
        return _state.provider.governanceContract;
    }
}
