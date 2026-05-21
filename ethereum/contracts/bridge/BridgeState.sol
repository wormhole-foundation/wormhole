// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./BridgeStructs.sol";

contract BridgeStorage {
    struct Provider {
        uint16 chainId;
        uint16 governanceChainId;
        // Required number of block confirmations to assume finality
        uint8 finality;
        bytes32 governanceContract;
        address WETH;
    }

    struct Asset {
        uint16 chainId;
        bytes32 assetAddress;
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

        // Mapping to safely identify wrapped assets
        mapping(address => bool) isWrappedAsset;

        // Mapping of native assets to amount outstanding on other chains
        mapping(address => uint256) outstandingBridged;

        // Mapping of bridge contracts on other chains
        mapping(uint16 => bytes32) bridgeImplementations;

        // EIP-155 Chain ID
        uint256 evmChainId;

        // Address authorized to call pause(). Configured via the SetPauserAddresses governance action.
        // May be address(0) (unassigned), in which case pause() reverts before comparing msg.sender.
        // See the "Pausing" section of whitepapers/0003_token_bridge.md.
        address pauser;

        // Address authorized to call unpause(). Configured via the SetPauserAddresses governance action.
        // May be address(0) (unassigned), in which case unpause() reverts before comparing msg.sender;
        // recovery then requires governance to first assign a non-zero unpauser.
        address unpauser;

        // Whether the Token Bridge is currently paused. When true, all entry points except governance
        // and unpause revert.
        bool paused;
    }
}

contract BridgeState {
    BridgeStorage.State _state;
}
