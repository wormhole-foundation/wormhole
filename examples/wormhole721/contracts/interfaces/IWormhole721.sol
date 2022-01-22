// SPDX-License-Identifier: Apache2
pragma solidity ^0.8.3;

/// ERC165 interfaceId is 0x647bffff
/* is IERC165 */
interface IWormhole721 {
  function wormholeInit(uint16 chainId, address wormhole) external;

  function wormholeRegisterContract(uint16 chainId, bytes32 nftContract) external;

  function wormholeGetContract(uint16 chainId) external view returns (bytes32);

  function wormholeTransfer(
    uint256 tokenID,
    uint16 recipientChain,
    bytes32 recipient,
    uint32 nonce
  ) external payable returns (uint64 sequence);

  function wormholeCompleteTransfer(bytes memory encodedVm) external;
}
