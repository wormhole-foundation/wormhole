// contracts/Setters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./State.sol";

contract Setters is State {
    /// @notice updates the current guardian set index
    /// @param newIndex the new guardian set index
    function updateGuardianSetIndex(uint32 newIndex) internal {
        _state.guardianSetIndex = newIndex;
    }

    /// @notice sets a guardian set to expire 1 day from now
    /// @param index the index for a given guardian set
    function expireGuardianSet(uint32 index) internal {
        _state.guardianSets[index].expirationTime = uint32(block.timestamp) + 86400;
    }

    /// @notice stores a guardian set
    /// @param set the guardian set 
    /// @param index the index to store the guardian set under
    function storeGuardianSet(Structs.GuardianSet memory set, uint32 index) internal {
        uint setLength = set.keys.length;
        for (uint i = 0; i < setLength; i++) {
            require(set.keys[i] != address(0), "Invalid key");
        }
        _state.guardianSets[index] = set;
    }

    /// @notice sets the implemented contract as initialized
    /// @dev https://github.com/wormhole-foundation/wormhole/issues/1930
    /// @param implementatiom the address of the implementation
    function setInitialized(address implementatiom) internal {
        _state.initializedImplementations[implementatiom] = true;
    }

    /// @notice sets a governance action as consumed to prevent reentry
    /// @param hash a hash from a governance VM
    function setGovernanceActionConsumed(bytes32 hash) internal {
        _state.consumedGovernanceActions[hash] = true;
    }

    /// @notice sets the chain id
    /// @param chainId the chain id
    function setChainId(uint16 chainId) internal {
        _state.provider.chainId = chainId;
    }

    /// @notice sets the chain id of the governance
    /// @param chainId the chain id of the governance
    function setGovernanceChainId(uint16 chainId) internal {
        _state.provider.governanceChainId = chainId;
    }

    /// @notice sets the governance contract
    function setGovernanceContract(bytes32 governanceContract) internal {
        _state.provider.governanceContract = governanceContract;
    }

    /// @notice sets the message fee
    /// @param newFee the fee to set the message fee to
    function setMessageFee(uint256 newFee) internal {
        _state.messageFee = newFee;
    }

    /// @notice updates the next sequence for a given emitter
    function setNextSequence(address emitter, uint64 sequence) internal {
        _state.sequences[emitter] = sequence;
    }

    /// @notice sets the EVM chain id
    function setEvmChainId(uint256 evmChainId) internal {
        require(evmChainId == block.chainid, "invalid evmChainId");
        _state.evmChainId = evmChainId;
    }
}