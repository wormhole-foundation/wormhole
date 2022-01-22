// SPDX-License-Identifier: Apache2
pragma solidity ^0.8.3;

contract NFTStorage {
  struct State {
    // Wormhole bridge contract address and chainId
    address payable wormhole;
    uint16 chainId;
    // Mapping of consumed token transfers
    mapping(bytes32 => bool) completedTransfers;
    // Mapping of NFT contracts on other chains
    mapping(uint16 => bytes32) nftContracts;
  }
}
