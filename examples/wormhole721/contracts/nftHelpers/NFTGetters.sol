// SPDX-License-Identifier: Apache2
pragma solidity ^0.8.3;

import "../interfaces/IWormhole.sol";
import "./NFTState.sol";

contract NFTGetters is NFTState {
  function isTransferCompleted(bytes32 hash) public view returns (bool) {
    return _wormholeState.completedTransfers[hash];
  }

  function nftContract(uint16 chainId_) public view returns (bytes32) {
    return _wormholeState.nftContracts[chainId_];
  }

  function wormhole() public view returns (IWormhole) {
    return IWormhole(_wormholeState.wormhole);
  }

  function chainId() public view returns (uint16) {
    return _wormholeState.chainId;
  }
}
