// SPDX-License-Identifier: Apache2
pragma solidity ^0.8.3;

contract NFTStructs {
  struct Transfer {
    // PayloadID uint8 = 1
    // TokenID of the token
    uint256 tokenId;
    // Address of the recipient. Left-zero-padded if shorter than 32 bytes
    bytes32 to;
    // Chain ID of the recipient
    uint16 toChain;
  }
}
