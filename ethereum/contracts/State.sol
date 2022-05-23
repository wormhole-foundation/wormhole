// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./Structs.sol";

contract Events {
    event LogGuardianSetChanged(
        uint32 oldGuardianIndex,
        uint32 newGuardianIndex
    );

    event LogMessagePublished(
        address emitter_address,
        uint32 nonce,
        bytes payload
    );
}

contract Storage {
    struct WormholeState {
        Structs.Provider provider;
        // Mapping of guardian_set_index => guardian set
        mapping(uint32 => Structs.GuardianSet) guardianSets;
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
    }
}

contract State {
    Storage.WormholeState _state;
}
