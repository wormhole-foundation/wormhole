// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./NFTBridgeStructs.sol";

contract NFTBridgeStorage {
    struct Provider {
        uint16 chainId;
        uint16 governanceChainId;
        bytes32 governanceContract;
    }

    struct Asset {
        uint16 chainId;
        bytes32 assetAddress;
    }

    struct SPLCache {
        bytes32 name;
        bytes32 symbol;
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

        // Mapping of bridge contracts on other chains
        mapping(uint16 => bytes32) bridgeImplementations;

        // Mapping of spl token info caches (chainID => nativeAddress => SPLCache)
        mapping(uint256 => SPLCache) splCache;

        // Required number of block confirmations to assume finality
        uint8 finality;

        // These 248 bits (31 bytes) are unused, and we reserve them for future
        // state variables that can be packed into the same word slot as
        // 'finality' above. Anything smaller than a full word can be allocated
        // here to save gas
        uint248 unused;
    }
}

contract NFTBridgeState {
    NFTBridgeStorage.State _state;
}
