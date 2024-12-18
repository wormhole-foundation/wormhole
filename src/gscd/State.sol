// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "wormhole-sdk/interfaces/IWormhole.sol";

contract State {
  struct Provider {
    uint16 chainId;
    uint16 governanceChainId;
    bytes32 governanceContract;
  }

  struct WormholeState {
    Provider provider;

    // Mapping of guardian_set_index => guardian set
    mapping(uint32 => IWormhole.GuardianSet) guardianSets;

    // Current active guardian set index
    uint32 guardianSetIndex;

    // Period for which a guardian set stays active after it has been replaced
    uint32 guardianSetExpiry;

    // Sequence numbers per emitter
    mapping(address => uint64) sequences;

    // Mapping of consumed governance actions
    mapping(bytes32 => bool) consumedGovernanceActions;

    // Mapping of initialized implementations
    mapping(address => bool) initializedImplementations;

    uint256 messageFee;

    // EIP-155 Chain ID
    uint256 evmChainId;

    // Guardian set hashes. 
    mapping(uint32 => bytes32) guardianSetHashes;
  }

  WormholeState _state;
}
