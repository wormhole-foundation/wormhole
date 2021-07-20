// contracts/Setters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./State.sol";

contract Setters is State {
    function updateGuardianSetIndex(uint32 newIndex) internal {
        _state.guardianSetIndex = newIndex;
    }

    function expireGuardianSet(uint32 index) internal {
        _state.guardianSets[index].expirationTime = uint32(block.timestamp) + 86400;
    }

    function storeGuardianSet(Structs.GuardianSet memory set, uint32 index) internal {
        _state.guardianSets[index] = set;
    }

    function setInitialized(address implementatiom) internal {
        _state.initializedImplementations[implementatiom] = true;
    }

    function setGovernanceActionConsumed(bytes32 hash) internal {
        _state.consumedGovernanceActions[hash] = true;
    }

    function setChainId(uint16 chainId) internal {
        _state.provider.chainId = chainId;
    }

    function setGovernanceChainId(uint16 chainId) internal {
        _state.provider.governanceChainId = chainId;
    }

    function setGovernanceContract(bytes32 governanceContract) internal {
        _state.provider.governanceContract = governanceContract;
    }

    function setMessageFee(uint256 newFee) internal {
        _state.messageFee = newFee;
    }

    function setNextSequence(address emitter, uint64 sequence) internal {
        _state.sequences[emitter] = sequence;
    }
}