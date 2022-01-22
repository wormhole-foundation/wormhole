// SPDX-License-Identifier: Apache2
pragma solidity ^0.8.3;

import "./NFTState.sol";

contract NFTSetters is NFTState {
  function _setWormhole(address wh) internal {
    _wormholeState.wormhole = payable(wh);
  }

  function _setChainId(uint16 chainId_) internal {
    _wormholeState.chainId = chainId_;
  }

  function _setTransferCompleted(bytes32 hash) internal {
    _wormholeState.completedTransfers[hash] = true;
  }

  function _setNftContract(uint16 chainId, bytes32 nftContract) internal {
    _wormholeState.nftContracts[chainId] = nftContract;
  }
}
