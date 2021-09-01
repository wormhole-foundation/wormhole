// contracts/Getters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

import "../interfaces/IWormhole.sol";

import "./BridgeState.sol";

contract BridgeGetters is BridgeState {
    function governanceActionIsConsumed(bytes32 hash) public view returns (bool) {
        return _state.consumedGovernanceActions[hash];
    }

    function isInitialized(address impl) public view returns (bool) {
        return _state.initializedImplementations[impl];
    }

    function isTransferCompleted(bytes32 hash) public view returns (bool) {
        return _state.completedTransfers[hash];
    }

    function wormhole() public view returns (IWormhole) {
        return IWormhole(_state.wormhole);
    }

    function chainId() public view returns (uint16){
        return _state.provider.chainId;
    }

    function governanceChainId() public view returns (uint16){
        return _state.provider.governanceChainId;
    }

    function governanceContract() public view returns (bytes32){
        return _state.provider.governanceContract;
    }

    function wrappedAsset(uint16 tokenChainId, bytes32 tokenAddress) public view returns (address){
        return _state.wrappedAssets[tokenChainId][tokenAddress];
    }

    function bridgeContracts(uint16 chainId_) public view returns (bytes32){
        return _state.bridgeImplementations[chainId_];
    }

    function tokenImplementation() public view returns (address){
        return _state.tokenImplementation;
    }

    function WETH() public view returns (IWETH){
        return IWETH(_state.provider.WETH);
    }

    function outstandingBridged(address token) public view returns (uint256){
        return _state.outstandingBridged[token];
    }

    function isWrappedAsset(address token) public view returns (bool){
        return _state.isWrappedAsset[token];
    }
}

interface IWETH is IERC20 {
    function deposit() external payable;

    function withdraw(uint amount) external;
}