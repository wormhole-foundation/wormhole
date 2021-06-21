// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./BridgeStructs.sol";

contract BridgeStorage {
    struct Provider {
        uint16 chainId;
        uint16 governanceChainId;
        bytes32 governanceContract;
        address WETH;
    }

    struct State {
        address payable wormhole;
        address tokenImplementation;

        Provider provider;

        // Mapping of consumed governance actions
        mapping(bytes32 => bool) consumedGovernanceActions;

        // Mapping of consumed token transfers
        mapping(bytes32 => bool) completedTransfers;

        // Mapping of initialized implementations
        mapping(address => bool) initializedImplementations;

        // Mapping of wrapped assets (chainID => nativeAddress => wrappedAddress)
        mapping(uint16 => mapping(bytes32 => address)) wrappedAssets;

        // Mapping of bridge contracts on other chains
        mapping(uint16 => bytes32) bridgeImplementations;
    }
}

contract BridgeState {
    BridgeStorage.State _state;
}