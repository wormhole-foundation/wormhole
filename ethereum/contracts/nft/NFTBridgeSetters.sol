// contracts/Setters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./NFTBridgeState.sol";

contract NFTBridgeSetters is NFTBridgeState {
    function setInitialized(address implementatiom) internal {
        _state.initializedImplementations[implementatiom] = true;
    }

    function setGovernanceActionConsumed(bytes32 hash) internal {
        _state.consumedGovernanceActions[hash] = true;
    }

    function setTransferCompleted(bytes32 hash) internal {
        _state.completedTransfers[hash] = true;
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

    function setBridgeImplementation(uint16 chainId, bytes32 bridgeContract) internal {
        _state.bridgeImplementations[chainId] = bridgeContract;
    }

    function setTokenImplementation(address impl) internal {
        _state.tokenImplementation = impl;
    }

    function setWormhole(address wh) internal {
        _state.wormhole = payable(wh);
    }

    function setWrappedAsset(uint16 tokenChainId, bytes32 tokenAddress, address wrapper) internal {
        _state.wrappedAssets[tokenChainId][tokenAddress] = wrapper;
        _state.isWrappedAsset[wrapper] = true;
    }

    function setSplCache(uint256 tokenId, NFTBridgeStorage.SPLCache memory cache) internal {
        _state.splCache[tokenId] = cache;
    }

    function clearSplCache(uint256 tokenId) internal {
        delete _state.splCache[tokenId];
    }
}